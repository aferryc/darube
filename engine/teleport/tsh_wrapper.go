package teleport

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gravitational/teleport/api/profile"
	"github.com/gravitational/teleport/api/utils/keypaths"

	"engine/sslutil"
	"engine/store"
)

// tshFallbackPaths are checked in order when tsh is not on PATH.
// GUI/app launchers strip PATH, so we probe common install locations.
var tshFallbackPaths = []string{
	"/usr/local/bin/tsh",
	"/opt/homebrew/bin/tsh",
	"/usr/bin/tsh",
}

func resolveTSH() (string, error) {
	if p, err := exec.LookPath("tsh"); err == nil {
		return p, nil
	}
	for _, p := range tshFallbackPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("tsh not found on PATH or in %v", tshFallbackPaths)
}

func defaultTSHDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tsh")
}

// TSHProfile holds cluster, user, and profile parsed from the tsh profile on disk.
type TSHProfile struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
	Profile string `json:"profile"`
}

// DetectTSHProfile reads the active tsh profile from disk via the Go API.
func DetectTSHProfile() (TSHProfile, error) {
	dir := defaultTSHDir()
	p, err := profile.FromDir(dir, "")
	if err != nil {
		return TSHProfile{}, fmt.Errorf("could not load tsh profile (run `tsh login` first): %w", err)
	}
	if p.SiteName == "" && p.Username == "" {
		return TSHProfile{}, fmt.Errorf("tsh profile is empty — run `tsh login` first")
	}
	cluster := p.SiteName
	if cluster == "" {
		cluster = "default"
	}
	return TSHProfile{Cluster: cluster, User: p.Username, Profile: p.Name()}, nil
}

// resolvedProfile holds the coordinates needed to build keypaths.
type resolvedProfile struct {
	Dir       string
	ProxyHost string // hostname only, no port
	Username  string
	Cluster   string
}

// loadResolvedProfile reads the tsh profile via the Go API and merges any
// per-connection overrides from cfg.
func loadResolvedProfile(cfg store.ConnectionConfig) (resolvedProfile, error) {
	dir := defaultTSHDir()
	// Try the named profile first; fall back to the current active profile.
	// The stored profile name may be a cluster name rather than a proxy hostname,
	// in which case FromDir will fail and we recover gracefully.
	p, err := profile.FromDir(dir, cfg.TeleportProfile)
	if err != nil {
		p, err = profile.FromDir(dir, "")
		if err != nil {
			return resolvedProfile{}, fmt.Errorf("could not load tsh profile (run `tsh login` first): %w", err)
		}
	}

	// WebProxyAddr may include a port; we only need the hostname for keypaths.
	proxyHost := p.WebProxyAddr
	if h, _, splitErr := net.SplitHostPort(proxyHost); splitErr == nil {
		proxyHost = h
	}

	cluster := cfg.TeleportCluster
	if cluster == "" {
		cluster = p.SiteName
	}
	username := cfg.TeleportUser
	if username == "" {
		username = p.Username
	}

	return resolvedProfile{
		Dir:       dir,
		ProxyHost: proxyHost,
		Username:  username,
		Cluster:   cluster,
	}, nil
}

// dbEnvInfo holds the complete connection parameters parsed from `tsh db env`
// or derived from keypaths + the profile. It covers both TLS-cert mode and
// proxy-tunnel mode.
type dbEnvInfo struct {
	// Connection target (overrides whatever the user typed in the form).
	Host string
	Port string
	User string
	Pass string
	DB   string

	// TLS material (empty in proxy-tunnel mode).
	SSLRootCert string
	SSLCert     string
	SSLKey      string
}

func (e dbEnvInfo) hasTLS() bool  { return e.SSLCert != "" || e.SSLRootCert != "" }
func (e dbEnvInfo) hasHost() bool { return e.Host != "" || e.Port != "" }

