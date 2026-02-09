package config

// AuthConfig contains configuration for JWT authentication.
type AuthConfig struct {
	JwtSecret    string `yaml:"jwt_secret"`
	JwtExpiresIn int    `yaml:"jwt_expires_in"`
}

// HTTPConfig contains configuration for an HTTP server.
type HTTPConfig struct {
	HTTPServerPort string     `yaml:"http_server_port"`
	Auth           AuthConfig `yaml:"auth"`
}
