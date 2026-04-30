package sslutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// RegisterMySQLTLSConfig builds and registers a named tls.Config for go-sql-driver/mysql.
func RegisterMySQLTLSConfig(configName, caPath, certPath, keyPath string) error {
	rootCertPool, err := x509.SystemCertPool()
	if err != nil || rootCertPool == nil {
		rootCertPool = x509.NewCertPool()
	}
	if caPath != "" {
		info, err := os.Stat(caPath)
		if err != nil {
			return fmt.Errorf("mysql tls: stat CA cert %q: %w", caPath, err)
		}

		appendPEM := func(p string) (bool, error) {
			pem, err := os.ReadFile(p)
			if err != nil {
				return false, fmt.Errorf("mysql tls: read CA cert %q: %w", p, err)
			}
			if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
				return false, fmt.Errorf("mysql tls: no valid certs found in CA PEM %q", p)
			}
			return true, nil
		}

		if info.IsDir() {
			entries, err := os.ReadDir(caPath)
			if err != nil {
				return fmt.Errorf("mysql tls: read CA dir %q: %w", caPath, err)
			}
			added := false
			for _, ent := range entries {
				if ent.IsDir() {
					continue
				}
				p := filepath.Join(caPath, ent.Name())
				ok, err := appendPEM(p)
				if err != nil {
					continue
				}
				added = added || ok
			}
			if !added {
				return fmt.Errorf("mysql tls: no valid CA certs found in dir %q", caPath)
			}
		} else {
			if _, err := appendPEM(caPath); err != nil {
				return err
			}
		}
	}

	clientCert := make([]tls.Certificate, 0, 1)
	if certPath != "" || keyPath != "" {
		if certPath == "" || keyPath == "" {
			return fmt.Errorf("mysql tls: both client cert and key are required (cert=%q key=%q)", certPath, keyPath)
		}
		certs, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return fmt.Errorf("mysql tls: load client cert/key (cert=%q key=%q): %w", certPath, keyPath, err)
		}
		clientCert = append(clientCert, certs)
	}

	tlsCfg := &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: clientCert,
	}

	// "Enable SSL" in Darube is primarily "encrypt the connection".
	// If the user provides a CA, we verify the chain but intentionally do not
	// require hostname validation (similar to Postgres sslmode=verify-ca).
	// If no CA is provided, we skip verification to maximize compatibility.
	if caPath == "" {
		tlsCfg.InsecureSkipVerify = true
	} else {
		tlsCfg.InsecureSkipVerify = true
		tlsCfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("mysql tls: no server certificates")
			}
			leaf, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("mysql tls: parse leaf cert: %w", err)
			}

			intermediates := x509.NewCertPool()
			for _, der := range rawCerts[1:] {
				c, parseErr := x509.ParseCertificate(der)
				if parseErr != nil {
					continue
				}
				intermediates.AddCert(c)
			}

			_, err = leaf.Verify(x509.VerifyOptions{
				Roots:         rootCertPool,
				Intermediates: intermediates,
				CurrentTime:   time.Now(),
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			if err != nil {
				return fmt.Errorf("mysql tls: verify server cert: %w", err)
			}
			return nil
		}
	}

	if err := mysql.RegisterTLSConfig(configName, tlsCfg); err != nil {
		// RegisterTLSConfig rejects duplicate names; treat that as success so
		// reconnects don't unexpectedly downgrade to skip-verify.
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "already") && strings.Contains(msg, "registered") {
			return nil
		}
		return err
	}
	return nil
}
