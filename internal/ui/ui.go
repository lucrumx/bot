// Package ui provides functionality for serving the frontend static.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:frontend/.output/public
var frontendContend embed.FS

// GetFileSystem returns the embedded filesystem for the frontend static files.
func GetFileSystem() (http.FileSystem, error) {
	root, err := fs.Sub(frontendContend, "frontend/.output/public")
	if err != nil {
		return nil, err
	}

	return http.FS(root), nil
}
