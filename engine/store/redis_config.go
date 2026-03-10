package store

// RedisConfig holds the connection details for a Redis instance.
type RedisConfig struct {
	ID             string `json:"id"`
	ConnectionName string `json:"connection_name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user,omitempty"`
	Password       string `json:"password,omitempty"`
	IsCluster      bool   `json:"is_cluster"`
	FolderID       string `json:"folder_id,omitempty"`
}
