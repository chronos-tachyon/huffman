package huffman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func makeTestEncoder() Encoder {
	var e Encoder
	e.Init(6, []uint32{5, 9, 12, 13, 16, 45})
	return e
}

func TestEncoder_SizeBySymbol(t *testing.T) {
	e := makeTestEncoder()

	expectSizes := []byte{4, 4, 3, 3, 3, 1}
	actualSizes := e.SizeBySymbol()
	if !bytes.Equal(expectSizes, actualSizes) {
		t.Errorf("wrong sizes:\n\texpect: %#v\n\tactual: %#v", expectSizes, actualSizes)
	}
}

func TestEncoder_Encode(t *testing.T) {
	e := makeTestEncoder()

	type testRow struct {
		sym  Symbol
		size byte
		bits uint32
	}

	testData := [...]testRow{
		{sym: 0, size: 4, bits: 0x07},
		{sym: 1, size: 4, bits: 0x0f},
		{sym: 2, size: 3, bits: 0x01},
		{sym: 3, size: 3, bits: 0x05},
		{sym: 4, size: 3, bits: 0x03},
		{sym: 5, size: 1, bits: 0x00},
	}
	for _, row := range testData {
		name := fmt.Sprintf("Symbol(%d)", row.sym)
		t.Run(name, func(t *testing.T) {
			hc := e.Encode(row.sym)
			if hc.Size != row.size {
				t.Errorf("expected size %d, got %d", row.size, hc.Size)
			}
			if hc.Bits != row.bits {
				t.Errorf("expected bits %016b, got %016b", row.bits, hc.Bits)
			}
		})
	}
}

func TestEncoder_DebugString(t *testing.T) {
	e := makeTestEncoder()

	expectDebug := strings.Join([]string{
		"Encoder{\n",
		"\tMinSize() = 1\n",
		"\tMaxSize() = 4\n",
		"\tEncode(0) = \"0111\"\n",
		"\tEncode(1) = \"1111\"\n",
		"\tEncode(2) = \"001\"\n",
		"\tEncode(3) = \"101\"\n",
		"\tEncode(4) = \"011\"\n",
		"\tEncode(5) = \"0\"\n",
		"}\n",
	}, "")
	actualDebug := e.DebugString()
	if expectDebug != actualDebug {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectDebug, actualDebug)
	}
}

func TestEncoder_GoString(t *testing.T) {
	e := makeTestEncoder()

	expectGo := "NewEncoderFromSizes([]byte{4,4,3,3,3,1})"
	actualGo := e.GoString()
	if expectGo != actualGo {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectGo, actualGo)
	}
}

func TestEncoder_String(t *testing.T) {
	e := makeTestEncoder()

	expectString := "(Huffman encoder with 6 symbols, with coded lengths of 1 .. 4 bits)"
	actualString := e.String()
	if expectString != actualString {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectString, actualString)
	}
}

func TestEncoder_MarshalJSON(t *testing.T) {
	e := makeTestEncoder()

	raw, err := json.Marshal(e)
	if err != nil {
		t.Errorf("json.Marshal failed: %v", err)
	}
	expectJSON := "[4,4,3,3,3,1]"
	actualJSON := string(raw)
	if expectJSON != actualJSON {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectJSON, actualJSON)
	}
}

func TestEncoder_UnmarshalJSON(t *testing.T) {
	raw := []byte("[4,4,3,3,3,1]")

	var e Encoder
	err := json.Unmarshal(raw, &e)
	if err != nil {
		t.Errorf("json.Unmarshal failed: %v", err)
	}

	expectDebug := strings.Join([]string{
		"Encoder{\n",
		"\tMinSize() = 1\n",
		"\tMaxSize() = 4\n",
		"\tEncode(0) = \"0111\"\n",
		"\tEncode(1) = \"1111\"\n",
		"\tEncode(2) = \"001\"\n",
		"\tEncode(3) = \"101\"\n",
		"\tEncode(4) = \"011\"\n",
		"\tEncode(5) = \"0\"\n",
		"}\n",
	}, "")
	actualDebug := e.DebugString()
	if expectDebug != actualDebug {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectDebug, actualDebug)
	}
}
