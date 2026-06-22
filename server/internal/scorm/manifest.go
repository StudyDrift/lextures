package scorm

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/beevik/etree"
)

// PackageType identifies an uploaded learning package format.
type PackageType string

const (
	TypeSCORM12   PackageType = "scorm12"
	TypeSCORM2004 PackageType = "scorm2004"
	TypeCMI5      PackageType = "cmi5"
)

// SCO is a shareable content object / assignable unit from a manifest.
type SCO struct {
	Identifier string   `json:"identifier"`
	Title      string   `json:"title"`
	LaunchHref string   `json:"launchHref"`
	Mastery    *float64 `json:"masteryScore,omitempty"`
}

// Manifest is parsed package metadata.
type Manifest struct {
	Title       string      `json:"title"`
	PackageType PackageType `json:"packageType"`
	Scos        []SCO       `json:"scos"`
}

const maxScormUploadBytes = 100 << 20 // 100 MB
const maxZipEntries = 10000
const maxZipFileBytes = 50 << 20

// ReadZipBytes loads a zip upload into memory with size limits.
func ReadZipBytes(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("scorm: package exceeds maximum size (%d MB)", limit>>20)
	}
	return data, nil
}

// ZipReaderAt wraps bytes for zip.NewReader.
func ZipReaderAt(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

// ParseAndValidateZip detects package type, parses manifest, validates structure.
func ParseAndValidateZip(r io.ReaderAt, size int64) (Manifest, json.RawMessage, error) {
	if size <= 0 {
		return Manifest{}, nil, fmt.Errorf("scorm: empty package")
	}
	if size > maxScormUploadBytes {
		return Manifest{}, nil, fmt.Errorf("scorm: package exceeds maximum size (%d MB)", maxScormUploadBytes>>20)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("scorm: invalid zip archive")
	}
	if len(zr.File) > maxZipEntries {
		return Manifest{}, nil, fmt.Errorf("scorm: zip contains too many entries (zip bomb guard)")
	}
	for _, f := range zr.File {
		if f.UncompressedSize64 > uint64(maxZipFileBytes) {
			return Manifest{}, nil, fmt.Errorf("scorm: zip entry exceeds maximum uncompressed size")
		}
	}
	pt, err := detectPackageType(zr)
	if err != nil {
		return Manifest{}, nil, err
	}
	switch pt {
	case TypeSCORM12:
		m, raw, err := parseSCORM12Manifest(zr)
		if err != nil {
			return Manifest{}, nil, err
		}
		m.PackageType = TypeSCORM12
		return m, raw, nil
	case TypeSCORM2004:
		return Manifest{}, nil, fmt.Errorf("scorm: SCORM 2004 packages are not supported yet; use SCORM 1.2")
	case TypeCMI5:
		return Manifest{}, nil, fmt.Errorf("scorm: cmi5 packages are not supported yet; use SCORM 1.2")
	default:
		return Manifest{}, nil, fmt.Errorf("scorm: unrecognized package (expected imsmanifest.xml or cmi5.xml)")
	}
}

func detectPackageType(zr *zip.Reader) (PackageType, error) {
	var hasManifest, hasCMI5 bool
	for _, f := range zr.File {
		base := strings.ToLower(strings.TrimPrefix(filepathBase(f.Name), "/"))
		switch base {
		case "imsmanifest.xml":
			hasManifest = true
		case "cmi5.xml":
			hasCMI5 = true
		}
	}
	if hasCMI5 {
		return TypeCMI5, nil
	}
	if hasManifest {
		return TypeSCORM12, nil
	}
	return "", fmt.Errorf("scorm: package missing imsmanifest.xml")
}

func filepathBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func parseSCORM12Manifest(zr *zip.Reader) (Manifest, json.RawMessage, error) {
	var manifestXML []byte
	var manifestPath string
	for _, f := range zr.File {
		if strings.EqualFold(filepathBase(f.Name), "imsmanifest.xml") {
			rc, err := f.Open()
			if err != nil {
				return Manifest{}, nil, fmt.Errorf("scorm: cannot read imsmanifest.xml")
			}
			manifestXML, err = io.ReadAll(io.LimitReader(rc, 4<<20))
			_ = rc.Close()
			if err != nil {
				return Manifest{}, nil, fmt.Errorf("scorm: cannot read imsmanifest.xml")
			}
			manifestPath = strings.ReplaceAll(f.Name, "\\", "/")
			break
		}
	}
	if len(manifestXML) == 0 {
		return Manifest{}, nil, fmt.Errorf("scorm: missing imsmanifest.xml")
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(manifestXML); err != nil {
		return Manifest{}, nil, fmt.Errorf("scorm: invalid imsmanifest.xml")
	}
	root := doc.Root()
	if root == nil {
		return Manifest{}, nil, fmt.Errorf("scorm: empty imsmanifest.xml")
	}
	// Detect 2004 from schemaversion
	for _, el := range walkElements(root) {
		if elLocalName(el.Tag) == "schemaversion" {
			ver := strings.TrimSpace(el.Text())
			if strings.HasPrefix(ver, "2004") || strings.Contains(strings.ToLower(ver), "cam") {
				return Manifest{}, nil, fmt.Errorf("scorm: SCORM 2004 packages are not supported yet; use SCORM 1.2")
			}
		}
	}
	resources := map[string]*etree.Element{}
	var orgTitle string
	for _, el := range walkElements(root) {
		tag := elLocalName(el.Tag)
		switch tag {
		case "organization":
			if orgTitle == "" {
				for _, t := range el.ChildElements() {
					if elLocalName(t.Tag) == "title" {
						orgTitle = strings.TrimSpace(t.Text())
					}
				}
			}
		case "resource":
			id := strings.TrimSpace(el.SelectAttrValue("identifier", ""))
			if id != "" {
				resources[id] = el
			}
		}
	}
	var scos []SCO
	for _, el := range walkElements(root) {
		if elLocalName(el.Tag) != "item" {
			continue
		}
		ref := strings.TrimSpace(el.SelectAttrValue("identifierref", ""))
		if ref == "" {
			continue
		}
		res, ok := resources[ref]
		if !ok {
			continue
		}
		href := strings.TrimSpace(res.SelectAttrValue("href", ""))
		scormType := strings.ToLower(res.SelectAttrValue("adlcp:scormtype", res.SelectAttrValue("scormtype", "")))
		if scormType != "" && scormType != "sco" {
			continue
		}
		title := ""
		for _, ch := range el.ChildElements() {
			if elLocalName(ch.Tag) == "title" {
				title = strings.TrimSpace(ch.Text())
			}
		}
		if title == "" {
			title = orgTitle
		}
		if href == "" {
			continue
		}
		launch := resolveLaunchHref(manifestPath, href)
		scos = append(scos, SCO{
			Identifier: strings.TrimSpace(el.SelectAttrValue("identifier", ref)),
			Title:      title,
			LaunchHref: launch,
		})
	}
	if len(scos) == 0 {
		return Manifest{}, nil, fmt.Errorf("scorm: no launchable SCO found in manifest")
	}
	m := Manifest{
		Title: orgTitle,
		Scos:  scos,
	}
	if m.Title == "" {
		m.Title = scos[0].Title
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return Manifest{}, nil, err
	}
	return m, append(json.RawMessage(nil), raw...), nil
}

func resolveLaunchHref(manifestPath, href string) string {
	href = strings.ReplaceAll(href, "\\", "/")
	if manifestPath == "" {
		return href
	}
	dir := manifestPath
	if i := strings.LastIndex(dir, "/"); i >= 0 {
		dir = dir[:i+1]
	} else {
		dir = ""
	}
	return strings.TrimPrefix(dir+href, "/")
}

func elLocalName(tag string) string {
	if i := strings.LastIndex(tag, "}"); i >= 0 {
		return tag[i+1:]
	}
	return tag
}

func walkElements(e *etree.Element) []*etree.Element {
	var out []*etree.Element
	var walk func(*etree.Element)
	walk = func(x *etree.Element) {
		if x == nil {
			return
		}
		out = append(out, x)
		for _, ch := range x.ChildElements() {
			walk(ch)
		}
	}
	walk(e)
	return out
}
