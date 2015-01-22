package store

import (
	"encoding/gob"
	"encoding/json"
	"io"

	"sourcegraph.com/sourcegraph/srclib/unit"
)

// A Codec is an encoder and decoder pair used by the flat file store
// to encode and decode data stored in files.
type Codec interface {
	// Encode encodes v into w.
	Encode(w io.Writer, v interface{}) error

	// Decode decodes r into v.
	Decode(r io.Reader, v interface{}) error
}

type statefulCodec interface {
	NewDecoder(io.Reader) decoder
}

type decoder interface {
	Decode(interface{}) error
}

func newDecoder(c Codec, r io.Reader) decoder {
	if sc, ok := c.(statefulCodec); ok {
		return sc.NewDecoder(r)
	}
	return unstatefulDecoder{c, r}
}

type unstatefulDecoder struct {
	c Codec
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

type GobAndJSONCodec struct{}

func (GobAndJSONCodec) Encode(w io.Writer, v interface{}) (err error) {
	switch v.(type) {
	case *unit.SourceUnit:
		return json.NewEncoder(w).Encode(v)
	}
	return gob.NewEncoder(w).Encode(v)
}

func (GobAndJSONCodec) Decode(r io.Reader, v interface{}) (err error) {
	switch v.(type) {
	case *unit.SourceUnit:
		return json.NewDecoder(r).Decode(v)
	}
	return gob.NewDecoder(r).Decode(v)
}

func (GobAndJSONCodec) NewDecoder(r io.Reader) decoder {
	// WARNING: this should really return a decoder type that
	// type-switches on v to choose between gob and json, but it
	// doesn't. and that works for now because only graph data calls
	// NewDecoder.
	return gob.NewDecoder(r)
}

var _ statefulCodec = (*GobAndJSONCodec)(nil)
