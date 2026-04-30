package teleport

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// LoginResult is returned by Login.
type LoginResult struct {
	// Output is everything tsh printed (useful for debugging SSO/MFA flows).
	Output string `json:"output"`
}

// Login runs `tsh login --proxy=<host>` and waits for it to complete.
// For browser-based SSO, tsh will open the system browser automatically.
// The call blocks until tsh exits or ctx is cancelled.
func Login(ctx context.Context, proxyHost string) (LoginResult, error) {
	if proxyHost == "" {
		return LoginResult{}, fmt.Errorf("proxy host is required")
	}

	tshPath, err := resolveTSH()
	if err != nil {
		return LoginResult{}, fmt.Errorf("teleport: %w", err)
	}

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, tshPath, "login", "--proxy="+proxyHost)
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(out.String())
		if msg == "" {
			msg = err.Error()
		}
		return LoginResult{Output: msg}, fmt.Errorf("tsh login failed: %s", msg)
	}

	return LoginResult{Output: strings.TrimSpace(out.String())}, nil
}
