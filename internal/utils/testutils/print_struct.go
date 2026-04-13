package testutils

import (
	"encoding/json"
	"testing"
)

func PrintStruct(t *testing.T, v interface{}, label string) {
	j, _ := json.MarshalIndent(v, "", "  ")
	if label != "" {
		label = label + ": "
	}
	t.Logf("%s\n%s", label, j)
}
