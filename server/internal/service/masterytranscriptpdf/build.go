// SBG mastery transcript export (port of server/src/services/mastery_transcript_pdf.rs).
package masterytranscriptpdf

// Line is one standards row: Code (optional) and Label.
type Line struct {
	Code  string
	Label string
}
