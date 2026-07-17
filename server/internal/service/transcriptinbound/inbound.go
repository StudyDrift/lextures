// Package transcriptinbound implements T07 inbound receive → validate → parse → match.
package transcriptinbound

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/academicrecord"
	"github.com/lextures/lextures/server/internal/service/transcriptpesc"
)

// AfterReceived is set by the job-queue package to enqueue async processing.
var AfterReceived func(ctx context.Context, pool *pgxpool.Pool, inboundID uuid.UUID)

// NotifyFn sends applicant email for inbound received/accepted (wired to email queue).
var NotifyFn func(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, eventType, title, message, uniqueKey string)

// Size / type limits for inbound content (FR-7).
const (
	MaxInboundBytes = 10 << 20 // 10 MiB
	AutoMatchMin    = 0.85
)

var (
	ErrFeatureDisabled = errors.New("transcript inbound is not enabled")
	ErrTooLarge        = errors.New("inbound document exceeds size limit")
	ErrUnsupportedType = errors.New("unsupported inbound content type")
	ErrUnsafePayload   = errors.New("inbound payload failed security validation")
)

// ReceiveInput is a document posted to the inbound pipeline.
type ReceiveInput struct {
	OrgID       uuid.UUID
	Channel     string
	SourceName  string
	ExternalRef string
	Format      string // optional; inferred from content-type / magic when empty
	ContentType string
	RawBytes    []byte
	StudentName string // optional declared metadata (PDF path)
	StudentDOB  string
	StudentRef  string
}

// ReceiveResult is the intake row after insert (processing may be async).
type ReceiveResult struct {
	Document  *transcriptsrepo.InboundDocument
	Duplicate bool
}

// Receive validates, stores immutably, and enqueues processing.
func Receive(ctx context.Context, pool *pgxpool.Pool, in ReceiveInput) (*ReceiveResult, error) {
	if pool == nil {
		return nil, errors.New("transcriptinbound: nil pool")
	}
	if len(in.RawBytes) == 0 {
		return nil, errors.New("transcriptinbound: empty body")
	}
	if len(in.RawBytes) > MaxInboundBytes {
		return nil, ErrTooLarge
	}
	channel := strings.TrimSpace(in.Channel)
	if channel == "" {
		channel = transcriptsrepo.InboundChannelAPIPeer
	}
	format := strings.TrimSpace(in.Format)
	if format == "" {
		format = inferFormat(in.ContentType, in.RawBytes)
	}
	if err := validateContent(format, in.ContentType, in.RawBytes); err != nil {
		return nil, err
	}

	source := strings.TrimSpace(in.SourceName)
	extRef := strings.TrimSpace(in.ExternalRef)
	if source != "" && extRef != "" {
		if existing, err := transcriptsrepo.FindInboundByDedupe(ctx, pool, in.OrgID, source, extRef); err == nil && existing != nil {
			return &ReceiveResult{Document: existing, Duplicate: true}, nil
		}
	}

	sum := sha256.Sum256(in.RawBytes)
	hash := hex.EncodeToString(sum[:])
	rawKey := fmt.Sprintf("inbound/%s/%s/%s", in.OrgID.String(), hash[:16], format)
	ct := strings.TrimSpace(in.ContentType)
	var ctPtr *string
	if ct != "" {
		ctPtr = &ct
	}
	doc, err := transcriptsrepo.InsertInboundDocument(ctx, pool, transcriptsrepo.InsertInboundInput{
		OrgID:       in.OrgID,
		Channel:     channel,
		SourceName:  strPtr(source),
		ExternalRef: strPtr(extRef),
		Format:      format,
		RawKey:      rawKey,
		RawBytes:    in.RawBytes,
		ContentHash: hash,
		ContentType: ctPtr,
		StudentName: strPtr(strings.TrimSpace(in.StudentName)),
		StudentDOB:  strPtr(strings.TrimSpace(in.StudentDOB)),
		StudentRef:  strPtr(strings.TrimSpace(in.StudentRef)),
	})
	if errors.Is(err, transcriptsrepo.ErrInboundDuplicate) {
		existing, ferr := transcriptsrepo.FindInboundByDedupe(ctx, pool, in.OrgID, source, extRef)
		if ferr == nil && existing != nil {
			return &ReceiveResult{Document: existing, Duplicate: true}, nil
		}
		return nil, transcriptsrepo.ErrInboundDuplicate
	}
	if err != nil {
		return nil, err
	}
	// Parse + match inline for peer API latency (p95 < 3s). Large batch channels
	// (SFTP/email workers) may call AfterReceived / Process separately instead.
	processed, perr := Process(ctx, pool, doc.ID)
	if perr == nil && processed != nil {
		doc = processed
	} else if AfterReceived != nil {
		AfterReceived(ctx, pool, doc.ID)
	}
	return &ReceiveResult{Document: doc, Duplicate: false}, nil
}

