package huffman

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func makeTestDecoder() Decoder {
	var d Decoder
	err := d.Init([]byte{4, 4, 3, 3, 3, 1})
	if err != nil {
		panic(err)
	}
	return d
}

func TestDecoder_SizeBySymbol(t *testing.T) {
	d := makeTestDecoder()

	expectSizes := []byte{4, 4, 3, 3, 3, 1}
	actualSizes := d.SizeBySymbol()
	if !bytes.Equal(expectSizes, actualSizes) {
		t.Errorf("wrong sizes:\n\texpect: %#v\n\tactual: %#v", expectSizes, actualSizes)
	}
}

func TestDecoder_Decode(t *testing.T) {
	d := makeTestDecoder()

	type testRow struct {
		size byte
		bits uint32
		min  byte
		max  byte
		sym  Symbol
	}

	testData := [...]testRow{
		{size: 0, bits: 0x00, min: 1, max: 4, sym: InvalidSymbol},
		{size: 1, bits: 0x00, min: 1, max: 1, sym: 5},
		{size: 1, bits: 0x01, min: 3, max: 4, sym: InvalidSymbol},
		{size: 2, bits: 0x01, min: 3, max: 3, sym: InvalidSymbol},
		{size: 2, bits: 0x03, min: 3, max: 4, sym: InvalidSymbol},
		{size: 3, bits: 0x01, min: 3, max: 3, sym: 2},
		{size: 3, bits: 0x03, min: 3, max: 3, sym: 4},
		{size: 3, bits: 0x05, min: 3, max: 3, sym: 3},
		{size: 3, bits: 0x07, min: 4, max: 4, sym: InvalidSymbol},
		{size: 4, bits: 0x07, min: 4, max: 4, sym: 0},
		{size: 4, bits: 0x0f, min: 4, max: 4, sym: 1},
	}
	for _, row := range testData {
		hc := MakeCode(row.size, row.bits)
		t.Run(hc.String(), func(t *testing.T) {
			sym, min, max := d.Decode(hc)
			if sym != row.sym {
				t.Errorf("expected symbol %d, got %d", row.sym, sym)
			}
			if min != row.min {
				t.Errorf("expected minimum size %d, got %d", row.min, min)
			}
			if max != row.max {
				t.Errorf("expected maximum size %d, got %d", row.max, max)
			}
		})
	}
}

func TestDecoder_DebugString(t *testing.T) {
	d := makeTestDecoder()

	expectDebug := strings.Join([]string{
		"Decoder{\n",
		"\tMinSize() = 1\n",
		"\tMaxSize() = 4\n",
		"\tDecode(\"\") = {-1, 1, 4}\n",
		"\tDecode(\"0\") = {5, 1, 1}\n",
		"\tDecode(\"1\") = {-1, 3, 4}\n",
		"\tDecode(\"01\") = {-1, 3, 3}\n",
		"\tDecode(\"11\") = {-1, 3, 4}\n",
		"\tDecode(\"001\") = {2, 3, 3}\n",
		"\tDecode(\"011\") = {4, 3, 3}\n",
		"\tDecode(\"101\") = {3, 3, 3}\n",
		"\tDecode(\"111\") = {-1, 4, 4}\n",
		"\tDecode(\"0111\") = {0, 4, 4}\n",
		"\tDecode(\"1111\") = {1, 4, 4}\n",
		"}\n",
	}, "")
	actualDebug := d.DebugString()
	if expectDebug != actualDebug {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectDebug, actualDebug)
	}
}

func TestDecoder_GoString(t *testing.T) {
	d := makeTestDecoder()

	expectGo := "NewDecoder([]byte{4,4,3,3,3,1})"
	actualGo := d.GoString()
	if expectGo != actualGo {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectGo, actualGo)
	}
}

func TestDecoder_String(t *testing.T) {
	d := makeTestDecoder()

	expectString := "(Huffman decoder with 6 symbols, with coded lengths of 1 .. 4 bits)"
	actualString := d.String()
	if expectString != actualString {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectString, actualString)
	}
}

func TestDecoder_MarshalJSON(t *testing.T) {
	d := makeTestDecoder()

	raw, err := json.Marshal(d)
	if err != nil {
		t.Errorf("json.Marshal failed: %v", err)
	}
	expectJSON := "[4,4,3,3,3,1]"
	actualJSON := string(raw)
	if expectJSON != actualJSON {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectJSON, actualJSON)
	}
}

func TestDecoder_UnmarshalJSON(t *testing.T) {
	raw := []byte("[4,4,3,3,3,1]")

	var d Decoder
	err := json.Unmarshal(raw, &d)
	if err != nil {
		t.Errorf("json.Unmarshal failed: %v", err)
	}

	expectDebug := strings.Join([]string{
		"Decoder{\n",
		"\tMinSize() = 1\n",
		"\tMaxSize() = 4\n",
		"\tDecode(\"\") = {-1, 1, 4}\n",
		"\tDecode(\"0\") = {5, 1, 1}\n",
		"\tDecode(\"1\") = {-1, 3, 4}\n",
		"\tDecode(\"01\") = {-1, 3, 3}\n",
		"\tDecode(\"11\") = {-1, 3, 4}\n",
		"\tDecode(\"001\") = {2, 3, 3}\n",
		"\tDecode(\"011\") = {4, 3, 3}\n",
		"\tDecode(\"101\") = {3, 3, 3}\n",
		"\tDecode(\"111\") = {-1, 4, 4}\n",
		"\tDecode(\"0111\") = {0, 4, 4}\n",
		"\tDecode(\"1111\") = {1, 4, 4}\n",
		"}\n",
	}, "")
	actualDebug := d.DebugString()
	if expectDebug != actualDebug {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectDebug, actualDebug)
	}
}
