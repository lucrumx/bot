package utils

import (
	"fmt"
	"strconv"
)

// JSONFloat64 is a wrapper around float64 that allows unmarshaling from JSON strings.
type JSONFloat64 float64

// UnmarshalJSON implements the json.Unmarshaler interface. Should be used if float64 comes as a string with double quotes.
func (f *JSONFloat64) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*f = 0
		return nil
	}

	if data[0] == '"' {
		if len(data) < 2 {
			return fmt.Errorf("invalid JSON float64 format: %s", string(data))
		}
		data = data[1 : len(data)-1]
	}

	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}

	*f = JSONFloat64(val)

	return nil
}
