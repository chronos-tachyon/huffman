package huffman

import (
	"strings"
	"testing"
)

func TestDecoder(t *testing.T) {
	var d Decoder
	err := d.Init([]byte{4, 4, 3, 3, 3, 1})
	if err != nil {
		panic(err)
	}

	expectDump := strings.Join([]string{
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

	var buf strings.Builder
	_, _ = d.Dump(&buf)
	actualDump := buf.String()

	if expectDump != actualDump {
		t.Errorf("wrong output:\n\texpect: %s\n\tactual: %s", expectDump, actualDump)
	}
}
