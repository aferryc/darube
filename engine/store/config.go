package store

// ConnectionConfig holds the necessary details to map a saved database connection.
type ConnectionConfig struct {
	ID             string `json:"id"`
	ConnectionName string `json:"connection_name"`
	DBType         string `json:"db_type"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	DBName         string `json:"dbname"`
	EnableSSL      bool   `json:"enable_ssl"`
	SSLDir         string `json:"ssl_dir,omitempty"` // Deprecated in favor of specific paths
	CACertPath     string `json:"ca_cert_path,omitempty"`
	ClientCertPath string `json:"client_cert_path,omitempty"`
	ClientKeyPath  string `json:"client_key_path,omitempty"`
	User           string `json:"user"`
	Password       string `json:"password"`
	FolderID       string `json:"folder_id,omitempty"`
}