// runTSHDBEnv executes `tsh db env [--cluster=<c>] <service>` and parses its
// stdout into a dbEnvInfo. It handles both shell-export style (PGSSLCERT=...)
// and key=value style (ca_file=...).
func runTSHDBEnv(tshPath string, cfg store.ConnectionConfig) (dbEnvInfo, error) {
	args := []string{"db", "env"}
	if cfg.TeleportCluster != "" {
		args = append(args, "--cluster="+cfg.TeleportCluster)
	}
	args = append(args, cfg.TeleportDBService)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(tshPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return dbEnvInfo{}, fmt.Errorf("tsh db env failed (run `tsh db login %s` first): %s",
			cfg.TeleportDBService, msg)
	}

	return parseDBEnv(stdout.String()), nil
}

// parseDBEnv extracts connection parameters from `tsh db env` output.
// Handles both PG env-var style and key=value (tsh db config) style.
func parseDBEnv(out string) dbEnvInfo {
	kvRe := regexp.MustCompile(`(?m)(?:export\s+)?(\w+)=([\S]+)`)
	m := map[string]string{}
	for _, match := range kvRe.FindAllStringSubmatch(out, -1) {
		m[match[1]] = strings.Trim(match[2], `"'`)
	}

	pick := func(keys ...string) string {
		for _, k := range keys {
			if v := m[k]; v != "" {
				return v
			}
		}
		return ""
	}

	return dbEnvInfo{
		Host:        pick("PGHOST", "MYSQL_HOST"),
		Port:        pick("PGPORT", "MYSQL_TCP_PORT", "MYSQL_UNIX_PORT"),
		User:        pick("PGUSER", "MYSQL_USER"),
		Pass:        pick("PGPASSWORD", "MYSQL_PWD"),
		DB:          pick("PGDATABASE", "MYSQL_DATABASE"),
		SSLRootCert: pick("PGSSLROOTCERT", "SSL_CA_FILE", "ca_file"),
		SSLCert:     pick("PGSSLCERT", "SSL_CERT_FILE", "cert_file"),
		SSLKey:      pick("PGSSLKEY", "SSL_KEY_FILE", "key_file"),
	}
}

// enrichWithKeypaths fills in any missing TLS cert paths using the Go API
// keypaths, so we can avoid a subprocess when the cert files already exist.
func enrichWithKeypaths(env *dbEnvInfo, rp resolvedProfile, dbService string) {
	if env.hasTLS() {
		return // already have certs from tsh db env output
	}
	cert := keypaths.DatabaseCertPath(rp.Dir, rp.ProxyHost, rp.Username, rp.Cluster, dbService)
	key := keypaths.DatabaseKeyPath(rp.Dir, rp.ProxyHost, rp.Username, rp.Cluster, dbService)
	ca := keypaths.TLSCAsPathCluster(rp.Dir, rp.ProxyHost, rp.Cluster)

	if _, err := os.Stat(cert); err != nil {
		return
	}
	if _, err := os.Stat(key); err != nil {
		return
	}
	env.SSLCert = cert
	env.SSLKey = key
	if _, err := os.Stat(ca); err == nil {
		env.SSLRootCert = ca
	}
	// When reading certs from keypaths (no tsh db env), the proxy host is the
	// connection target. Override only if the env doesn't already have a host.
	if env.Host == "" {
		env.Host = rp.ProxyHost
	}
}

// proxyTunnelInfo holds the address/credentials from a `tsh proxy db --tunnel` process.
type proxyTunnelInfo struct {
	Host     string
	Port     string
	User     string
	Password string
}

