package store

import (
	"encoding/binary"
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

type JSONCodec struct{}

func (JSONCodec) Encode(w io.Writer, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int64(len(b))); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func (JSONCodec) Decode(r io.Reader, v interface{}) error {
	var n int64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return err
	}
	return json.NewDecoder(io.LimitReader(r, n)).Decode(v)
}
