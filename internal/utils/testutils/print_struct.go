package testutils

import (
	"encoding/json"
	"testing"
)

// PrintStruct print struct formated
func PrintStruct(t *testing.T, v interface{}, label string) {
	j, _ := json.MarshalIndent(v, "", "  ")
	if label != "" {
		label = label + ": "
	}
	t.Logf("%s\n%s", label, j)
}
