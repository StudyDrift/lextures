// Package stateprivacy implements state-specific student data privacy controls for
// CA SOPIPA (Cal. Ed. Code §§ 49073.1–49073.6), NY Ed Law 2-d (§ 2-d(1)–(7) and
// 8 NYCRR Part 121), and IL SOPPA (105 ILCS 85/ §§ 5–30) — plan 10.6.
package stateprivacy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/rbac"
	repo "github.com/lextures/lextures/server/internal/repos/stateprivacy"
)

// AdminPermission gates state-privacy admin actions (approve/deny requests, set jurisdiction).
const AdminPermission = "compliance:stateprivacy:admin:*"

// DeletionDeadlineDays is the IL SOPPA statutory response period (105 ILCS 85/25(c)).
// We use calendar days as a safe fallback per plan 10.6 §14.
const DeletionDeadlineDays = 30

// DeletionDeadlineWarning is the window before due_at at which an escalation is sent.
const DeletionDeadlineWarning = 5 * 24 * time.Hour

// Valid jurisdictions.
const (
	JurisdictionCA = "CA"
	JurisdictionNY = "NY"
	JurisdictionIL = "IL"
)

var (
	ErrNotFound            = errors.New("stateprivacy: record not found")
	ErrAlreadyExists       = errors.New("stateprivacy: active request already exists")
	ErrForbidden           = errors.New("stateprivacy: forbidden")
	ErrInvalidJurisdiction = errors.New("stateprivacy: invalid jurisdiction; must be CA, NY, or IL")
)

// CheckAdmin returns true when the user holds the compliance:stateprivacy:admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// GetOrgJurisdiction returns the state jurisdiction tag for an org, or "" if unset.
func GetOrgJurisdiction(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (string, error) {
	j, err := repo.OrgJurisdiction(ctx, pool, orgID)
	if err != nil {
		return "", fmt.Errorf("stateprivacy: get jurisdiction: %w", err)
	}
	return j, nil
}

// GetParentDisclosure returns disclosure events for a student for the current and prior school year.
// schoolYearStart should be set to the start of the prior school year (approximately 18 months ago).
func GetParentDisclosure(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]repo.DisclosureEvent, error) {
	// Expose current and prior school year (approx. 18 months).
	cutoff := time.Now().UTC().AddDate(-1, -6, 0)
	events, err := repo.ListDisclosureEvents(ctx, pool, studentID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("stateprivacy: get disclosure: %w", err)
	}
	return events, nil
}

// SubmitDeletionRequest creates an IL SOPPA data-deletion request.
// Returns ErrAlreadyExists when a pending/in-progress request already exists for the student.
func SubmitDeletionRequest(ctx context.Context, pool *pgxpool.Pool, orgID, studentID uuid.UUID, requesterID *uuid.UUID, requesterEmail string) (uuid.UUID, error) {
	existing, err := repo.ListDeletionRequestsForStudent(ctx, pool, studentID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("stateprivacy: check existing deletion requests: %w", err)
	}
	for _, r := range existing {
		if r.Status == "pending" || r.Status == "in_progress" {
			return uuid.UUID{}, ErrAlreadyExists
		}
	}
	id, err := repo.InsertDeletionRequest(ctx, pool, orgID, studentID, requesterID, requesterEmail)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("stateprivacy: submit deletion request: %w", err)
	}
	return id, nil
}

// GetDeletionRequest returns a deletion request by ID, or ErrNotFound.
func GetDeletionRequest(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*repo.DeletionRequest, error) {
	r, err := repo.GetDeletionRequest(ctx, pool, id)
	if err != nil {
		return nil, fmt.Errorf("stateprivacy: get deletion request: %w", err)
	}
	if r == nil {
		return nil, ErrNotFound
	}
	return r, nil
}

// ApproveDeletionRequest marks a request as completed.
func ApproveDeletionRequest(ctx context.Context, pool *pgxpool.Pool, id, adminID uuid.UUID, notes string) error {
	r, err := repo.GetDeletionRequest(ctx, pool, id)
	if err != nil {
		return fmt.Errorf("stateprivacy: get deletion request: %w", err)
	}
	if r == nil {
		return ErrNotFound
	}
	var n *string
	if notes != "" {
		n = &notes
	}
	return repo.UpdateDeletionRequestStatus(ctx, pool, id, adminID, "completed", n)
}