// startDBProxy launches `tsh proxy db [--tunnel] <service>` and waits
// for it to print a listening address. The returned CancelFunc must be called
// to stop the proxy when the connection is done.
func startDBProxy(ctx context.Context, tshPath string, cfg store.ConnectionConfig, tunnel bool) (proxyTunnelInfo, context.CancelFunc, error) {
	args := []string{"proxy", "db"}
	if cfg.TeleportCluster != "" {
		args = append(args, "--cluster="+cfg.TeleportCluster)
	}
	if cfg.User != "" {
		args = append(args, "--db-user="+cfg.User)
	}
	// For PostgreSQL, db-name is typically required. If empty, tsh may fall back to
	// an active database cert; we still pass it when provided.
	if cfg.DBName != "" {
		args = append(args, "--db-name="+cfg.DBName)
	}
	if tunnel {
		args = append(args, "--tunnel")
	}
	args = append(args, cfg.TeleportDBService)

	proxyCtx, cancelCtx := context.WithCancel(ctx)
	cmd := exec.Command(tshPath, args...)
	// Start tsh in its own process group so we can reliably terminate it (and any
	// children) when the connection is closed or times out.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancelCtx()
		return proxyTunnelInfo{}, nil, fmt.Errorf("could not pipe tsh proxy db stdout: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancelCtx()
		return proxyTunnelInfo{}, nil, fmt.Errorf("could not start tsh proxy db: %w", err)
	}

	var cancelOnce sync.Once
	cancel := func() {
		cancelOnce.Do(func() {
			cancelCtx()
			if cmd.Process == nil {
				return
			}
			// Try graceful shutdown first.
			pgid, _ := syscall.Getpgid(cmd.Process.Pid)
			if pgid <= 0 {
				pgid = cmd.Process.Pid
			}
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
			time.AfterFunc(500*time.Millisecond, func() {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
				_ = cmd.Process.Kill()
			})
		})
	}

	// Also cancel if the provided ctx is done.
	go func() {
		<-proxyCtx.Done()
		cancel()
	}()

	addrCh := make(chan proxyTunnelInfo, 1)
	errCh := make(chan error, 1)

	go func() {
		var info proxyTunnelInfo
		found := false
		// Examples:
		// - "Started DB proxy on 127.0.0.1:61740"
		// - "Started authenticated tunnel ... on 127.0.0.1:49415."
		// - "Started DB proxy on localhost:61740"
		addrRe := regexp.MustCompile(`(?i)(127\.0\.0\.1|localhost|\[::1\]):(\d+)`)
		userRe := regexp.MustCompile(`(?i)(?:username|user)[:\s]+(\S+)`)
		passRe := regexp.MustCompile(`(?i)(?:password|pass)[:\s]+(\S+)`)

		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if !found {
				if m := addrRe.FindStringSubmatch(line); len(m) > 2 {
					info.Host = m[1]
					if info.Host == "[::1]" {
						info.Host = "127.0.0.1"
					} else if strings.EqualFold(info.Host, "localhost") {
						info.Host = "127.0.0.1"
					}
					info.Port = m[2]
				}
				if m := userRe.FindStringSubmatch(line); len(m) > 1 {
					info.User = m[1]
				}
				if m := passRe.FindStringSubmatch(line); len(m) > 1 {
					info.Password = m[1]
				}
				if info.Port != "" {
					found = true
					// Give it a brief moment to print credentials on subsequent lines.
					time.AfterFunc(300*time.Millisecond, func() { addrCh <- info })
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			if !found {
				errCh <- fmt.Errorf("tsh proxy db exited before printing a listening address")
			}
		}
	}()

	select {
	case info := <-addrCh:
		if info.Host == "" {
			info.Host = "127.0.0.1"
		}
		return info, cancel, nil
	case err := <-errCh:
		cancel()
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return proxyTunnelInfo{}, nil, fmt.Errorf("tsh proxy db: %w: %s", err, msg)
		}
		return proxyTunnelInfo{}, nil, fmt.Errorf("tsh proxy db: %w", err)
	case <-time.After(15 * time.Second):
		cancel()
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return proxyTunnelInfo{}, nil, fmt.Errorf("tsh proxy db did not print a listening address within 15 s: %s", msg)
		}
		return proxyTunnelInfo{}, nil, fmt.Errorf("tsh proxy db did not print a listening address within 15 s")
	}
}

