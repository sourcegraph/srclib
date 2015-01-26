package store

import (
	"encoding/binary"
	"encoding/json"
	"io"

	"sourcegraph.com/sourcegraph/srclib/unit"

	pbio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
)

// Codec is the codec used by all file-backed stores. It should only be
// set at init time or when you can guarantee that no stores will be
// using the codec.
var Codec codec = ProtobufCodec{}

// A codec is an encoder and decoder pair used by the FS-backed store
// to encode and decode data stored in files.
type codec interface {
	// Encode encodes v into w.
	Encode(w io.Writer, v interface{}) error

	// NewDecoder creates a new decoder from r.
	NewDecoder(r io.Reader) decoder
}

type decoder interface {
	// Decode decodes the next value into v. It returns the number of
	// bytes that make up v's serialization, including any length
	// headers.
	Decode(v interface{}) (uint32, error)
}

type JSONCodec struct{}

func (JSONCodec) Encode(w io.Writer, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func (JSONCodec) NewDecoder(r io.Reader) decoder {
	return &jsonDecoder{Reader: r}
}

type jsonDecoder struct{ io.Reader }

func (d *jsonDecoder) Decode(v interface{}) (uint32, error) {
	var n uint32
	if err := binary.Read(d.Reader, binary.LittleEndian, &n); err != nil {
		return 0, err
	}
	return uint32(binary.Size(n)) + n, json.NewDecoder(io.LimitReader(d.Reader, int64(n))).Decode(v)
}

type ProtobufCodec struct{}

func (ProtobufCodec) Encode(w io.Writer, v interface{}) error {
	switch v := v.(type) {
	case *unit.SourceUnit:
		return JSONCodec{}.Encode(w, v)
	default:
		dw := pbio.NewDelimitedWriter(w)
		return dw.WriteMsg(v.(proto.Message))
	}
}

func (ProtobufCodec) NewDecoder(r io.Reader) decoder {
	return &protobufDecoder{r: r}
}

type protobufDecoder struct {
	r io.Reader

	j   decoder
	pbr pbio.ReadCloser
}

func (d *protobufDecoder) Decode(v interface{}) (uint32, error) {
	switch v := v.(type) {
	case *unit.SourceUnit:
		if d.j == nil {
			d.j = JSONCodec{}.NewDecoder(d.r)
		}
		return d.j.Decode(v)
	default:
		if d.pbr == nil {
			d.pbr = pbio.NewDelimitedReader(d.r, 2*1024*1024)
		}
		err := d.pbr.ReadMsg(v.(proto.Message))
		if err != nil {
			return 0, err
		}
		n := v.(interface {
			Size() int
		}).Size()
		buf := make([]byte, binary.MaxVarintLen64)
		return uint32(n + binary.PutUvarint(buf, uint64(n))), nil
	}
}
