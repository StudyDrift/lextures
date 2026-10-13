package homeschoolrecords

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

// BuildHoursLogCSV exports the hours log as CSV bytes.
func BuildHoursLogCSV(log HoursLog) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"date", "subject", "hours", "source", "notes", "editable", "id"}); err != nil {
		return nil, err
	}
	for _, e := range log.Entries {
		id := ""
		if e.ID != nil {
			id = *e.ID
		}
		editable := "false"
		if e.Editable {
			editable = "true"
		}
		if err := w.Write([]string{
			e.Date, e.Subject, fmt.Sprintf("%.2f", e.Hours), e.Source, e.Notes, editable, id,
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildPortfolioZIP packs the portfolio PDF plus a hours-summary CSV into a ZIP.
func BuildPortfolioZIP(pdfBytes []byte, log HoursLog, studentSlug string) ([]byte, error) {
	slug := strings.TrimSpace(studentSlug)
	if slug == "" {
		slug = "student"
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	pdfName := fmt.Sprintf("%s-portfolio.pdf", slug)
	w, err := zw.Create(pdfName)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(pdfBytes); err != nil {
		return nil, err
	}
	csvBytes, err := BuildHoursLogCSV(log)
	if err != nil {
		return nil, err
	}
	cw, err := zw.Create(fmt.Sprintf("%s-hours.csv", slug))
	if err != nil {
		return nil, err
	}
	if _, err := cw.Write(csvBytes); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
