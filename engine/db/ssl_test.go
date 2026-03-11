package db

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegisterMySQLTLSConfig_NoCerts(t *testing.T) {
	name := fmt.Sprintf("test_tls_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterMySQLTLSConfig_WithCAAndClientCert(t *testing.T) {
	dir := t.TempDir()

	// CA file with invalid PEM should still register (it just won't append).
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, []byte("not pem"), 0644); err != nil {
		t.Fatalf("write ca: %v", err)
	}
	name := fmt.Sprintf("test_tls_ca_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, caPath, "missing", "missing"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid client cert/key pair should be loaded and registered.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "darube-test"},
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

	name = fmt.Sprintf("test_tls_client_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, "", certPath, keyPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
