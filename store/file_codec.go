package store

import (
	"encoding/json"
	"io"
)

// Codec is the codec used by all file-backed stores. It should only be
// set at init time or when you can guarantee that no stores will be
// using the codec.
var Codec codec = JSONCodec{}

// A codec is an encoder and decoder pair used by the FS-backed store
// to encode and decode data stored in files.
type codec interface {
	// Encode encodes v into w.
	Encode(w io.Writer, v interface{}) error

	// Decode decodes r into v.
	Decode(r io.Reader, v interface{}) error
}

// A statefulCodec is a codec whose decoder reads beyond the most
// recently decoded object. These decoders must store state (the
// buffered portion of the byte stream).
type statefulCodec interface {
	NewDecoder(io.Reader) decoder
}

type decoder interface {
	Decode(interface{}) error
}

func newDecoder(c codec, r io.Reader) decoder {
	if sc, ok := c.(statefulCodec); ok {
		return sc.NewDecoder(r)
	}
	return unstatefulDecoder{c, r}
}

type unstatefulDecoder struct {
	c codec
	r io.Reader
}

func (d unstatefulDecoder) Decode(v interface{}) error { return d.c.Decode(d.r, v) }

type JSONCodec struct{}

func (JSONCodec) Encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func (JSONCodec) Decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func (JSONCodec) NewDecoder(r io.Reader) decoder {
	return json.NewDecoder(r)
}

var _ statefulCodec = (*JSONCodec)(nil)
