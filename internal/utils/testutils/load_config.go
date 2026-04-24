package testutils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/lucrumx/bot/internal/config"
)

// LoadTestConfig load local config.yaml to Config
func LoadTestConfig(t *testing.T) *config.Config {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "Could not get current file path")

	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../.."))
	configPath := filepath.Join(projectRoot, "config.yaml")

	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "config.yaml not found — needed for integration tests")

	var cfg config.Config
	require.NoError(t, yaml.Unmarshal(data, &cfg))

	return &cfg
}
