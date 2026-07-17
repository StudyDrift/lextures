// Package transcriptedi emits ANSI X12 TS130 / SPEEDE–shaped transcript EDI (T06).
package transcriptedi

import (
	"fmt"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

// BuildTS130 renders a minimal X12 TS130 (Student Educational Record) interchange.
// This is SPEEDE-shaped for interoperability scaffolding; receivers may require
// profile-specific segments in a later conformance pass.
func BuildTS130(rec *academicrecord.AcademicRecord) ([]byte, error) {
	if rec == nil {
		return nil, fmt.Errorf("transcriptedi: nil record")
	}
	docID := rec.ContentDocumentID()
	if docID == "" {
		docID = "UNKNOWN"
	}
	now := time.Now().UTC()
	ctrl := strings.ReplaceAll(docID, "-", "")
	if len(ctrl) > 9 {
		ctrl = ctrl[:9]
	}
	isaDate := now.Format("060102")
	isaTime := now.Format("1504")
	gsDate := now.Format("20060102")

	name := strings.TrimSpace(rec.Student.Name)
	if name == "" {
		name = "UNKNOWN"
	}
	school := strings.TrimSpace(rec.Institution.Name)
	if school == "" {
		school = "UNKNOWN"
	}

	var b strings.Builder
	write := func(seg string) { b.WriteString(seg); b.WriteString("~\n") }

	write(fmt.Sprintf("ISA*00*          *00*          *ZZ*LEXTURES       *ZZ*RECEIVER       *%s*%s*^*00501*%s*0*P*>", isaDate, isaTime, pad9(ctrl)))
	write(fmt.Sprintf("GS*RA*LEXTURES*RECEIVER*%s*%s*%s*X*005010", gsDate, isaTime, ctrl))
	write(fmt.Sprintf("ST*130*%s", ctrl))
	write(fmt.Sprintf("BGN*11*%s*%s", docID, gsDate))
	write("ERP*ST")
	write(fmt.Sprintf("REF*TD*%s", docID))
	write(fmt.Sprintf("N1*SY*%s*91*%s", school, schoolCode(rec)))
	write(fmt.Sprintf("N1*SZ*%s", name))
	if id := strings.TrimSpace(rec.Student.StudentID); id != "" {
		write(fmt.Sprintf("REF*SY*%s", id))
	}
	for _, term := range rec.Terms {
		write(fmt.Sprintf("SSE*%s", sanitize(term.Label)))
		for _, c := range term.Courses {
			write(fmt.Sprintf("CRS*%s*%s*%s*%s*%g*%g",
				sanitize(c.Code), sanitize(c.Title), sanitize(c.Grade),
				sanitize(term.Label), c.CreditsAttempted, c.CreditsEarned))
		}
	}
	if rec.Cumulative.GPA != nil {
		write(fmt.Sprintf("SUM*N*%g*%g*%g", rec.Cumulative.CreditsAttempted, rec.Cumulative.CreditsEarned, *rec.Cumulative.GPA))
	} else {
		write(fmt.Sprintf("SUM*N*%g*%g", rec.Cumulative.CreditsAttempted, rec.Cumulative.CreditsEarned))
	}
	// Segment count: ST through SE inclusive — approximate from written lines minus ISA/GS/GE/IEA.
	segs := strings.Count(b.String(), "~")
	// ST..SE not yet including SE; add SE then GE/IEA.
	write(fmt.Sprintf("SE*%d*%s", segs+1, ctrl))
	write(fmt.Sprintf("GE*1*%s", ctrl))
	write(fmt.Sprintf("IEA*1*%s", pad9(ctrl)))

	out := []byte(b.String())
	if err := ValidateStructure(out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidateStructure checks required TS130 envelope segments are present.
func ValidateStructure(edi []byte) error {
	s := string(edi)
	for _, need := range []string{"ISA*", "GS*", "ST*130*", "SE*", "GE*", "IEA*"} {
		if !strings.Contains(s, need) {
			return fmt.Errorf("transcriptedi: missing %s", strings.TrimSuffix(need, "*"))
		}
	}
	return nil
}

func pad9(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "1"
	}
	for len(s) < 9 {
		s = "0" + s
	}
	if len(s) > 9 {
		return s[:9]
	}
	return s
}

func schoolCode(rec *academicrecord.AcademicRecord) string {
	if rec == nil {
		return "0000"
	}
	id := strings.TrimSpace(rec.Institution.CeebActID)
	if id == "" {
		return "0000"
	}
	return sanitize(id)
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "*", " ")
	s = strings.ReplaceAll(s, "~", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
