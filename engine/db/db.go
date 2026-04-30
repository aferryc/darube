package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"engine/store"
	"engine/teleport"
)

// Connect builds the data source name (DSN) from the ConnectionConfig
// and attempts to open and verify the connection.
func Connect(config store.ConnectionConfig) (*sql.DB, func(), error) {
	// Map human readable type to standard Go driver names
	driverName := config.DBType
	switch config.DBType {
	case "postgres", "postgresql":
		driverName = "postgres"
	case "mysql", "mariadb":
		driverName = "mysql"
	case "sqlserver", "mssql":
		driverName = "sqlserver"
	case "sqlite", "sqlite3":
		driverName = "sqlite"
	case "oracle":
		driverName = "oracle"
	}

	dsn, err := buildDSN(config, driverName)
	if err != nil {
		return nil, nil, err
	}

	// Resolve Teleport profile from settings if TeleportProfileID is set.
	cfg := config
	store.ResolveTeleportConfig(&cfg)

	// If Teleport is enabled for this connection, ensure tsh is available and
	// let the wrapper adjust or validate the DSN as needed.
	dsn, cleanup, err := teleport.EnsureAndBuildDSN(cfg, dsn)
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, fmt.Errorf("failed to open connection: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err = db.PingContext(pingCtx); err != nil {
		db.Close()
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, cleanup, nil
}

func buildDSN(c store.ConnectionConfig, driverName string) (string, error) {
	switch driverName {
	case "postgres":
		// Format: postgres://username:password@localhost:5432/database_name
		// We can also use keyword format: host=... port=... user=... password=... sslmode=...
		sslMode := "disable"
		if c.EnableSSL {
			sslMode = "require" // Default requirement if SSL is enabled
		}

		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, sslMode)
		if c.DBName != "" {
			dsn += fmt.Sprintf(" dbname=%s", c.DBName)
		}

		if c.EnableSSL && (c.CACertPath != "" || c.ClientCertPath != "" || c.ClientKeyPath != "") {
			if c.CACertPath != "" {
				dsn += fmt.Sprintf(" sslrootcert=%s", c.CACertPath)
				if sslMode == "require" {
					// Prefer verify-ca for broader compatibility (e.g. IP hosts).
					dsn = strings.Replace(dsn, "sslmode=require", "sslmode=verify-ca", 1)
				}
			}
			if c.ClientCertPath != "" {
				dsn += fmt.Sprintf(" sslcert=%s", c.ClientCertPath)
			}
			if c.ClientKeyPath != "" {
				dsn += fmt.Sprintf(" sslkey=%s", c.ClientKeyPath)
			}
		}

		return dsn, nil

	case "mysql":
		// Format: username:password@tcp(localhost:3306)/dbname
		sslMode := "false"
		if c.EnableSSL {
			// Align with Postgres sslmode=require: encrypted, but not necessarily verified.
			// If the user provides a CA/certs, we'll register a named TLS config which
			// will validate the chain (and optionally client certs).
			sslMode = "skip-verify"
			if c.CACertPath != "" || c.ClientCertPath != "" || c.ClientKeyPath != "" {
				tlsConfigName := fmt.Sprintf("customTLS_%s", c.ID)
				err := registerMySQLTLSConfig(tlsConfigName, c.CACertPath, c.ClientCertPath, c.ClientKeyPath)
				if err == nil {
					sslMode = tlsConfigName
				} else {
					sslMode = "skip-verify"
				}
			}
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
			c.User, c.Password, c.Host, c.Port, c.DBName, sslMode)
		return dsn, nil

	case "sqlserver":
		// Format: sqlserver://username:password@host:port?database=...&encrypt=...
		u := &url.URL{
			Scheme: "sqlserver",
			User:   url.UserPassword(c.User, c.Password),
			Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		}
		q := u.Query()
		if c.DBName != "" {
			q.Add("database", c.DBName)
		}
		if c.EnableSSL {
			q.Add("encrypt", "true")
			// For parity with Postgres sslmode=require and MySQL tls=skip-verify:
			// encrypt the connection, but don't require a verifiable server cert.
			q.Add("trustservercertificate", "true")
		} else {
			// Teleport proxies usually terminate TLS themselves. Setting encrypt=disable tells the
			// go-mssqldb driver to not attempt a TLS upgrade over the ALPN tunnel, but we actually
			// need to pass trustservercertificate=true to bypass the prelogin handshake gracefully.
			q.Add("trustservercertificate", "true")
		}
		u.RawQuery = q.Encode()
		return u.String(), nil

	case "sqlite":
		// modernc.org/sqlite DSN is typically a filepath, ":memory:", or "file:...".
		// We store it in file_path, but also accept host for backwards/experimental usage.
		if c.FilePath != "" {
			return c.FilePath, nil
		}
		if c.Host != "" {
			return c.Host, nil
		}
		return "", fmt.Errorf("sqlite: file_path is required")

	case "oracle":
		// go-ora DSN as URL: oracle://user:pass@host:port/service_name
		// We store service name in dbname.
		if c.Host == "" || c.Port == 0 || c.User == "" || c.DBName == "" {
			return "", fmt.Errorf("oracle: host, port, user, and dbname(service) are required")
		}
		u := &url.URL{
			Scheme: "oracle",
			User:   url.UserPassword(c.User, c.Password),
			Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
			Path:   "/" + c.DBName,
		}
		return u.String(), nil

	default:
		return "", fmt.Errorf("unsupported database type: %s", driverName)
	}
}
