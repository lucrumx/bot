package utils

import (
	"fmt"
)

// FlexibleBool is a boolean type that can be unmarshaled from various JSON representations, including true/false, "true"/"false", and null/empty string as false.
type FlexibleBool bool

// UnmarshalJSON implements the json.Unmarshaler interface. Should be used if boolean comes as a string with double quotes or as a boolean value.
func (b *FlexibleBool) UnmarshalJSON(data []byte) error {
	s := string(data)
	switch s {
	case "true", `"true"`:
		*b = true
	case "false", `"false"`, `""`, "null":
		*b = false
	default:
		return fmt.Errorf("FlexibleBool: cannot unmarshal %s", s)
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface. Marshals FlexibleBool as "true" or "false" strings.
func (b FlexibleBool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}
