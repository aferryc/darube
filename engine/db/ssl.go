package db

import "engine/sslutil"

func registerMySQLTLSConfig(configName, caPath, certPath, keyPath string) error {
	return sslutil.RegisterMySQLTLSConfig(configName, caPath, certPath, keyPath)
}
