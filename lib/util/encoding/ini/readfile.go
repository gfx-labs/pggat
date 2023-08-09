package ini

import (
	"bytes"
	"os"
)

func ReadFile(path string) ([]byte, error) {
	lines, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result bytes.Buffer
	result.Grow(len(lines))

	var line []byte
	for {
		line, lines, _ = bytes.Cut(lines, []byte{'\n'})
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			result.WriteByte('\n')
			if len(lines) == 0 {
				break
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("%include")) {
			included := string(bytes.TrimSpace(bytes.TrimPrefix(line, []byte("%include"))))
			b, err := ReadFile(included)
			if err != nil {
				return nil, err
			}
			result.Write(b)
			result.WriteByte('\n')
			continue
		}

		result.Write(line)
		result.WriteByte('\n')
	}

	return result.Bytes(), nil
}
