// Package academicrecord assembles and hashes the canonical academic transcript model (T01).
package academicrecord

// SchemaVersion pins the canonical JSON shape for content hashing.
const SchemaVersion = "acadrec/1.0"

// TemplateVersion pins PDF/XML layout for reproducibility.
const TemplateVersion = "acadrec-render/1.0"

// Variant distinguishes sealed official records from previews and scoped variants.
type Variant string

const (
	VariantOfficial    Variant = "official"
	VariantUnofficial  Variant = "unofficial"
	VariantPartial     Variant = "partial"
	VariantInProgress  Variant = "in_progress"
)

// AcademicRecord is the byte-stable source of truth for issued transcripts.
type AcademicRecord struct {
	SchemaVersion   string            `json:"schemaVersion"`
	TemplateVersion string            `json:"templateVersion"`
	Variant         Variant           `json:"variant"`
	GeneratedAt     string            `json:"generatedAt"` // RFC3339 UTC, truncated to seconds
	Student         StudentBlock      `json:"student"`
	Institution     InstitutionBlock  `json:"institution"`
	Program         *ProgramBlock     `json:"program,omitempty"`
	Terms           []TermBlock       `json:"terms"`
	Cumulative      CumulativeBlock   `json:"cumulative"`
	Honors          []string          `json:"honors,omitempty"`
	Degrees         []DegreeBlock     `json:"degreesConferred,omitempty"`
	Standing        string            `json:"standing,omitempty"`
	Legend          map[string]string `json:"legend"`
	HasInProgress   bool              `json:"hasInProgress,omitempty"`
}

// StudentBlock identifies the learner on the record.
type StudentBlock struct {
	Name             string `json:"name"`
	StudentID        string `json:"studentId,omitempty"`
	BirthDateMasked  string `json:"birthDateMasked,omitempty"`
}

// InstitutionBlock identifies the issuing school.
type InstitutionBlock struct {
	Name     string `json:"name"`
	CeebActID string `json:"ceebActId,omitempty"`
}

// ProgramBlock is the student's program/plan summary.
type ProgramBlock struct {
	Degree string   `json:"degree,omitempty"`
	Major  []string `json:"major,omitempty"`
	Minor  []string `json:"minor,omitempty"`
}

// TermBlock groups course lines for one academic term.
type TermBlock struct {
	Label       string       `json:"label"`
	StartedOn   string       `json:"startedOn,omitempty"` // YYYY-MM-DD
	EndedOn     string       `json:"endedOn,omitempty"`
	TermID      string       `json:"termId,omitempty"`
	Courses     []CourseLine `json:"courses"`
	TermGPA     *float64     `json:"termGpa,omitempty"`
	TermCredits float64      `json:"termCredits"`
}

// CourseLine is one graded enrollment on the transcript.
type CourseLine struct {
	Code             string   `json:"code"`
	Title            string   `json:"title"`
	CreditsAttempted float64  `json:"creditsAttempted"`
	CreditsEarned    float64  `json:"creditsEarned"`
	Grade            string   `json:"grade"`
	QualityPoints    *float64 `json:"qualityPoints,omitempty"`
	InProgress       bool     `json:"inProgress,omitempty"`
	Transfer         bool     `json:"transfer,omitempty"`
}

// CumulativeBlock holds running GPA and credit totals.
type CumulativeBlock struct {
	GPA              *float64 `json:"gpa,omitempty"`
	CreditsAttempted float64  `json:"creditsAttempted"`
	CreditsEarned    float64  `json:"creditsEarned"`
	QualityPoints    float64  `json:"qualityPoints"`
}

// DegreeBlock is a conferred credential.
type DegreeBlock struct {
	Degree      string `json:"degree"`
	ConferredOn string `json:"conferredOn"`
}

// DefaultLegend returns the standard grade-symbol legend.
func DefaultLegend() map[string]string {
	return map[string]string{
		"A":  "Excellent",
		"A-": "Excellent",
		"B+": "Good",
		"B":  "Good",
		"B-": "Good",
		"C+": "Satisfactory",
		"C":  "Satisfactory",
		"C-": "Satisfactory",
		"D+": "Poor",
		"D":  "Poor",
		"D-": "Poor",
		"F":  "Failing",
		"P":  "Pass (not in GPA)",
		"W":  "Withdrawn (not in GPA)",
		"I":  "Incomplete",
		"AU": "Audit",
		"NC": "No credit",
		"IP": "In progress",
	}
}
