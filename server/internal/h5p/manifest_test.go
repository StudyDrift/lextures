package h5p

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseAndValidateZip_validTrueFalse(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("h5p.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`{"title":"Quiz","mainLibrary":"H5P.TrueFalse","embedTypes":["iframe"]}`))
	_ = zw.Close()
	data := buf.Bytes()
	m, _, err := ParseAndValidateZip(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if m.MainLibrary != "H5P.TrueFalse" {
		t.Fatalf("got %q", m.MainLibrary)
	}
}

func TestParseAndValidateZip_unsupportedLibrary(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("h5p.json")
	_, _ = w.Write([]byte(`{"title":"X","mainLibrary":"H5P.UnknownType"}`))
	_ = zw.Close()
	data := buf.Bytes()
	_, _, err := ParseAndValidateZip(bytes.NewReader(data), int64(len(data)))
	if err == nil {
		t.Fatal("expected unsupported error")
	}
}
