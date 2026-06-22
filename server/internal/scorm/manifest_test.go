package scorm

import (
	"archive/zip"
	"bytes"
	"testing"
)

const testManifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest identifier="test_pkg" version="1.0"
  xmlns="http://www.imsproject.org/xsd/imscp_rootv1p1p2"
  xmlns:adlcp="http://www.adlnet.org/xsd/adlcp_rootv1p2">
  <metadata>
    <schema>ADL SCORM</schema>
    <schemaversion>1.2</schemaversion>
  </metadata>
  <organizations default="org1">
    <organization identifier="org1">
      <title>Test SCORM Course</title>
      <item identifier="item1" identifierref="res1">
        <title>SCO One</title>
      </item>
    </organization>
  </organizations>
  <resources>
    <resource identifier="res1" type="webcontent" adlcp:scormtype="sco" href="index.html"/>
  </resources>
</manifest>`

func buildTestZip(manifestXML string, extraFiles map[string]string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, err := w.Create("imsmanifest.xml")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write([]byte(manifestXML)); err != nil {
		return nil, err
	}
	for name, content := range extraFiles {
		f, err := w.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestParseAndValidateZip_SCORM12(t *testing.T) {
	data, err := buildTestZip(testManifestXML, map[string]string{"index.html": "<html></html>"})
	if err != nil {
		t.Fatal(err)
	}
	reader := bytes.NewReader(data)
	m, raw, err := ParseAndValidateZip(reader, int64(len(data)))
	if err != nil {
		t.Fatalf("ParseAndValidateZip: %v", err)
	}
	if m.PackageType != TypeSCORM12 {
		t.Fatalf("package type %q want scorm12", m.PackageType)
	}
	if len(m.Scos) != 1 {
		t.Fatalf("scos len %d want 1", len(m.Scos))
	}
	if m.Scos[0].LaunchHref != "index.html" {
		t.Fatalf("launch href %q", m.Scos[0].LaunchHref)
	}
	if len(raw) == 0 {
		t.Fatal("expected manifest raw json")
	}
}

func TestParseAndValidateZip_rejectsCMI5(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, err := w.Create("cmi5.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("<course/>")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	_, _, err = ParseAndValidateZip(bytes.NewReader(data), int64(len(data)))
	if err == nil {
		t.Fatal("expected cmi5 rejection")
	}
}

func TestGradePoints_scaled(t *testing.T) {
	raw := 85.0
	max := 100.0
	state := RegistrationState{
		CompletionStatus: "passed",
		ScoreRaw:         &raw,
		ScoreMax:         &max,
	}
	pts, maxPts, ok := GradePoints(state, 100)
	if !ok {
		t.Fatal("expected grade points")
	}
	if pts != 85 || maxPts != 100 {
		t.Fatalf("got pts=%v max=%v", pts, maxPts)
	}
}

func TestApplyCMIUpdate_suspendData(t *testing.T) {
	state := RegistrationState{}
	ApplyCMIUpdate(&state, CMIUpdate{
		"cmi.core.suspend_data": "bookmark123",
		"cmi.core.lesson_location": "page2",
	})
	if state.SuspendData != "bookmark123" || state.Location != "page2" {
		t.Fatalf("suspend=%q location=%q", state.SuspendData, state.Location)
	}
}
