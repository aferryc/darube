package teleport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyTLSEnv_MySQL_UsesDeterministicTLSName(t *testing.T) {
	dir := t.TempDir()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "darube-test-client"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Minute),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	base := "u:p@tcp(127.0.0.1:3306)/db?tls=false"
	env := dbEnvInfo{
		Host:    "10.0.0.1",
		Port:    "3036",
		SSLCert: certPath,
		SSLKey:  keyPath,
	}

	dsn1, err := applyTLSEnv(base, "mysql", env)
	if err != nil {
		t.Fatalf("applyTLSEnv: %v", err)
	}
	dsn2, err := applyTLSEnv(base, "mysql", env)
	if err != nil {
		t.Fatalf("applyTLSEnv(2): %v", err)
	}
	if dsn1 != dsn2 {
		t.Fatalf("expected deterministic DSN, got %q vs %q", dsn1, dsn2)
	}
	if !strings.Contains(dsn1, "tcp(10.0.0.1:3036)") {
		t.Fatalf("expected host/port override: %s", dsn1)
	}
	if !strings.Contains(dsn1, "tls=teleport_tls_") {
		t.Fatalf("expected teleport tls name: %s", dsn1)
	}
}

