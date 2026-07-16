// Package transcriptpesc emits PESC-shaped College Transcript XML from a canonical academic record (T01).
package transcriptpesc

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/academicrecord"
)

const pescNamespace = "urn:org:pesc:message:CollegeTranscript:v1.0.0"

// CollegeTranscript is a PESC College Transcript v1.x–shaped document.
type CollegeTranscript struct {
	XMLName     xml.Name        `xml:"CollegeTranscript"`
	Xmlns       string          `xml:"xmlns,attr"`
	Transmission TransmissionHeader `xml:"TransmissionData"`
	Student     Student         `xml:"Student"`
}

// TransmissionHeader identifies the message.
type TransmissionHeader struct {
	DocumentID      string `xml:"DocumentID"`
	CreatedDateTime string `xml:"CreatedDateTime"`
	DocumentTypeCode string `xml:"DocumentTypeCode"`
}

// Student wraps person and academic record sections.
type Student struct {
	Person         Person         `xml:"Person"`
	AcademicRecord AcademicRecord `xml:"AcademicRecord"`
}

// Person is the student identity block.
type Person struct {
	Name      Name   `xml:"Name"`
	SchoolID  string `xml:"SchoolAssignedPersonID,omitempty"`
}

// Name is a structured person name.
type Name struct {
	FirstName string `xml:"FirstName,omitempty"`
	LastName  string `xml:"LastName,omitempty"`
	FullName  string `xml:"CompositeName,omitempty"`
}

// AcademicRecord holds school and session (term) data.
type AcademicRecord struct {
	School   School    `xml:"School"`
	Academic []Session `xml:"AcademicSession"`
	GPA      *GPABlock `xml:"GPA,omitempty"`
}

// School identifies the issuing institution.
type School struct {
	OrganizationName string `xml:"OrganizationName"`
}

// Session is one academic term.
type Session struct {
	Name     string  `xml:"AcademicSessionName"`
	Courses  []Course `xml:"Course"`
	SessionGPA *GPABlock `xml:"GPA,omitempty"`
}

// Course is one course line.
type Course struct {
	SubjectCode      string  `xml:"CourseSubjectAbbreviation,omitempty"`
	Number           string  `xml:"CourseNumber,omitempty"`
	Title            string  `xml:"CourseTitle"`
	CreditAttempted  float64 `xml:"CourseCreditValue"`
	CreditEarned     float64 `xml:"CourseCreditEarned"`
	Grade            string  `xml:"CourseAcademicGrade"`
	QualityPoints    *float64 `xml:"CourseQualityPointsEarned,omitempty"`
}

// GPABlock is a PESC GPA element.
type GPABlock struct {
	CreditHoursAttempted float64 `xml:"CreditHoursAttempted"`
	CreditHoursEarned    float64 `xml:"CreditHoursEarned"`
	GradePointAverage    float64 `xml:"GradePointAverage,omitempty"`
}

// BuildXML renders the academic record as PESC-shaped College Transcript XML.
func BuildXML(rec *academicrecord.AcademicRecord) ([]byte, error) {
	if rec == nil {
		return nil, fmt.Errorf("transcriptpesc: nil record")
	}
	docID := rec.ContentDocumentID()
	first, last := splitName(rec.Student.Name)
	ct := CollegeTranscript{
		Xmlns: pescNamespace,
		Transmission: TransmissionHeader{
			DocumentID:       docID,
			CreatedDateTime:  rec.GeneratedAt,
			DocumentTypeCode: "CollegeTranscript",
		},
		Student: Student{
			Person: Person{
				Name: Name{
					FirstName: first,
					LastName:  last,
					FullName:  rec.Student.Name,
				},
				SchoolID: rec.Student.StudentID,
			},
			AcademicRecord: AcademicRecord{
				School: School{OrganizationName: rec.Institution.Name},
			},
		},
	}
	for _, term := range rec.Terms {
		sess := Session{Name: term.Label}
		for _, c := range term.Courses {
			subj, num := splitCourseCode(c.Code)
			course := Course{
				SubjectCode:     subj,
				Number:          num,
				Title:           c.Title,
				CreditAttempted: c.CreditsAttempted,
				CreditEarned:    c.CreditsEarned,
				Grade:           c.Grade,
				QualityPoints:   c.QualityPoints,
			}
			sess.Courses = append(sess.Courses, course)
		}
		if term.TermGPA != nil {
			sess.SessionGPA = &GPABlock{
				CreditHoursEarned: term.TermCredits,
				GradePointAverage: *term.TermGPA,
			}
		}
		ct.Student.AcademicRecord.Academic = append(ct.Student.AcademicRecord.Academic, sess)
	}
	if rec.Cumulative.GPA != nil {
		ct.Student.AcademicRecord.GPA = &GPABlock{
			CreditHoursAttempted: rec.Cumulative.CreditsAttempted,
			CreditHoursEarned:    rec.Cumulative.CreditsEarned,
			GradePointAverage:    *rec.Cumulative.GPA,
		}
	} else {
		ct.Student.AcademicRecord.GPA = &GPABlock{
			CreditHoursAttempted: rec.Cumulative.CreditsAttempted,
			CreditHoursEarned:    rec.Cumulative.CreditsEarned,
		}
	}

	out, err := xml.MarshalIndent(ct, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// ValidateStructure checks required PESC-shaped elements are present (lightweight conformance).
func ValidateStructure(xmlBytes []byte) error {
	var ct CollegeTranscript
	if err := xml.Unmarshal(xmlBytes, &ct); err != nil {
		return fmt.Errorf("transcriptpesc: parse: %w", err)
	}
	if ct.Transmission.DocumentTypeCode != "CollegeTranscript" {
		return fmt.Errorf("transcriptpesc: missing DocumentTypeCode")
	}
	if strings.TrimSpace(ct.Student.Person.Name.FullName) == "" &&
		strings.TrimSpace(ct.Student.Person.Name.LastName) == "" {
		return fmt.Errorf("transcriptpesc: missing student name")
	}
	if strings.TrimSpace(ct.Student.AcademicRecord.School.OrganizationName) == "" {
		return fmt.Errorf("transcriptpesc: missing school name")
	}
	return nil
}

func splitName(full string) (first, last string) {
	parts := strings.Fields(strings.TrimSpace(full))
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return "", parts[0]
	default:
		return parts[0], parts[len(parts)-1]
	}
}

func splitCourseCode(code string) (subj, num string) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ""
	}
	// Split on first digit run: MATH101 → MATH, 101
	i := strings.IndexFunc(code, func(r rune) bool { return r >= '0' && r <= '9' })
	if i <= 0 {
		return code, ""
	}
	return strings.TrimSpace(code[:i]), strings.TrimSpace(code[i:])
}