// EnsureAndBuildDSN is the main entry point for Teleport-aware connection setup.
//
// Flow:
//  1. Prefer `tsh proxy db --tunnel` so GUI clients can connect through a local port
//     (required in TLS routing mode for many drivers).
//  2. If tunnel mode fails, fall back to injecting certs into the DSN using
//     `tsh db env` or keypaths (direct mode).
func EnsureAndBuildDSN(cfg store.ConnectionConfig, baseDSN string) (string, context.CancelFunc, error) {
	if !cfg.TeleportEnabled {
		return baseDSN, nil, nil
	}
	if cfg.TeleportDBService == "" {
		return "", nil, fmt.Errorf("teleport: teleport_db_service must be set")
	}

	tshPath, err := resolveTSH()
	if err != nil {
		return "", nil, fmt.Errorf("teleport: %w", err)
	}

	// Step 1: prefer an authenticated local tunnel. This is the most compatible
	// way to support 3rd party/GUI database clients with TLS routing.
	if info, cancel, err := startDBProxy(context.Background(), tshPath, cfg, true); err == nil {
		dsn, dsnErr := applyProxyTunnel(baseDSN, cfg.DBType, info)
		if dsnErr != nil {
			cancel()
			return "", nil, dsnErr
		}
		return dsn, cancel, nil
	}

	// Step 2: try `tsh db env` for the authoritative connection parameters.
	env, envErr := runTSHDBEnv(tshPath, cfg)
	if envErr != nil {
		// Step 3: fall back to reading cert files from the well-known keypaths.
		rp, rpErr := loadResolvedProfile(cfg)
		if rpErr != nil {
			return "", nil, fmt.Errorf("teleport: %w — and tsh db env failed: %w", rpErr, envErr)
		}
		enrichWithKeypaths(&env, rp, cfg.TeleportDBService)
	}

	if env.hasTLS() {
		dsn, err := applyTLSEnv(baseDSN, cfg.DBType, env)
		return dsn, nil, err
	}

	return "", nil, fmt.Errorf("teleport: could not determine TLS material; run `tsh db login %s` first", cfg.TeleportDBService)
}

// applyTLSEnv rewrites the DSN with connection parameters from `tsh db env`.
// It overrides host/port (so the connection goes to the Teleport proxy, not
// whatever the user typed) and injects TLS cert paths.
func applyTLSEnv(dsn, dbType string, env dbEnvInfo) (string, error) {
	switch dbType {
	case "postgres", "postgresql":
		stripSSL := regexp.MustCompile(`\s*ssl\w*=[^\s]*`)
		dsn = strings.TrimSpace(stripSSL.ReplaceAllString(dsn, ""))

		if env.Host != "" {
			dsn = overridePGKV(dsn, "host", env.Host)
		}
		if env.Port != "" {
			dsn = overridePGKV(dsn, "port", env.Port)
		}
		if env.User != "" {
			dsn = overridePGKV(dsn, "user", env.User)
		}
		if env.DB != "" {
			dsn = overridePGKV(dsn, "dbname", env.DB)
		}

		dsn += " sslmode=verify-full"
		if env.SSLRootCert != "" {
			dsn += " sslrootcert=" + env.SSLRootCert
		}
		dsn += " sslcert=" + env.SSLCert + " sslkey=" + env.SSLKey
		return dsn, nil

	case "mysql", "mariadb":
		if env.Host != "" || env.Port != "" {
			hostRe := regexp.MustCompile(`tcp\(([^)]+)\)`)
			if m := hostRe.FindStringSubmatch(dsn); len(m) > 1 {
				parts := strings.SplitN(m[1], ":", 2)
				h, p := parts[0], ""
				if len(parts) == 2 {
					p = parts[1]
				}
				if env.Host != "" {
					h = env.Host
				}
				if env.Port != "" {
					p = env.Port
				}
				dsn = hostRe.ReplaceAllString(dsn, "tcp("+h+":"+p+")")
			}
		}
		sum := sha1.Sum([]byte(env.SSLRootCert + "|" + env.SSLCert + "|" + env.SSLKey))
		tlsName := "teleport_tls_" + hex.EncodeToString(sum[:])[:10]
		ca := ""
		if _, statErr := os.Stat(env.SSLRootCert); statErr == nil {
			ca = env.SSLRootCert
		}
		if err := sslutil.RegisterMySQLTLSConfig(tlsName, ca, env.SSLCert, env.SSLKey); err != nil {
			return "", fmt.Errorf("teleport: could not register MySQL TLS config: %w", err)
		}
		tlsRe := regexp.MustCompile(`([?&])tls=[^&]*`)
		if tlsRe.MatchString(dsn) {
			return tlsRe.ReplaceAllString(dsn, "${1}tls="+tlsName), nil
		} else if strings.Contains(dsn, "?") {
			return dsn + "&tls=" + tlsName, nil
		}
		return dsn + "?tls=" + tlsName, nil

	default:
		return dsn, nil
	}
}

