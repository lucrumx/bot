// Package utils provides utility functions for the application.
package utils

import (
	"log"
	"os"
)

// GetEnv retrieves the value of the environment variable named by the key.
// If the variable is not present and a defaultValue is provided, it returns the defaultValue.
// If the variable is not present and no defaultValue is provided, it logs a fatal error.
func GetEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	if defaultValue != "" {
		return defaultValue
	}

	log.Fatalf("Environment variable %s not set and no default value provided", key)
	return ""
}