// DenyDeletionRequest marks a request as denied with a reason.
func DenyDeletionRequest(ctx context.Context, pool *pgxpool.Pool, id, adminID uuid.UUID, reason string) error {
	r, err := repo.GetDeletionRequest(ctx, pool, id)
	if err != nil {
		return fmt.Errorf("stateprivacy: get deletion request: %w", err)
	}
	if r == nil {
		return ErrNotFound
	}
	return repo.UpdateDeletionRequestStatus(ctx, pool, id, adminID, "denied", &reason)
}

// GetAnnualNoticeStatus returns whether the annual notice was sent for an org/year.
func GetAnnualNoticeStatus(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, jurisdiction string, year int) (sent bool, sentAt *time.Time, err error) {
	j, err := repo.GetAnnualNoticeJob(ctx, pool, orgID, jurisdiction, year)
	if err != nil {
		return false, nil, fmt.Errorf("stateprivacy: get annual notice status: %w", err)
	}
	if j == nil || j.SentAt == nil {
		return false, nil, nil
	}
	return true, j.SentAt, nil
}

// ComplianceChecklist returns the outstanding state obligations for an org.
func ComplianceChecklist(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]ChecklistItem, error) {
	jurisdiction, err := repo.OrgJurisdiction(ctx, pool, orgID)
	if err != nil {
		return nil, fmt.Errorf("stateprivacy: checklist jurisdiction: %w", err)
	}
	if jurisdiction == "" {
		return []ChecklistItem{}, nil
	}

	var items []ChecklistItem
	currentYear := time.Now().Year()

	switch jurisdiction {
	case JurisdictionCA:
		items = append(items, caChecklist()...)
	case JurisdictionNY:
		nyItems, err := nyChecklist(ctx, pool, orgID, currentYear)
		if err != nil {
			return nil, err
		}
		items = append(items, nyItems...)
	case JurisdictionIL:
		ilItems, err := ilChecklist(ctx, pool, orgID)
		if err != nil {
			return nil, err
		}
		items = append(items, ilItems...)
	}

	return items, nil
}

// ChecklistItem represents one compliance obligation and its completion status.
type ChecklistItem struct {
	ID           string `json:"id"`
	Jurisdiction string `json:"jurisdiction"`
	Obligation   string `json:"obligation"`
	Statute      string `json:"statute"`
	Status       string `json:"status"` // "met" | "outstanding" | "not_applicable"
	GuidanceURL  string `json:"guidanceUrl,omitempty"`
}

// DPAAddendum returns the state-specific DPA addendum content for a given state.
func DPAAddendum(jurisdiction string) (*DPAContent, error) {
	switch jurisdiction {
	case JurisdictionCA:
		return caAddendum(), nil
	case JurisdictionNY:
		return nyAddendum(), nil
	case JurisdictionIL:
		return ilAddendum(), nil
	default:
		return nil, ErrInvalidJurisdiction
	}
}

// DPAContent holds the state-specific DPA addendum exhibit content.
type DPAContent struct {
	Jurisdiction string       `json:"jurisdiction"`
	StatuteName  string       `json:"statuteName"`
	StatuteCite  string       `json:"statuteCite"`
	Prohibitions []string     `json:"prohibitions"`
	ParentRights []string     `json:"parentRights"`
	Exhibits     []DPAExhibit `json:"exhibits"`
}

