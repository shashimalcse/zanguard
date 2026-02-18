package schema

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse parses the given YAML bytes into a RawSchema.
// Uses KnownFields to reject unknown keys.
func Parse(data []byte) (*RawSchema, error) {
	var raw RawSchema
	dec := yaml.NewDecoder(byteReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	if raw.Types == nil {
		raw.Types = make(map[string]*RawType)
	}
	return &raw, nil
}

// ParseFile reads a file and calls Parse.
func ParseFile(path string) (*RawSchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema file %q: %w", path, err)
	}
	return Parse(data)
}

// byteReader wraps a byte slice for yaml.NewDecoder.
type byteReaderImpl struct {
	data []byte
	pos  int
}

func (r *byteReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func byteReader(data []byte) *byteReaderImpl {
	return &byteReaderImpl{data: data}
}
