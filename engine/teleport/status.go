package teleport

import (
	"net"
	"strings"
	"time"

	"github.com/gravitational/teleport/api/profile"
)

// TSHStatus is a lightweight equivalent of `tsh status` suitable for UI checks.
// It uses Teleport's Go profile helpers (no subprocess) and inspects the local
// X509 cert expiry to determine if the user is currently logged in.
type TSHStatus struct {
	TSHAvailable bool   `json:"tsh_available"`
	TSHPath      string `json:"tsh_path,omitempty"`

	LoggedIn bool   `json:"logged_in"`
	Cluster  string `json:"cluster,omitempty"`
	User     string `json:"user,omitempty"`
	Profile  string `json:"profile,omitempty"`
	Proxy    string `json:"proxy,omitempty"`

	ExpiresAt string `json:"expires_at,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Status inspects the local tsh profile and cached credentials to determine
// whether the user is logged in. If profileName is empty, it checks the current
// active profile.
func Status(profileName string) TSHStatus {
	var st TSHStatus

	tshPath, err := resolveTSH()
	if err != nil {
		st.TSHAvailable = false
		st.LoggedIn = false
		st.Error = err.Error()
		return st
	}
	st.TSHAvailable = true
	st.TSHPath = tshPath

	dir := defaultTSHDir()
	p, err := profile.FromDir(dir, strings.TrimSpace(profileName))
	if err != nil && strings.TrimSpace(profileName) != "" {
		// The stored profile may be a display label (e.g. "StockbitTeleport")
		// rather than a real proxy-host profile filename. Fall back to the
		// current active profile so a valid `tsh login` session is still seen.
		p, err = profile.FromDir(dir, "")
	}
	if err != nil {
		st.LoggedIn = false
		st.Error = err.Error()
		return st
	}

	proxyHost := strings.TrimSpace(p.WebProxyAddr)
	if h, _, splitErr := net.SplitHostPort(proxyHost); splitErr == nil {
		proxyHost = h
	}

	st.Cluster = strings.TrimSpace(p.SiteName)
	st.User = strings.TrimSpace(p.Username)
	st.Profile = strings.TrimSpace(p.Name())
	st.Proxy = proxyHost

	exp, ok := p.Expiry()
	if ok {
		st.ExpiresAt = exp.UTC().Format(time.RFC3339)
		// Treat very-soon-to-expire creds as not logged in to avoid hangs in long operations.
		if time.Until(exp) > 30*time.Second {
			st.LoggedIn = true
			return st
		}
	}

	st.LoggedIn = false
	if st.Error == "" {
		st.Error = "tsh profile is present but credentials are missing or expired (run `tsh login`)"
	}
	return st
}

