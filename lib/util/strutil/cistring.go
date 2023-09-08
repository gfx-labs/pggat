package strutil

import (
	"bytes"
	"encoding/json"
	"strings"

	"pggat/lib/util/encoding/ini"
)

// CIString is a case-insensitive string.
type CIString struct {
	value string
}

func MakeCIString(value string) CIString {
	return CIString{
		strings.ToLower(value),
	}
}

func (T *CIString) String() string {
	return T.value
}

func (T *CIString) MarshalJSON() ([]byte, error) {
	return json.Marshal(T.value)
}

func (T *CIString) UnmarshalJSON(bytes []byte) error {
	if err := json.Unmarshal(bytes, &T.value); err != nil {
		return err
	}
	T.value = strings.ToLower(T.value)
	return nil
}

var _ json.Marshaler = (*CIString)(nil)
var _ json.Unmarshaler = (*CIString)(nil)

func (T *CIString) UnmarshalINI(b []byte) error {
	T.value = string(bytes.ToLower(b))
	return nil
}

var _ ini.Unmarshaller = (*CIString)(nil)
