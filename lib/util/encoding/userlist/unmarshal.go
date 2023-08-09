package userlist

import (
	"bytes"
	"errors"
	"strconv"
)

func Unmarshal(data []byte) (map[string]string, error) {
	var res = make(map[string]string)

	var line []byte
	for {
		line, data, _ = bytes.Cut(data, []byte{'\n'})
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if len(data) == 0 {
				break
			}

			continue
		}

		fields := bytes.Fields(line)
		if len(fields) < 2 {
			return nil, errors.New("expected \"key\" \"value\"")
		}

		key, err := strconv.Unquote(string(fields[0]))
		if err != nil {
			return nil, err
		}
		value, err := strconv.Unquote(string(fields[1]))
		if err != nil {
			return nil, err
		}

		res[key] = value
	}

	return res, nil
}
