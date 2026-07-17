package transcriptpesc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

// MaxXMLBytes is the hard size limit for inbound PESC documents (10 MiB).
const MaxXMLBytes = 10 << 20

// ParseXML maps PESC-shaped College Transcript XML into the canonical academic-record model.
// The decoder disables external entities / DTD to mitigate XXE and entity expansion.
func ParseXML(xmlBytes []byte) (*academicrecord.AcademicRecord, error) {
	if len(xmlBytes) == 0 {
		return nil, fmt.Errorf("transcriptpesc: empty document")
	}
	if len(xmlBytes) > MaxXMLBytes {
		return nil, fmt.Errorf("transcriptpesc: document exceeds %d bytes", MaxXMLBytes)
	}
	var ct CollegeTranscript
	dec := xml.NewDecoder(bytes.NewReader(xmlBytes))
	dec.Strict = true
	dec.Entity = map[string]string{} // deny custom entity expansion
	// Go's encoding/xml never resolves external entities; keep CharsetReader nil (UTF-8 only).
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(strings.TrimSpace(charset)) {
		case "", "utf-8", "utf8":
			return input, nil
		default:
			return nil, fmt.Errorf("transcriptpesc: unsupported charset %q", charset)
		}
	}
	if err := dec.Decode(&ct); err != nil {
		return nil, fmt.Errorf("transcriptpesc: parse: %w", err)
	}
	if err := validateParsed(&ct); err != nil {
		return nil, err
	}
	return mapToCanonical(&ct), nil
}

func validateParsed(ct *CollegeTranscript) error {
	if ct == nil {
		return fmt.Errorf("transcriptpesc: nil document")
	}
	code := strings.TrimSpace(ct.Transmission.DocumentTypeCode)
	if code != "" && code != "CollegeTranscript" {
		return fmt.Errorf("transcriptpesc: unexpected DocumentTypeCode %q", code)
	}
	name := strings.TrimSpace(ct.Student.Person.Name.FullName)
	if name == "" {
		first := strings.TrimSpace(ct.Student.Person.Name.FirstName)
		last := strings.TrimSpace(ct.Student.Person.Name.LastName)
		name = strings.TrimSpace(first + " " + last)
	}
	if name == "" {
		return fmt.Errorf("transcriptpesc: missing student name")
	}
	if strings.TrimSpace(ct.Student.AcademicRecord.School.OrganizationName) == "" {
		return fmt.Errorf("transcriptpesc: missing school name")
	}
	return nil
}

func mapToCanonical(ct *CollegeTranscript) *academicrecord.AcademicRecord {
	full := strings.TrimSpace(ct.Student.Person.Name.FullName)
	if full == "" {
		full = strings.TrimSpace(strings.TrimSpace(ct.Student.Person.Name.FirstName) + " " + strings.TrimSpace(ct.Student.Person.Name.LastName))
	}
	generatedAt := strings.TrimSpace(ct.Transmission.CreatedDateTime)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	}
	rec := &academicrecord.AcademicRecord{
		SchemaVersion:   academicrecord.SchemaVersion,
		TemplateVersion: academicrecord.TemplateVersion,
		Variant:         academicrecord.VariantOfficial,
		GeneratedAt:     generatedAt,
		Student: academicrecord.StudentBlock{
			Name:      full,
			StudentID: strings.TrimSpace(ct.Student.Person.SchoolID),
		},
		Institution: academicrecord.InstitutionBlock{
			Name: strings.TrimSpace(ct.Student.AcademicRecord.School.OrganizationName),
		},
		Legend: academicrecord.DefaultLegend(),
	}
	var terms []academicrecord.TermBlock
	for _, sess := range ct.Student.AcademicRecord.Academic {
		term := academicrecord.TermBlock{Label: strings.TrimSpace(sess.Name)}
		var credits float64
		for _, c := range sess.Courses {
			code := strings.TrimSpace(c.SubjectCode + c.Number)
			if code == "" {
				code = strings.TrimSpace(c.Title)
			}
			line := academicrecord.CourseLine{
				Code:             code,
				Title:            strings.TrimSpace(c.Title),
				CreditsAttempted: c.CreditAttempted,
				CreditsEarned:    c.CreditEarned,
				Grade:            strings.TrimSpace(c.Grade),
				QualityPoints:    c.QualityPoints,
				Transfer:         true,
			}
			credits += c.CreditEarned
			term.Courses = append(term.Courses, line)
		}
		term.TermCredits = credits
		if sess.SessionGPA != nil && sess.SessionGPA.GradePointAverage > 0 {
			gpa := sess.SessionGPA.GradePointAverage
			term.TermGPA = &gpa
		}
		terms = append(terms, term)
	}
	rec.Terms = terms
	if ct.Student.AcademicRecord.GPA != nil {
		g := ct.Student.AcademicRecord.GPA
		rec.Cumulative = academicrecord.CumulativeBlock{
			CreditsAttempted: g.CreditHoursAttempted,
			CreditsEarned:    g.CreditHoursEarned,
		}
		if g.GradePointAverage > 0 {
			v := g.GradePointAverage
			rec.Cumulative.GPA = &v
		}
	}
	return rec
}
