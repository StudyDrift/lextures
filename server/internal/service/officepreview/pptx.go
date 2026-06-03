package officepreview

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"path"
	"regexp"
	"strings"

	markitdown "github.com/conductor-oss/markitdown"
)

var pptxImagePlaceholderRe = regexp.MustCompile(`!\[[^\]]*\]\(image\)`)

// convertPptxToHTML renders slides as positioned visual canvases (fallback: text extraction).
func convertPptxToHTML(data []byte, filename, mimeType string) (string, error) {
	if html, err := convertPptxToVisualHTML(data, filename, mimeType); err == nil {
		return html, nil
	}
	return convertPptxToHTMLText(data, filename, mimeType)
}

// convertPptxToHTMLText builds slide HTML from extracted markdown text.
func convertPptxToHTMLText(data []byte, filename, mimeType string) (string, error) {
	result, err := converter.ConvertReader(bytes.NewReader(data), markitdown.StreamInfo{
		Extension: ".pptx",
		Filename:  filename,
		MIMEType:  mimeType,
	})
	if err != nil {
		return "", err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pptx zip: %w", err)
	}

	slidePaths, err := pptxSlidePaths(zr)
	if err != nil {
		return "", err
	}
	if len(slidePaths) == 0 {
		return wrapHTMLDocument(`<p class="pptx-empty">This presentation has no slides to preview.</p>`), nil
	}

	slideBodies := splitPptxMarkdownBySlide(result.Markdown)
	var htmlSlides strings.Builder
	for i, slidePath := range slidePaths {
		md := ""
		if i < len(slideBodies) {
			md = slideBodies[i]
		}
		images := extractPptxSlideImages(zr, slidePath)
		md, usedImages := replacePptxImagePlaceholders(md, images)
		inner := markdownFragmentToHTML(md)
		imgHTML := pptxSlideImagesHTML(images[usedImages:])
		fmt.Fprintf(&htmlSlides,
			`<section class="slide"><header class="slide-header">Slide %d</header><div class="slide-body">%s%s</div></section>`,
			i+1,
			inner,
			imgHTML,
		)
	}

	return wrapHTMLDocument(htmlSlides.String()), nil
}

func replacePptxImagePlaceholders(md string, images []pptxImage) (string, int) {
	if len(images) == 0 {
		return md, 0
	}
	idx := 0
	out := pptxImagePlaceholderRe.ReplaceAllStringFunc(md, func(_ string) string {
		if idx >= len(images) {
			return ""
		}
		img := images[idx]
		idx++
		return fmt.Sprintf(`<img src="%s" alt="%s"/>`, img.dataURI, escapeAttr(img.alt))
	})
	return out, idx
}

func pptxSlideImagesHTML(images []pptxImage) string {
	if len(images) == 0 {
		return ""
	}
	var b strings.Builder
	for _, img := range images {
		fmt.Fprintf(&b,
			`<figure class="slide-figure"><img src="%s" alt="%s"/></figure>`,
			img.dataURI,
			escapeAttr(img.alt),
		)
	}
	return sanitizeHTML(b.String())
}

func splitPptxMarkdownBySlide(md string) []string {
	parts := strings.Split(md, "<!-- Slide number:")
	if len(parts) <= 1 {
		return []string{strings.TrimSpace(md)}
	}
	out := make([]string, 0, len(parts)-1)
	for _, p := range parts[1:] {
		if idx := strings.Index(p, "-->"); idx >= 0 {
			p = p[idx+3:]
		}
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func pptxSlidePaths(zr *zip.Reader) ([]string, error) {
	presData, err := readZipFile(zr, "ppt/presentation.xml")
	if err != nil {
		return fallbackSlidePaths(zr), nil
	}
	rels, _ := parsePackageRels(zr, "ppt/_rels/presentation.xml.rels")
	var rids []string
	dec := xml.NewDecoder(bytes.NewReader(presData))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "sldId" {
			if rid := xmlRelAttr(se, "id"); rid != "" {
				rids = append(rids, rid)
			}
		}
	}
	var paths []string
	for _, rid := range rids {
		if rel, ok := rels[rid]; ok {
			paths = append(paths, resolveOOXMLPath("ppt/presentation.xml", rel.Target))
		}
	}
	if len(paths) == 0 {
		return fallbackSlidePaths(zr), nil
	}
	return paths, nil
}

func fallbackSlidePaths(zr *zip.Reader) []string {
	var paths []string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			paths = append(paths, f.Name)
		}
	}
	return paths
}

type pptxImage struct {
	alt     string
	dataURI string
}

func extractPptxSlideImages(zr *zip.Reader, slidePath string) []pptxImage {
	slideData, err := readZipFile(zr, slidePath)
	if err != nil {
		return nil
	}
	relsPath := pptxSlideRelsPath(slidePath)
	rels, _ := parsePackageRels(zr, relsPath)
	embeds := findPptxBlipEmbeds(slideData)
	var images []pptxImage
	for _, embed := range embeds {
		rel, ok := rels[embed.id]
		if !ok {
			continue
		}
		mediaPath := resolveOOXMLPath(slidePath, rel.Target)
		raw, err := readZipFile(zr, mediaPath)
		if err != nil {
			continue
		}
		ext := strings.ToLower(path.Ext(mediaPath))
		if ext == ".emf" || ext == ".wmf" {
			continue
		}
		images = append(images, pptxImage{
			alt:     embed.alt,
			dataURI: dataURIForPath(mediaPath, raw),
		})
	}
	return images
}

type blipEmbed struct {
	id  string
	alt string
}

func findPptxBlipEmbeds(slideData []byte) []blipEmbed {
	dec := xml.NewDecoder(bytes.NewReader(slideData))
	var out []blipEmbed
	var currentAlt string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "docPr", "cNvPr":
				currentAlt = xmlAttr(t, "descr", "title", "name")
			case "blip":
				id := xmlRelAttr(t, "embed")
				if id != "" {
					alt := currentAlt
					if alt == "" {
						alt = "Slide image"
					}
					out = append(out, blipEmbed{id: id, alt: alt})
				}
				currentAlt = ""
			}
		case xml.EndElement:
			if t.Name.Local == "pic" {
				currentAlt = ""
			}
		}
	}
	return out
}

func pptxSlideRelsPath(slidePath string) string {
	base := path.Base(slidePath)
	return "ppt/slides/_rels/" + strings.TrimSuffix(base, ".xml") + ".xml.rels"
}
