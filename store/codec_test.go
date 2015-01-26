package store

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime"
	"testing"

	"strings"

	"sourcegraph.com/sourcegraph/srclib/graph"
)

func TestCodec(t *testing.T) {
	tests := []struct {
		codec codec
	}{
		{
			codec: JSONCodec{},
		},
	}
	for _, test := range tests {
		ns := []int{0, 1, 2, 3, 4, 5, 10, 100, 1000, 5000}
		for _, n := range ns {
			orig := makeGraphData(t, n)

			var buf bytes.Buffer
			if err := test.codec.Encode(&buf, orig); err != nil {
				t.Errorf("%T (%d): Encode: %s", test.codec, n, err)
				continue
			}

			var decoded graph.Output
			if err := test.codec.Decode(&buf, &decoded); err != nil {
				t.Errorf("%T (%d): Decode: %s", test.codec, n, err)
				continue
			}

			if !reflect.DeepEqual(orig, decoded) {
				t.Errorf("%T (%d): got %v, want %v", test.codec, n, orig, decoded)
				continue
			}
		}
	}
}

func BenchmarkJSONCodec_Encode_500(b *testing.B)   { benchmarkCodec_Encode(b, JSONCodec{}, 500) }
func BenchmarkJSONCodec_Encode_5000(b *testing.B)  { benchmarkCodec_Encode(b, JSONCodec{}, 5000) }
func BenchmarkJSONCodec_Encode_50000(b *testing.B) { benchmarkCodec_Encode(b, JSONCodec{}, 50000) }

func BenchmarkJSONCodec_Decode_500(b *testing.B)   { benchmarkCodec_Decode(b, JSONCodec{}, 500) }
func BenchmarkJSONCodec_Decode_5000(b *testing.B)  { benchmarkCodec_Decode(b, JSONCodec{}, 5000) }
func BenchmarkJSONCodec_Decode_50000(b *testing.B) { benchmarkCodec_Decode(b, JSONCodec{}, 50000) }

func makeGraphData(t testing.TB, n int) graph.Output {
	data := graph.Output{}
	if n > 0 {
		data.Defs = make([]*graph.Def, n)
		data.Refs = make([]*graph.Ref, n)
	}
	for i := 0; i < n; i++ {
		data.Defs[i] = &graph.Def{
			DefKey:   graph.DefKey{Path: fmt.Sprintf("def-path-%d", i)},
			Name:     fmt.Sprintf("def-name-%d", i),
			Kind:     "mykind",
			DefStart: (i % 53) * 37,
			DefEnd:   (i%53)*37 + (i % 20),
			File:     fmt.Sprintf("dir%d/subdir%d/subsubdir%d/def-file-%d.foo", i%39, i%29, i%19, i%101),
			Exported: i%5 == 0,
			Local:    i%3 == 0,
			Data:     []byte(`"` + strings.Repeat("abcd", 50) + `"`),
		}
		data.Refs[i] = &graph.Ref{
			DefPath: fmt.Sprintf("ref-path-%d", i),
			Def:     i%5 == 0,
			Start:   (i % 51) * 39,
			End:     (i%51)*37 + (i % 18),
			File:    fmt.Sprintf("dir%d/subdir%d/subsubdir%d/ref-file-%d.foo", i%37, i%27, i%17, i%101),
		}
		if i%3 == 0 {
			data.Refs[i].DefUnit = fmt.Sprintf("def-unit-%d", i%17)
			data.Refs[i].DefUnitType = fmt.Sprintf("def-unit-type-%d", i%3)
			if i%7 == 0 {
				data.Refs[i].DefRepo = fmt.Sprintf("def-repo-%d", i%13)
			}
		}
	}
	return data
}

func benchmarkCodec_Encode(b *testing.B, c codec, n int) {
	orig := makeGraphData(b, n)

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := c.Encode(ioutil.Discard, orig); err != nil {
			b.Fatalf("%T (%d): Encode: %s", c, n, err)
		}
	}
}

func benchmarkCodec_Decode(b *testing.B, c codec, n int) {
	orig := makeGraphData(b, n)
	var buf bytes.Buffer
	if err := c.Encode(&buf, orig); err != nil {
		b.Errorf("%T (%d): Encode: %s", c, n, err)
	}
	bb := buf.Bytes()

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var decoded graph.Output
		if err := c.Decode(bytes.NewReader(bb), &decoded); err != nil {
			b.Fatalf("%T (%d): Decode: %s", c, n, err)
		}
	}
}
