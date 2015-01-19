package store

import (
	"compress/gzip"
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

type JSONCodec struct{}

func (JSONCodec) Encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func (JSONCodec) Decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

type GobCodec struct{}

func (GobCodec) Encode(w io.Writer, v interface{}) error {
	return gob.NewEncoder(w).Encode(v)
}

func (GobCodec) Decode(r io.Reader, v interface{}) error {
	return gob.NewDecoder(r).Decode(v)
}

type GobAndJSONGzipCodec struct{}

func (GobAndJSONGzipCodec) Encode(w io.Writer, v interface{}) (err error) {
	gzw := gzip.NewWriter(w)
	defer func() {
		err2 := gzw.Close()
		if err == nil {
			err = err2
		}
	}()
	switch v.(type) {
	case *unit.SourceUnit:
		return json.NewEncoder(gzw).Encode(v)
	}
	return gob.NewEncoder(gzw).Encode(v)
}

func (GobAndJSONGzipCodec) Decode(r io.Reader, v interface{}) (err error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		err2 := gzr.Close()
		if err == nil {
			err = err2
		}
	}()
	switch v.(type) {
	case *unit.SourceUnit:
		return json.NewDecoder(gzr).Decode(v)
	}
	return gob.NewDecoder(gzr).Decode(v)
}

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
