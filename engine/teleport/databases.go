package teleport

import (
	"context"
	"fmt"
	"net"
	"strings"

	apiclient "github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/profile"
	"github.com/gravitational/teleport/api/types"
)

// DatabaseInfo is a trimmed view of a Teleport database resource.
type DatabaseInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Protocol    string `json:"protocol"`
	URI         string `json:"uri"`
}

// ListDatabases returns all databases accessible via the current tsh profile.
// It connects to the Teleport proxy using the credentials already on disk
// (written by `tsh login`) — no browser or password prompt needed.
func ListDatabases(ctx context.Context, profileName string) ([]DatabaseInfo, error) {
	dir := defaultTSHDir()
	p, err := profile.FromDir(dir, profileName)
	if err != nil {
		p, err = profile.FromDir(dir, "")
		if err != nil {
			return nil, fmt.Errorf("could not load tsh profile (run `tsh login` first): %w", err)
		}
	}

	proxyHost := p.WebProxyAddr
	if h, _, splitErr := net.SplitHostPort(proxyHost); splitErr == nil {
		proxyHost = h
	}
	// Teleport web proxy listens on port 443 (or 3080 for self-hosted).
	proxyAddr := proxyHost + ":443"

	clt, err := apiclient.New(ctx, apiclient.Config{
		Addrs:       []string{proxyAddr},
		Credentials: []apiclient.Credentials{apiclient.LoadProfile(dir, p.Name())},
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to Teleport proxy: %w", err)
	}
	defer clt.Close()

	dbs, err := clt.GetDatabases(ctx)
	if err != nil {
		// Fallback: try the standard auth port.
		clt2, err2 := apiclient.New(ctx, apiclient.Config{
			Addrs:       []string{proxyHost + ":3025"},
			Credentials: []apiclient.Credentials{apiclient.LoadProfile(dir, p.Name())},
		})
		if err2 != nil {
			return nil, fmt.Errorf("could not list databases: %w", err)
		}
		defer clt2.Close()
		dbs, err = clt2.GetDatabases(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not list databases: %w", err)
		}
	}

	return dbsToInfo(dbs), nil
}

func dbsToInfo(dbs []types.Database) []DatabaseInfo {
	out := make([]DatabaseInfo, 0, len(dbs))
	for _, d := range dbs {
		out = append(out, DatabaseInfo{
			Name:        d.GetName(),
			Description: strings.TrimSpace(d.GetDescription()),
			Protocol:    d.GetProtocol(),
			URI:         d.GetURI(),
		})
	}
	return out
}
