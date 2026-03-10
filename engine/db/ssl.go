package db

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/go-sql-driver/mysql"
)

// registerMySQLTLSConfig builds and registers a custom tls.Config for go-sql-driver/mysql.
func registerMySQLTLSConfig(configName, caPath, certPath, keyPath string) error {
	rootCertPool := x509.NewCertPool()
	if caPath != "" {
		pem, err := os.ReadFile(caPath)
		if err == nil {
			if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
				fmt.Printf("Failed to append CA cert for MySQL from %s\n", caPath)
			}
		}
	}

	clientCert := make([]tls.Certificate, 0, 1)
	if certPath != "" && keyPath != "" {
		certs, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err == nil {
			clientCert = append(clientCert, certs)
		}
	}

	// Register it
	return mysql.RegisterTLSConfig(configName, &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: clientCert,
		InsecureSkipVerify: caPath == "", // Just skip verify if they didn't provide a CA to pin to
	})
}
