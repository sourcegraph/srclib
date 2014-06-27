package util2

import (
	"bytes"

	"github.com/aybabtme/color/brush"
)

// ColorizeDiff takes a byte slice of lines and returns the same, but with diff
// highlighting. That is, lines starting with '+' are green and lines starting
// with '-' are red.
func ColorizeDiff(diff []byte) []byte {
	lines := bytes.Split(diff, []byte{'\n'})
	for i, line := range lines {
		if bytes.HasPrefix(line, []byte{'-'}) {
			lines[i] = []byte(brush.Red(string(line)).String())
		}
		if bytes.HasPrefix(line, []byte{'+'}) {
			lines[i] = []byte(brush.Green(string(line)).String())
		}
	}
	return bytes.Join(lines, []byte{'\n'})
}
