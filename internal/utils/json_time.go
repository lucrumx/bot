package utils

import (
	"strconv"
	"time"
)

// Time is a wrapper around time.Time that allows unmarshaling from JSON strings representing milliseconds since epoch.
type Time struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface. Should be used if time comes as a string with double quotes.
func (t *Time) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == `""` || s == "0" {
		t.Time = time.Time{}
		return nil
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(ms)
	return nil
}

// MarshalJSON implements the json.Marshaler interface. Marshals Time as milliseconds since epoch.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("0"), nil
	}
	return []byte(strconv.FormatInt(t.Time.UnixMilli(), 10)), nil
}
