package yrelay

import (
	"bytes"
	"testing"
)

func TestWriteVarUint(t *testing.T) {
	var buf bytes.Buffer
	writeVarUint(&buf, 0)
	if !bytes.Equal(buf.Bytes(), []byte{0}) {
		t.Fatalf("0: got %v", buf.Bytes())
	}
	buf.Reset()
	writeVarUint(&buf, 127)
	if !bytes.Equal(buf.Bytes(), []byte{127}) {
		t.Fatalf("127: got %v", buf.Bytes())
	}
	buf.Reset()
	writeVarUint(&buf, 128)
	if !bytes.Equal(buf.Bytes(), []byte{0x80, 0x01}) {
		t.Fatalf("128: got %v", buf.Bytes())
	}
}

func TestEncodeSyncUpdateRoundTrip(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5}
	msg := EncodeSyncUpdate(payload)
	if msg[0] != 0 || msg[1] != 2 {
		t.Fatalf("header: %v", msg[:2])
	}
	got := ExtractUpdateFromMsg(msg)
	if !bytes.Equal(got, payload) {
		t.Fatalf("extract: got %v want %v", got, payload)
	}
}

func TestEncodeEmptySyncSteps(t *testing.T) {
	if !bytes.Equal(EncodeEmptySyncStep1(), []byte{0, 0, 0}) {
		t.Fatal("sync step 1")
	}
	if !bytes.Equal(EncodeEmptySyncStep2(), []byte{0, 1, 0}) {
		t.Fatal("sync step 2")
	}
}

func TestExtractUpdateFromMsgInvalid(t *testing.T) {
	if ExtractUpdateFromMsg(nil) != nil {
		t.Fatal("nil")
	}
	if ExtractUpdateFromMsg([]byte{0}) != nil {
		t.Fatal("short")
	}
}
