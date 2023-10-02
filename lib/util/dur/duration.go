package dur

import (
	"encoding/json"
	"time"
)

type Duration time.Duration

func (T *Duration) Duration() time.Duration {
	return time.Duration(*T)
}

func (T *Duration) UnmarshalJSON(bytes []byte) error {
	// try as string
	var str string
	if err := json.Unmarshal(bytes, &str); err == nil {
		*(*time.Duration)(T), err = time.ParseDuration(str)
		return err
	}

	// try num
	var num int64
	if err := json.Unmarshal(bytes, &num); err != nil {
		return err
	}
	*T = Duration(num)

	return nil
}

var _ json.Unmarshaler = (*Duration)(nil)
