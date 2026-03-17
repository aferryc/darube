package teleport

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"engine/store"
)

// TSHProfile holds cluster, user, and profile parsed from tsh status.
type TSHProfile struct {
	Cluster string `json:"cluster"`
	User    string `json:"user"`
	Profile string `json:"profile"`
}

// DetectTSHProfile runs tsh status and parses the current profile (cluster, user).
// Profile is set to the cluster name by default. Parses stdout even if tsh exits
// with an error (e.g. expired cert), so we can still return partial profile info.
func DetectTSHProfile() (TSHProfile, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("tsh", "status")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	out := stdout.String()

	var cluster, user string
	clusterRe := regexp.MustCompile(`(?m)^\s*Cluster:\s+(.+)$`)
	userRe := regexp.MustCompile(`(?m)^\s*Logged in as:\s+(.+)$`)
	if m := clusterRe.FindStringSubmatch(out); len(m) > 1 {
		cluster = strings.TrimSpace(m[1])
	}
	if m := userRe.FindStringSubmatch(out); len(m) > 1 {
		user = strings.TrimSpace(m[1])
	}
	if cluster == "" && user == "" {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "could not parse tsh status output (run tsh login first)"
		}
		return TSHProfile{}, fmt.Errorf("%s", msg)
	}
	profile := cluster
	if profile == "" {
		profile = "default"
	}
	return TSHProfile{Cluster: cluster, User: user, Profile: profile}, nil
}

// EnsureAndBuildDSN is a minimal shim around the tsh CLI. For now it simply
// verifies that tsh is available on the PATH and defers to the normal DB
// connection logic by returning an empty DSN override. This keeps the surface
// small so we can evolve it into a full tsh db login / tunnel helper later.
//
// If tsh is missing, it returns a descriptive error.
func EnsureAndBuildDSN(cfg store.ConnectionConfig, baseDSN string) (string, error) {
	if !cfg.TeleportEnabled {
		return baseDSN, nil
	}

	var stderr bytes.Buffer
	cmd := exec.Command("tsh", "version")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("teleport: tsh CLI is required for this connection but could not be executed: %s", msg)
	}

	// For now, we just ensure tsh exists. In a follow-up we can call tsh db
	// login / tsh db connect here and construct a DSN pointing at the local
	// Teleport-provided listener.
	return baseDSN, nil
}