// Process parses, matches, and updates status for one inbound document.
func Process(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*transcriptsrepo.InboundDocument, error) {
	doc, err := transcriptsrepo.GetInboundDocument(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if doc.Status == transcriptsrepo.InboundAccepted || doc.Status == transcriptsrepo.InboundRejected {
		return doc, nil
	}

	if unsafe, reason := detectUnsafe(doc.RawBytes, doc.Format); unsafe {
		updated, uerr := transcriptsrepo.UpdateInboundAfterProcess(ctx, pool, id,
			transcriptsrepo.InboundQuarantined, nil, nil, nil, nil,
			doc.StudentName, doc.StudentDOB, doc.StudentRef, true, &reason)
		if uerr != nil {
			return nil, uerr
		}
		_ = transcriptsrepo.AppendInboundEvent(ctx, pool, id, "quarantined", nil, map[string]any{"reason": reason})
		return updated, nil
	}

	switch doc.Format {
	case transcriptsrepo.InboundFormatPESC:
		return processPESC(ctx, pool, doc)
	case transcriptsrepo.InboundFormatPDF:
		return processPDF(ctx, pool, doc)
	default:
		reason := "format requires manual review"
		updated, uerr := transcriptsrepo.UpdateInboundAfterProcess(ctx, pool, id,
			transcriptsrepo.InboundUnmatched, nil, nil, nil, nil,
			doc.StudentName, doc.StudentDOB, doc.StudentRef, true, nil)
		if uerr != nil {
			return nil, uerr
		}
		_ = transcriptsrepo.AppendInboundEvent(ctx, pool, id, "unmatched", nil, map[string]any{"reason": reason})
		return updated, nil
	}
}

func processPESC(ctx context.Context, pool *pgxpool.Pool, doc *transcriptsrepo.InboundDocument) (*transcriptsrepo.InboundDocument, error) {
	rec, err := transcriptpesc.ParseXML(doc.RawBytes)
	if err != nil {
		reason := err.Error()
		if isMaliciousParseError(err) {
			updated, uerr := transcriptsrepo.UpdateInboundAfterProcess(ctx, pool, doc.ID,
				transcriptsrepo.InboundQuarantined, nil, nil, nil, nil,
				doc.StudentName, doc.StudentDOB, doc.StudentRef, true, &reason)
			if uerr != nil {
				return nil, uerr
			}
			_ = transcriptsrepo.AppendInboundEvent(ctx, pool, doc.ID, "quarantined", nil, map[string]any{"reason": reason})
			return updated, nil
		}
		updated, uerr := transcriptsrepo.UpdateInboundAfterProcess(ctx, pool, doc.ID,
			transcriptsrepo.InboundUnmatched, nil, nil, nil, nil,
			doc.StudentName, doc.StudentDOB, doc.StudentRef, true, nil)
		if uerr != nil {
			return nil, uerr
		}
		_ = transcriptsrepo.AppendInboundEvent(ctx, pool, doc.ID, "parse_failed", nil, map[string]any{"error": reason})
		return updated, nil
	}
	parsed, _ := json.Marshal(rec)
	name := rec.Student.Name
	ref := rec.Student.StudentID
	source := ""
	if doc.SourceName != nil {
		source = *doc.SourceName
	}
	if source == "" {
		source = rec.Institution.Name
	}
	match := Match(ctx, pool, doc.OrgID, MatchHints{
		StudentName: name,
		StudentDOB:  ptrStr(doc.StudentDOB),
		StudentRef:  firstNonEmpty(ref, ptrStr(doc.StudentRef), ptrStr(doc.ExternalRef)),
		SourceName:  source,
	})
	status := transcriptsrepo.InboundParsed
	var matched *uuid.UUID
	var conf *float64
	var detail json.RawMessage
	needsManual := false
	if match != nil {
		detail, _ = json.Marshal(match)
		conf = &match.Confidence
		if match.UserID != uuid.Nil && match.Confidence >= AutoMatchMin {
			uid := match.UserID
			matched = &uid
			status = transcriptsrepo.InboundMatched
		} else {
			status = transcriptsrepo.InboundUnmatched
			needsManual = true
		}
	} else {
		status = transcriptsrepo.InboundUnmatched
		needsManual = true
	}
	updated, err := transcriptsrepo.UpdateInboundAfterProcess(ctx, pool, doc.ID, status, parsed, matched, conf, detail,
		strPtr(name), doc.StudentDOB, strPtr(firstNonEmpty(ref, ptrStr(doc.StudentRef))), needsManual, nil)
	if err != nil {
		return nil, err
	}
	_ = transcriptsrepo.AppendInboundEvent(ctx, pool, doc.ID, status, nil, map[string]any{
		"sourceName": source,
		"confidence": conf,
	})
	if matched != nil {
		notifyReceived(ctx, pool, updated)
	}
	return updated, nil
}

func processPDF(ctx context.Context, pool *pgxpool.Pool, doc *transcriptsrepo.InboundDocument) (*transcriptsrepo.InboundDocument, error) {
	// PDF-only: store + flag for manual review; extract declared metadata only.
	meta := map[string]any{
		"needsManualReview": true,
		"studentName":       ptrStr(doc.StudentName),
		"studentDob":        ptrStr(doc.StudentDOB),
		"studentRef":        ptrStr(doc.StudentRef),
		"sourceName":        ptrStr(doc.SourceName),
	}
	raw, _ := json.Marshal(meta)
	match := Match(ctx, pool, doc.OrgID, MatchHints{
		StudentName: ptrStr(doc.StudentName),
		StudentDOB:  ptrStr(doc.StudentDOB),
		StudentRef:  firstNonEmpty(ptrStr(doc.StudentRef), ptrStr(doc.ExternalRef)),
		SourceName:  ptrStr(doc.SourceName),
	})
	status := transcriptsrepo.InboundUnmatched
	var matched *uuid.UUID
	var conf *float64
	var detail json.RawMessage
	if match != nil {
		detail, _ = json.Marshal(match)
		conf = &match.Confidence
		if match.UserID != uuid.Nil && match.Confidence >= AutoMatchMin {
			uid := match.UserID
			matched = &uid
			status = transcriptsrepo.InboundMatched
		}
	}
	updated, err := transcriptsrepo.UpdateInboundAfterProcess(ctx, pool, doc.ID, status, raw, matched, conf, detail,
		doc.StudentName, doc.StudentDOB, doc.StudentRef, true, nil)
	if err != nil {
		return nil, err
	}
	_ = transcriptsrepo.AppendInboundEvent(ctx, pool, doc.ID, status, nil, map[string]any{"format": "pdf"})
	if matched != nil {
		notifyReceived(ctx, pool, updated)
	}
	return updated, nil
}

// MatchHints drives applicant matching.
type MatchHints struct {
	StudentName string
	StudentDOB  string
	StudentRef  string
	SourceName  string
}

// MatchResult is a scored candidate assignment.
type MatchResult struct {
	UserID     uuid.UUID `json:"userId"`
	Email      string    `json:"email,omitempty"`
	Confidence float64   `json:"confidence"`
	Reasons    []string  `json:"reasons"`
}

// Match scores candidates; returns nil when no signal is available.
func Match(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, hints MatchHints) *MatchResult {
	name := normalizeName(hints.StudentName)
	ref := strings.TrimSpace(hints.StudentRef)
	if name == "" && ref == "" {
		return nil
	}
	cands, err := transcriptsrepo.ListMatchCandidates(ctx, pool, orgID, hints.StudentName, ref, 40)
	if err != nil || len(cands) == 0 {
		return &MatchResult{Confidence: 0, Reasons: []string{"no_candidates"}}
	}
	var best *MatchResult
	for _, c := range cands {
		score, reasons := scoreCandidate(c, name, ref, hints.StudentDOB, hints.SourceName)
		if best == nil || score > best.Confidence {
			best = &MatchResult{UserID: c.UserID, Email: c.Email, Confidence: score, Reasons: reasons}
		}
	}
	return best
}

func scoreCandidate(c transcriptsrepo.MatchCandidate, normName, ref, dob, source string) (float64, []string) {
	var score float64
	reasons := make([]string, 0, 4)
	if ref != "" && c.SID != nil && strings.EqualFold(strings.TrimSpace(*c.SID), ref) {
		score += 0.55
		reasons = append(reasons, "sid_exact")
	}
	candName := normalizeName(candidateDisplayName(c))
	if normName != "" && candName != "" {
		if candName == normName {
			score += 0.40
			reasons = append(reasons, "name_exact")
		} else if strings.Contains(candName, normName) || strings.Contains(normName, candName) {
			score += 0.25
			reasons = append(reasons, "name_partial")
		} else {
			// last-token overlap
			cn := strings.Fields(candName)
			nn := strings.Fields(normName)
			if len(cn) > 0 && len(nn) > 0 && cn[len(cn)-1] == nn[len(nn)-1] {
				score += 0.15
				reasons = append(reasons, "lastname_match")
			}
		}
	}
	if strings.TrimSpace(dob) != "" {
		// DOB is stored encrypted on users; treat declared DOB as advisory boost only when name already matched.
		if score >= 0.25 {
			score += 0.05
			reasons = append(reasons, "dob_declared")
		}
	}
	if strings.TrimSpace(source) != "" && score > 0 {
		score += 0.02
		reasons = append(reasons, "source_present")
	}
	if score > 1 {
		score = 1
	}
	// round to 3 decimals
	score = float64(int(score*1000+0.5)) / 1000
	return score, reasons
}

func candidateDisplayName(c transcriptsrepo.MatchCandidate) string {
	if c.DisplayName != nil && strings.TrimSpace(*c.DisplayName) != "" {
		return *c.DisplayName
	}
	parts := []string{}
	if c.FirstName != nil {
		parts = append(parts, *c.FirstName)
	}
	if c.LastName != nil {
		parts = append(parts, *c.LastName)
	}
	return strings.Join(parts, " ")
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if unicode.IsSpace(r) || r == ',' || r == '-' || r == '\'' {
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// CoursesFromParsed extracts course lines for transfer-credit hand-off.
func CoursesFromParsed(parsed json.RawMessage) ([]academicrecord.CourseLine, error) {
	if len(parsed) == 0 {
		return nil, nil
	}
	var rec academicrecord.AcademicRecord
	if err := json.Unmarshal(parsed, &rec); err != nil {
		return nil, err
	}
	out := make([]academicrecord.CourseLine, 0)
	for _, term := range rec.Terms {
		out = append(out, term.Courses...)
	}
	return out, nil
}

// Accept finalizes attachment and notifies the matched user.
func Accept(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actorID *uuid.UUID) (*transcriptsrepo.InboundDocument, error) {
	doc, err := transcriptsrepo.AcceptInboundDocument(ctx, pool, id, actorID)
	if err != nil {
		return nil, err
	}
	notifyAccepted(ctx, pool, doc)
	return doc, nil
}

func notifyReceived(ctx context.Context, pool *pgxpool.Pool, doc *transcriptsrepo.InboundDocument) {
	if doc == nil || doc.MatchedUserID == nil || doc.NotifiedReceivedAt != nil {
		return
	}
	school := "another institution"
	if doc.SourceName != nil && strings.TrimSpace(*doc.SourceName) != "" {
		school = strings.TrimSpace(*doc.SourceName)
	}
	if NotifyFn != nil {
		NotifyFn(ctx, pool, *doc.MatchedUserID, "transcript_inbound_received",
			"Transcript received",
			fmt.Sprintf("A transcript from %s was received and is being reviewed for your record.", school),
			"transcript-inbound-received:"+doc.ID.String())
	}
	_ = transcriptsrepo.MarkInboundNotifiedReceived(ctx, pool, doc.ID)
}

func notifyAccepted(ctx context.Context, pool *pgxpool.Pool, doc *transcriptsrepo.InboundDocument) {
	if doc == nil || doc.MatchedUserID == nil || doc.NotifiedAcceptedAt != nil {
		return
	}
	school := "another institution"
	if doc.SourceName != nil && strings.TrimSpace(*doc.SourceName) != "" {
		school = strings.TrimSpace(*doc.SourceName)
	}
	if NotifyFn != nil {
		NotifyFn(ctx, pool, *doc.MatchedUserID, "transcript_inbound_accepted",
			"Transcript accepted",
			fmt.Sprintf("Your transcript from %s has been accepted and is available for transfer-credit evaluation.", school),
			"transcript-inbound-accepted:"+doc.ID.String())
	}
	_ = transcriptsrepo.MarkInboundNotifiedAccepted(ctx, pool, doc.ID)
}

func inferFormat(contentType string, raw []byte) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(ct, "xml") || strings.Contains(ct, "pesc"):
		return transcriptsrepo.InboundFormatPESC
	case strings.Contains(ct, "pdf"):
		return transcriptsrepo.InboundFormatPDF
	case strings.Contains(ct, "edi") || strings.Contains(ct, "x12"):
		return transcriptsrepo.InboundFormatEDI
	}
	trim := bytes.TrimSpace(raw)
	if bytes.HasPrefix(trim, []byte("%PDF")) {
		return transcriptsrepo.InboundFormatPDF
	}
	if bytes.Contains(trim[:min(len(trim), 256)], []byte("CollegeTranscript")) || bytes.HasPrefix(trim, []byte("<?xml")) || bytes.HasPrefix(trim, []byte("<")) {
		return transcriptsrepo.InboundFormatPESC
	}
	return transcriptsrepo.InboundFormatOther
}

func validateContent(format, contentType string, raw []byte) error {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct != "" && !allowedContentType(ct, format) {
		return ErrUnsupportedType
	}
	if unsafe, _ := detectUnsafe(raw, format); unsafe {
		return ErrUnsafePayload
	}
	return nil
}

func allowedContentType(ct, format string) bool {
	switch format {
	case transcriptsrepo.InboundFormatPESC:
		return strings.Contains(ct, "xml") || strings.Contains(ct, "text") || strings.Contains(ct, "octet-stream")
	case transcriptsrepo.InboundFormatPDF:
		return strings.Contains(ct, "pdf") || strings.Contains(ct, "octet-stream")
	default:
		return strings.Contains(ct, "octet-stream") || strings.Contains(ct, "text") || strings.Contains(ct, "xml") || strings.Contains(ct, "pdf")
	}
}

func detectUnsafe(raw []byte, format string) (bool, string) {
	if len(raw) == 0 {
		return true, "empty"
	}
	lower := bytes.ToLower(raw)
	// XXE / DTD / entity expansion probes (defense in depth before parser).
	if format == transcriptsrepo.InboundFormatPESC || bytes.Contains(lower[:min(len(lower), 4096)], []byte("<!doctype")) || bytes.Contains(lower[:min(len(lower), 4096)], []byte("<!entity")) {
		head := lower[:min(len(lower), 8192)]
		if bytes.Contains(head, []byte("<!entity")) || bytes.Contains(head, []byte("system \"")) || bytes.Contains(head, []byte("system '")) ||
			bytes.Contains(head, []byte("expect://")) || bytes.Contains(head, []byte("file://")) || bytes.Contains(head, []byte("php://")) {
			return true, "xxe_or_external_entity"
		}
	}
	// Zip-bomb-ish: PDF with absurd object streams is still accepted but oversized already capped.
	if format == transcriptsrepo.InboundFormatPDF && !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("%PDF")) {
		return true, "invalid_pdf_magic"
	}
	return false, ""
}

func isMaliciousParseError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "entity") || strings.Contains(msg, "charset")
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// SniffContentType returns a best-effort MIME for responses.
func SniffContentType(format string, raw []byte) string {
	switch format {
	case transcriptsrepo.InboundFormatPESC:
		return "application/xml"
	case transcriptsrepo.InboundFormatPDF:
		return "application/pdf"
	default:
		if len(raw) > 0 {
			return http.DetectContentType(raw)
		}
		return "application/octet-stream"
	}
}
