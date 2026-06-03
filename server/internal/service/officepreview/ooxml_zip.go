package officepreview

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"path"
	"strings"
)

type packageRel struct {
	Target string
	Type   string
}

func readZipFile(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer func() { _ = rc.Close() }()
			return io.ReadAll(rc)
		}
	}
	return nil, zip.ErrFormat
}

func parsePackageRels(zr *zip.Reader, relsPath string) (map[string]packageRel, error) {
	data, err := readZipFile(zr, relsPath)
	if err != nil {
		return map[string]packageRel{}, nil
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	out := make(map[string]packageRel)
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "Relationship" {
			id := xmlAttr(se, "Id")
			target := xmlAttr(se, "Target")
			relType := xmlAttr(se, "Type")
			if id != "" && target != "" {
				out[id] = packageRel{Target: target, Type: relType}
			}
		}
	}
	return out, nil
}

// resolveOOXMLPath joins a part path with a relationship target.
func resolveOOXMLPath(basePath, target string) string {
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/")
	}
	return path.Clean(path.Join(path.Dir(basePath), target))
}

func dataURIForPath(filePath string, raw []byte) string {
	ext := strings.ToLower(path.Ext(filePath))
	ct := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		ct = "image/jpeg"
	case ".gif":
		ct = "image/gif"
	case ".bmp":
		ct = "image/bmp"
	case ".svg":
		ct = "image/svg+xml"
	case ".webp":
		ct = "image/webp"
	case ".emf", ".wmf":
		ct = "image/emf"
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(raw)
}

func xmlAttr(se xml.StartElement, names ...string) string {
	for _, want := range names {
		for _, attr := range se.Attr {
			if attr.Name.Local == want && attr.Value != "" {
				return attr.Value
			}
		}
	}
	return ""
}

// xmlRelAttr reads OOXML relationship attributes (e.g. r:id, r:embed on blip elements).
func xmlRelAttr(se xml.StartElement, local string) string {
	for _, attr := range se.Attr {
		if attr.Name.Local == local && attr.Value != "" && strings.Contains(attr.Name.Space, "relationships") {
			return attr.Value
		}
	}
	return ""
}

func pptxRelatedPartPath(rels map[string]packageRel, basePath, typeSuffix string) string {
	for _, rel := range rels {
		if strings.Contains(rel.Type, typeSuffix) {
			return resolveOOXMLPath(basePath, rel.Target)
		}
	}
	return ""
}

func escapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