// applyProxyTunnel redirects the DSN to the local tsh proxy listener.
func applyProxyTunnel(dsn, dbType string, info proxyTunnelInfo) (string, error) {
	switch dbType {
	case "postgres", "postgresql":
		stripSSL := regexp.MustCompile(`\s*ssl\w*=[^\s]*`)
		dsn = strings.TrimSpace(stripSSL.ReplaceAllString(dsn, ""))
		dsn = overridePGKV(dsn, "host", info.Host)
		dsn = overridePGKV(dsn, "port", info.Port)
		// In authenticated tunnel mode, Teleport ignores provided password.
		// We keep any existing user/password unless tsh explicitly prints overrides.
		if info.User != "" {
			dsn = overridePGKV(dsn, "user", info.User)
		}
		if info.Password != "" {
			dsn = overridePGKV(dsn, "password", info.Password)
		}
		dsn += " sslmode=disable"
		return dsn, nil

	case "mysql", "mariadb":
		hostRe := regexp.MustCompile(`tcp\(([^)]+)\)`)
		if m := hostRe.FindStringSubmatch(dsn); len(m) > 1 {
			dsn = hostRe.ReplaceAllString(dsn, "tcp("+info.Host+":"+info.Port+")")
		}
		// Authenticated tunnel terminates TLS locally; disable client-side TLS.
		tlsRe := regexp.MustCompile(`([?&])tls=[^&]*`)
		if tlsRe.MatchString(dsn) {
			dsn = tlsRe.ReplaceAllString(dsn, "${1}tls=false")
		} else if strings.Contains(dsn, "?") {
			dsn = dsn + "&tls=false"
		} else {
			dsn = dsn + "?tls=false"
		}
		return dsn, nil

	case "sqlserver", "mssql":
		var buf bytes.Buffer
		buf.WriteString("sqlserver://")
		if info.User != "" {
			buf.WriteString(info.User)
			if info.Password != "" {
				buf.WriteString(":")
				buf.WriteString(info.Password)
			}
			buf.WriteString("@")
		}
		buf.WriteString(info.Host)
		buf.WriteString(":")
		buf.WriteString(info.Port)
		if m := regexp.MustCompile(`[?&]database=([^&]+)`).FindStringSubmatch(dsn); len(m) > 1 {
			buf.WriteString("?database=")
			buf.WriteString(m[1])
			buf.WriteString("&trustservercertificate=true")
		} else {
			buf.WriteString("?trustservercertificate=true")
		}
		return buf.String(), nil

	default:
		return dsn, nil
	}
}

func overridePGKV(dsn, key, val string) string {
	if val == "" {
		return dsn
	}
	// Handle both `key=value` and `key=` (empty) forms.
	re := regexp.MustCompile(`(?i)\b` + key + `=\S*`)
	if re.MatchString(dsn) {
		return re.ReplaceAllString(dsn, key+"="+val)
	}
	return strings.TrimSpace(dsn) + " " + key + "=" + val
}
