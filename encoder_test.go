package huffman

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncoder(t *testing.T) {
	var e Encoder
	e.Init(6, []uint32{5, 9, 12, 13, 16, 45})

	expectDump := strings.Join([]string{
		"Encoder{\n",
		"\tMinSize() = 1\n",
		"\tMaxSize() = 4\n",
		"\tEncode(0) = \"1110\"\n",
		"\tEncode(1) = \"1111\"\n",
		"\tEncode(2) = \"100\"\n",
		"\tEncode(3) = \"101\"\n",
		"\tEncode(4) = \"110\"\n",
		"\tEncode(5) = \"0\"\n",
		"}\n",
	}, "")

	var buf strings.Builder
	_, _ = e.Dump(&buf)
	actualDump := buf.String()

	if expectDump != actualDump {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectDump, actualDump)
	}

	actualSizes := e.SizeBySymbol()
	expectSizes := []byte{4, 4, 3, 3, 3, 1}
	if !bytes.Equal(expectSizes, actualSizes) {
		t.Errorf("wrong sizes:\n\texpect: %#v\n\tactual: %#v", expectSizes, actualSizes)
	}
}