// DPAExhibit is one named exhibit within a state DPA addendum.
type DPAExhibit struct {
	Name    string `json:"name"`
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

// ProhibitionAttestation returns the platform-wide prohibitions shared by all three laws.
func ProhibitionAttestation() []string {
	return []string{
		"Lextures does not use student data for targeted advertising or to build advertising profiles.",
		"Lextures does not sell or rent student personal information for any commercial purpose.",
		"Lextures does not use student data for any purpose other than providing the contracted educational service.",
		"Lextures does not disclose student data to third parties except as required to provide the service or as required by law.",
	}
}

func caChecklist() []ChecklistItem {
	return []ChecklistItem{
		{
			ID:           "ca-sopipa-prohibition",
			Jurisdiction: JurisdictionCA,
			Obligation:   "Publish prohibition attestation (no targeted advertising, no sale of student data)",
			Statute:      "Cal. Ed. Code § 49073.1(b)(1)–(3)",
			Status:       "met",
		},
		{
			ID:           "ca-sopipa-sub-processor-disclosure",
			Jurisdiction: JurisdictionCA,
			Obligation:   "Maintain and publish sub-processor disclosure log accessible to parents",
			Statute:      "Cal. Ed. Code § 49073.1(b)(7)",
			Status:       "met",
		},
		{
			ID:           "ca-sopipa-dpa",
			Jurisdiction: JurisdictionCA,
			Obligation:   "Execute CA SOPIPA DPA addendum with district",
			Statute:      "Cal. Ed. Code § 49073.1",
			Status:       "outstanding",
		},
	}
}

func nyChecklist(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, year int) ([]ChecklistItem, error) {
	sent, _, err := GetAnnualNoticeStatus(ctx, pool, orgID, JurisdictionNY, year)
	if err != nil {
		return nil, err
	}
	noticeStatus := "outstanding"
	if sent {
		noticeStatus = "met"
	}

	overdue, err := repo.CountOverdueDeletionRequests(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("stateprivacy: ny checklist overdue: %w", err)
	}
	deletionStatus := "met"
	if overdue > 0 {
		deletionStatus = "outstanding"
	}

	return []ChecklistItem{
		{
			ID:           "ny-annual-notice",
			Jurisdiction: JurisdictionNY,
			Obligation:   fmt.Sprintf("Send annual Ed Law 2-d parent notice for school year %d", year),
			Statute:      "N.Y. Education Law § 2-d(5)(a); 8 NYCRR Part 121.3",
			Status:       noticeStatus,
		},
		{
			ID:           "ny-cpo-appointment",
			Jurisdiction: JurisdictionNY,
			Obligation:   "Confirm Chief Privacy Officer appointment on file with NYSED",
			Statute:      "N.Y. Education Law § 2-d(3)",
			Status:       "outstanding",
			GuidanceURL:  "https://www.nysed.gov/data-privacy-security/chief-privacy-officer",
		},
		{
			ID:           "ny-prohibition-attestation",
			Jurisdiction: JurisdictionNY,
			Obligation:   "Publish prohibition attestation (no targeted advertising, no sale of student data)",
			Statute:      "N.Y. Education Law § 2-d(5)(b)",
			Status:       "met",
		},
		{
			ID:           "ny-deletion-sla",
			Jurisdiction: JurisdictionNY,
			Obligation:   "No overdue parent data-deletion requests",
			Statute:      "N.Y. Education Law § 2-d(5)(d)",
			Status:       deletionStatus,
		},
	}, nil
}

func ilChecklist(ctx context.Context, pool *pgxpool.Pool, _ uuid.UUID) ([]ChecklistItem, error) {
	overdue, err := repo.CountOverdueDeletionRequests(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("stateprivacy: il checklist overdue: %w", err)
	}
	deletionStatus := "met"
	if overdue > 0 {
		deletionStatus = "outstanding"
	}

	return []ChecklistItem{
		{
			ID:           "il-soppa-deletion-sla",
			Jurisdiction: JurisdictionIL,
			Obligation:   "Fulfill parent data-deletion requests within 30 days",
			Statute:      "105 ILCS 85/25(c)",
			Status:       deletionStatus,
		},
		{
			ID:           "il-soppa-prohibition",
			Jurisdiction: JurisdictionIL,
			Obligation:   "Publish prohibition attestation (no targeted advertising, no sale of student data)",
			Statute:      "105 ILCS 85/10",
			Status:       "met",
		},
		{
			ID:           "il-soppa-nist",
			Jurisdiction: JurisdictionIL,
			Obligation:   "Maintain data security program consistent with NIST CSF",
			Statute:      "105 ILCS 85/15",
			Status:       "met",
		},
	}, nil
}

func caAddendum() *DPAContent {
	return &DPAContent{
		Jurisdiction: JurisdictionCA,
		StatuteName:  "Student Online Personal Information Protection Act (SOPIPA)",
		StatuteCite:  "Cal. Ed. Code §§ 49073.1–49073.6",
		Prohibitions: ProhibitionAttestation(),
		ParentRights: []string{
			"Parents may request a list of sub-processors and third parties with access to their child's student data.",
			"Parents may request deletion of their child's student data upon withdrawal from the school.",
		},
		Exhibits: []DPAExhibit{
			{
				Name:    "Exhibit CA-1",
				Heading: "SOPIPA Prohibition Attestations",
				Body:    "Operator attests that it shall not: (a) engage in targeted advertising on the Covered Site or Service, or target advertising based on any information acquired through the use of the Covered Site or Service; (b) use information created or gathered through the Covered Site or Service to amass a profile about a K–12 student; (c) sell a student's information, including without limitation, selling, renting, releasing, disclosing, disseminating, making available, transferring, or otherwise communicating orally, in writing, or by electronic or other means, a student's information to a third party for monetary or other valuable consideration.",
			},
			{
				Name:    "Exhibit CA-2",
				Heading: "Sub-Processor Disclosure",
				Body:    "Operator maintains a current list of sub-processors accessible to parents via the platform's compliance disclosure page. Any material change to sub-processors will be communicated to the district within 30 days.",
			},
		},
	}
}

func nyAddendum() *DPAContent {
	return &DPAContent{
		Jurisdiction: JurisdictionNY,
		StatuteName:  "New York Education Law 2-d",
		StatuteCite:  "N.Y. Education Law § 2-d; 8 NYCRR Part 121",
		Prohibitions: ProhibitionAttestation(),
		ParentRights: []string{
			"Parents have the right to inspect and review their child's student data.",
			"Parents have the right to request correction of inaccurate student data.",
			"Parents have the right to submit a complaint regarding unauthorized disclosure of student data.",
			"Parents will receive an annual written notice describing student data collected and how it is used.",
		},
		Exhibits: []DPAExhibit{
			{
				Name:    "Exhibit NY-1",
				Heading: "Ed Law 2-d Supplemental Information",
				Body:    "Pursuant to 8 NYCRR Part 121.3, the following supplemental information is provided: (a) the exclusive purposes for which student data will be used are educational services as defined in the Agreement; (b) whether and how a parent, student, eligible student, teacher or principal may challenge the accuracy of the student data; (c) where the student data will be stored; (d) the schedule for destruction or return of student data; and (e) the contact information of the Chief Privacy Officer of the educational agency.",
			},
			{
				Name:    "Exhibit NY-2",
				Heading: "Annual Parent Notice",
				Body:    "Operator agrees to provide parents/guardians of students enrolled in districts using this service with an annual written notice describing the categories of student data held, the purposes for which it is used, the identity of any third parties with access, and the parents' rights under Ed Law 2-d. The notice will be provided in English, Spanish, and, where required, Chinese.",
			},
		},
	}
}

func ilAddendum() *DPAContent {
	return &DPAContent{
		Jurisdiction: JurisdictionIL,
		StatuteName:  "Student Online Personal Protection Act (SOPPA)",
		StatuteCite:  "105 ILCS 85/ §§ 5–30",
		Prohibitions: ProhibitionAttestation(),
		ParentRights: []string{
			"Parents have the right to request deletion of their child's covered information within 30 days.",
			"Parents will receive written confirmation of deletion upon completion.",
		},
		Exhibits: []DPAExhibit{
			{
				Name:    "Exhibit IL-1",
				Heading: "SOPPA Prohibition Attestations",
				Body:    "Operator attests it shall not: (a) engage in targeted advertising based on student covered information; (b) build a profile on a student using covered information for any purpose other than providing the educational service; (c) sell, rent, or trade covered information; (d) disclose covered information except as required to provide the service or as required by law.",
			},
			{
				Name:    "Exhibit IL-2",
				Heading: "Data Deletion Procedure",
				Body:    "Upon receipt of a verified parent data-deletion request, Operator will fulfill the request within 30 calendar days and provide written confirmation to the requesting parent or guardian. Operator maintains a NIST Cybersecurity Framework-aligned data security program as required by 105 ILCS 85/15.",
			},
		},
	}
}
