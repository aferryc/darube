package teleport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// LoginResult is returned by Login.
type LoginResult struct {
	// Output is everything tsh printed (useful for debugging SSO/MFA flows).
	Output string `json:"output"`
}

// LoginOptions describes how to run `tsh login`.
//
// When Password is set, the login is driven non-interactively: tsh is run under
// a pseudo-terminal (tsh refuses password login without a real terminal) and
// the password (and OTP, if provided) are typed in response to its prompts.
// When Password is empty, tsh runs normally (e.g. browser SSO) using the
// current profile.
type LoginOptions struct {
	Proxy    string // proxy host; empty reuses the current profile on disk
	User     string // Teleport user; empty defaults to the profile/local user
	Password string // local-auth password; empty means interactive/SSO
	OTP      string // one-time code for OTP second factor
}

// loginTimeout bounds a single login attempt so a wrong password / missing
// prompt can't hang the request forever.
const loginTimeout = 90 * time.Second

// Login runs `tsh login` and waits for it to complete. The call blocks until
// tsh exits, the timeout elapses, or ctx is cancelled.
func Login(ctx context.Context, opts LoginOptions) (LoginResult, error) {
	tshPath, err := resolveTSH()
	if err != nil {
		return LoginResult{}, fmt.Errorf("teleport: %w", err)
	}

	args := []string{"login"}
	if p := strings.TrimSpace(opts.Proxy); p != "" {
		args = append(args, "--proxy="+p)
	}
	if u := strings.TrimSpace(opts.User); u != "" {
		args = append(args, "--user="+u)
	}
	// Force OTP so the second factor is a code we can type rather than a
	// hardware/webauthn assertion that needs a real device interaction.
	if strings.TrimSpace(opts.OTP) != "" {
		args = append(args, "--mfa-mode=otp")
	}

	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, tshPath, args...)

	// No credentials to type → run tsh normally (browser SSO, cached session).
	if opts.Password == "" {
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if runErr := cmd.Run(); runErr != nil {
			return LoginResult{Output: sanitizeOutput(out.String(), opts)}, loginError(out.String(), runErr)
		}
		return LoginResult{Output: sanitizeOutput(out.String(), opts)}, nil
	}

	// Password login: tsh insists on a real terminal, so run it under a PTY and
	// answer its prompts as they appear.
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return LoginResult{}, fmt.Errorf("teleport: could not allocate a terminal for tsh login: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	var (
		mu  sync.Mutex
		out bytes.Buffer
	)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				mu.Lock()
				out.Write(buf[:n])
				mu.Unlock()
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Feed the password (and OTP) when tsh prints the matching prompt.
	go feedCredentials(ctx, ptmx, func() string {
		mu.Lock()
		defer mu.Unlock()
		return out.String()
	}, opts)

	waitErr := cmd.Wait()
	_ = ptmx.Close() // unblock the reader if it hasn't hit EOF yet
	<-readDone

	mu.Lock()
	text := out.String()
	mu.Unlock()

	if waitErr != nil {
		return LoginResult{Output: sanitizeOutput(text, opts)}, loginError(text, waitErr)
	}
	return LoginResult{Output: sanitizeOutput(text, opts)}, nil
}

// feedCredentials watches tsh's output and types the password, then the OTP,
// each once, when their prompts appear.
func feedCredentials(ctx context.Context, w io.Writer, snapshot func() string, opts LoginOptions) {
	sentPass := false
	sentOTP := false
	needOTP := strings.TrimSpace(opts.OTP) != ""

	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		s := strings.ToLower(snapshot())

		if !sentPass && strings.Contains(s, "password") {
			_, _ = io.WriteString(w, opts.Password+"\n")
			sentPass = true
			continue // let the OTP prompt render before we look for it
		}
		if sentPass && needOTP && !sentOTP && looksLikeOTPPrompt(s) {
			_, _ = io.WriteString(w, strings.TrimSpace(opts.OTP)+"\n")
			sentOTP = true
		}
		if sentPass && (sentOTP || !needOTP) {
			return
		}
	}
}

func looksLikeOTPPrompt(lower string) bool {
	return strings.Contains(lower, "otp") ||
		strings.Contains(lower, "one-time") ||
		strings.Contains(lower, "second factor") ||
		strings.Contains(lower, "mfa")
}

// sanitizeOutput trims the captured output and redacts the OTP code, which the
// terminal echoes back on its prompt line.
func sanitizeOutput(text string, opts LoginOptions) string {
	text = strings.TrimSpace(text)
	if otp := strings.TrimSpace(opts.OTP); otp != "" {
		text = strings.ReplaceAll(text, otp, "******")
	}
	return text
}

func loginError(output string, runErr error) error {
	msg := strings.TrimSpace(output)
	if msg == "" {
		msg = runErr.Error()
	}
	return fmt.Errorf("tsh login failed: %s", msg)
}
