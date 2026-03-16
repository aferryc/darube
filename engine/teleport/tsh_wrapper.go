package teleport

import (
	"bytes"
	"fmt"
	"os/exec"

	"engine/store"
)

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

