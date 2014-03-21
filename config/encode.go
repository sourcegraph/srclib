package config

import (
	"bytes"

	"github.com/sqs/toml"
)

// Marshal returns a TOML-encoded string representing the root configuration for
// a repository.
func Marshal(config *Repository) ([]byte, error) {
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(config)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
