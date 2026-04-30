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

	// CA file with invalid PEM should error (can't verify anything with it).
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, []byte("not pem"), 0644); err != nil {
		t.Fatalf("write ca: %v", err)
	}
	name := fmt.Sprintf("test_tls_ca_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, caPath, "", ""); err == nil {
		t.Fatalf("expected error for invalid CA pem")
	}

	// Valid CA PEM should register (file and directory forms).
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	caTpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "darube-test-ca"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Minute),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:         true,
		BasicConstraintsValid: true,
	}
	caDer, err := x509.CreateCertificate(rand.Reader, &caTpl, &caTpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	caPath = filepath.Join(dir, "ca-valid.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDer}), 0644); err != nil {
		t.Fatalf("write ca: %v", err)
	}

	name = fmt.Sprintf("test_tls_ca_valid_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, caPath, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	caDir := filepath.Join(dir, "ca-dir")
	if err := os.MkdirAll(caDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDer}), 0644); err != nil {
		t.Fatalf("write ca dir: %v", err)
	}
	name = fmt.Sprintf("test_tls_ca_dir_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, caDir, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid client cert/key pair should be loaded and registered.
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	clientTpl := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "darube-test-client"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Minute),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	clientDer, err := x509.CreateCertificate(rand.Reader, &clientTpl, &clientTpl, &clientKey.PublicKey, clientKey)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDer}), 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)}), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	name = fmt.Sprintf("test_tls_client_%d", time.Now().UnixNano())
	if err := registerMySQLTLSConfig(name, "", certPath, keyPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
