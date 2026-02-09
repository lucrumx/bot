package config

// DatabaseConfig contains configuration for database connection.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DbName   string `yaml:"db_name"`
	Port     string `yaml:"port"`
	SslMode  string `yaml:"ssl_mode"`
}
