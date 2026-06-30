package csvimport

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

const batchSize = 500

// RowOutcome is the per-row result written to the result CSV.
type RowOutcome struct {
	RowNumber int
	Email     string
	Outcome   string // created, updated, deactivated, skipped, error
	Detail    string
}

// ProcessParams drives one import execution.
type ProcessParams struct {
	JobID         uuid.UUID
	OrgID         uuid.UUID
	ActorID       uuid.UUID
	MergeStrategy MergeStrategy
	DryRun        bool
	Rows          []ParsedRow
	CursorRow     int // 0-based index into Rows to resume from
	OnProgress    func(processed, errors int)
}

// ProcessResult summarizes an import run.
type ProcessResult struct {
	Outcomes         []RowOutcome
	Errors           []RowError
	ProcessedRows    int
	ErrorRows        int
	CreatedCount     int
	UpdatedCount     int
	DeactivatedCount int
	SkippedCount     int
	SeenExternalIDs  map[string]struct{}
	SeenEmails       map[string]struct{}
}

// Process applies merge logic to parsed rows starting at CursorRow.
func Process(ctx context.Context, pool *pgxpool.Pool, p ProcessParams) (*ProcessResult, error) {
	res := &ProcessResult{
		SeenExternalIDs: make(map[string]struct{}),
		SeenEmails:      make(map[string]struct{}),
	}
	start := p.CursorRow
	if start < 0 {
		start = 0
	}
	if start > len(p.Rows) {
		start = len(p.Rows)
	}

	for i := start; i < len(p.Rows); i++ {
		row := p.Rows[i]
		outcome, rowErrs := processOneRow(ctx, pool, p, row)
		res.Outcomes = append(res.Outcomes, outcome)
		if len(rowErrs) > 0 {
			res.Errors = append(res.Errors, rowErrs...)
			res.ErrorRows++
		}
		res.ProcessedRows++
		switch outcome.Outcome {
		case "created":
			res.CreatedCount++
		case "updated":
			res.UpdatedCount++
		case "deactivated":
			res.DeactivatedCount++
		case "skipped":
			res.SkippedCount++
		}
		if row.ExternalID != "" {
			res.SeenExternalIDs[row.ExternalID] = struct{}{}
		}
		res.SeenEmails[row.Email] = struct{}{}

		if p.OnProgress != nil && (i-start+1)%batchSize == 0 {
			p.OnProgress(res.ProcessedRows, res.ErrorRows)
		}
	}

	if !p.DryRun && p.MergeStrategy == MergeSync {
		deactivated, outcomes, err := deactivateMissing(ctx, pool, p, res.SeenExternalIDs, res.SeenEmails)
		if err != nil {
			return res, err
		}
		res.DeactivatedCount += deactivated
		res.Outcomes = append(res.Outcomes, outcomes...)
		res.ProcessedRows += len(outcomes)
	}

	if p.OnProgress != nil {
		p.OnProgress(res.ProcessedRows, res.ErrorRows)
	}
	return res, nil
}

func processOneRow(ctx context.Context, pool *pgxpool.Pool, p ProcessParams, row ParsedRow) (RowOutcome, []RowError) {
	outcome := RowOutcome{RowNumber: row.RowNumber, Email: row.Email}

	existing, lookupErr := lookupUser(ctx, pool, p.OrgID, row)
	if lookupErr != nil {
		if lookupErr == errOrgIsolation {
			return RowOutcome{RowNumber: row.RowNumber, Email: row.Email, Outcome: "error", Detail: "org_isolation_violation"},
				[]RowError{{Row: row.RowNumber, Column: "email", Message: "user belongs to another organization", Code: "org_isolation_violation"}}
		}
		return RowOutcome{RowNumber: row.RowNumber, Email: row.Email, Outcome: "error", Detail: lookupErr.Error()},
			[]RowError{{Row: row.RowNumber, Column: "email", Message: lookupErr.Error(), Code: "internal"}}
	}

	if existing == nil {
		if p.MergeStrategy == MergeCreateOnly || p.MergeStrategy == MergeUpsert || p.MergeStrategy == MergeSync {
			if p.DryRun {
				outcome.Outcome = "created"
				return outcome, nil
			}
			uid, err := createUser(ctx, pool, p.OrgID, p.ActorID, row)
			if err != nil {
				return RowOutcome{RowNumber: row.RowNumber, Email: row.Email, Outcome: "error", Detail: err.Error()},
					[]RowError{{Row: row.RowNumber, Column: "email", Message: err.Error(), Code: "create_failed"}}
			}
			outcome.Outcome = "created"
			_ = recordUserAudit(ctx, pool, p.OrgID, p.ActorID, adminaudit.EventUserCreate, uid, nil, rowSnapshot(row))
			return outcome, nil
		}
		outcome.Outcome = "skipped"
		outcome.Detail = "user not found"
		return outcome, nil
	}

	if p.MergeStrategy == MergeCreateOnly {
		outcome.Outcome = "skipped"
		outcome.Detail = "already exists"
		return outcome, nil
	}

	if p.DryRun {
		outcome.Outcome = "updated"
		return outcome, nil
	}

	before := existingSnapshot(existing)
	if err := updateUser(ctx, pool, p.OrgID, existing.ID, row); err != nil {
		return RowOutcome{RowNumber: row.RowNumber, Email: row.Email, Outcome: "error", Detail: err.Error()},
			[]RowError{{Row: row.RowNumber, Column: "email", Message: err.Error(), Code: "update_failed"}}
	}
	outcome.Outcome = "updated"
	_ = recordUserAudit(ctx, pool, p.OrgID, p.ActorID, adminaudit.EventUserUpdate, existing.ID, before, rowSnapshot(row))
	return outcome, nil
}

var errOrgIsolation = fmt.Errorf("org isolation violation")

type existingUser struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	Email         string
	FirstName     string
	LastName      string
	ExternalID    string
	Deactivated   bool
}

func lookupUser(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, row ParsedRow) (*existingUser, error) {
	if row.ExternalID != "" {
		u, err := loadUserByExternalID(ctx, pool, row.ExternalID)
		if err != nil {
			return nil, err
		}
		if u != nil {
			if u.OrgID != orgID {
				return nil, errOrgIsolation
			}
			return u, nil
		}
	}
	u, err := loadUserByEmail(ctx, pool, row.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	if u.OrgID != orgID {
		return nil, errOrgIsolation
	}
	return u, nil
}

func loadUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*existingUser, error) {
	var u existingUser
	var extID *string
	var deactivated bool
	err := pool.QueryRow(ctx, `
SELECT id, org_id, email, COALESCE(first_name,''), COALESCE(last_name,''), external_id,
       (deactivated_at IS NOT NULL OR login_blocked)
FROM "user".users WHERE email = $1
`, email).Scan(&u.ID, &u.OrgID, &u.Email, &u.FirstName, &u.LastName, &extID, &deactivated)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if extID != nil {
		u.ExternalID = *extID
	}
	u.Deactivated = deactivated
	return &u, nil
}

func loadUserByExternalID(ctx context.Context, pool *pgxpool.Pool, externalID string) (*existingUser, error) {
	var u existingUser
	var extID *string
	var deactivated bool
	err := pool.QueryRow(ctx, `
SELECT id, org_id, email, COALESCE(first_name,''), COALESCE(last_name,''), external_id,
       (deactivated_at IS NOT NULL OR login_blocked)
FROM "user".users WHERE external_id = $1
`, externalID).Scan(&u.ID, &u.OrgID, &u.Email, &u.FirstName, &u.LastName, &extID, &deactivated)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if extID != nil {
		u.ExternalID = *extID
	}
	u.Deactivated = deactivated
	return &u, nil
}

func createUser(ctx context.Context, pool *pgxpool.Pool, orgID, actorID uuid.UUID, row ParsedRow) (uuid.UUID, error) {
	ph, err := authservice.PlaceholderPasswordHash()
	if err != nil {
		return uuid.Nil, err
	}
	dn := strings.TrimSpace(row.FirstName + " " + row.LastName)
	if dn == "" {
		dn = row.Email
	}
	var ext any
	if row.ExternalID != "" {
		ext = row.ExternalID
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var uid uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO "user".users (email, password_hash, display_name, first_name, last_name, external_id, org_id)
VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), $6, $7)
RETURNING id
`, row.Email, ph, dn, row.FirstName, row.LastName, ext, orgID).Scan(&uid)
	if err != nil {
		return uuid.Nil, err
	}
	appRole := MapRoleToAppRole(row.Role)
	if _, err := rbac.AssignUserRoleFromProvisioningMapTx(ctx, tx, uid, "csv_import", row.Role, appRole); err != nil {
		return uuid.Nil, err
	}
	if err := rbac.AssignUserRoleByNameTx(ctx, tx, uid, appRole); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	if IsOrgAdminRole(row.Role) {
		if _, err := orgroles.Create(ctx, pool, orgID, uid, nil, orgroles.RoleOrgAdmin, &actorID, nil); err != nil {
			return uuid.Nil, err
		}
	}
	return uid, nil
}

func updateUser(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, userID uuid.UUID, row ParsedRow) error {
	dn := strings.TrimSpace(row.FirstName + " " + row.LastName)
	if dn == "" {
		dn = row.Email
	}
	var ext any
	if row.ExternalID != "" {
		ext = row.ExternalID
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
UPDATE "user".users SET
  email = $2,
  first_name = NULLIF($3,''),
  last_name = NULLIF($4,''),
  display_name = NULLIF($5,''),
  external_id = COALESCE($6, external_id),
  deactivated_at = NULL,
  login_blocked = FALSE,
  org_id = $7
WHERE id = $1
`, userID, row.Email, row.FirstName, row.LastName, dn, ext, orgID)
	if err != nil {
		return err
	}
	appRole := MapRoleToAppRole(row.Role)
	if _, err := rbac.AssignUserRoleFromProvisioningMapTx(ctx, tx, userID, "csv_import", row.Role, appRole); err != nil {
		return err
	}
	if err := rbac.AssignUserRoleByNameTx(ctx, tx, userID, appRole); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func deactivateMissing(ctx context.Context, pool *pgxpool.Pool, p ProcessParams, seenExt, seenEmail map[string]struct{}) (int, []RowOutcome, error) {
	rows, err := pool.Query(ctx, `
SELECT id, email, COALESCE(external_id,''), (deactivated_at IS NOT NULL OR login_blocked)
FROM "user".users
WHERE org_id = $1 AND account_type = 'standard'
`, p.OrgID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var outcomes []RowOutcome
	var count int
	for rows.Next() {
		var uid uuid.UUID
		var email, extID string
		var deactivated bool
		if err := rows.Scan(&uid, &email, &extID, &deactivated); err != nil {
			return count, outcomes, err
		}
		if deactivated {
			continue
		}
		inCSV := false
		if extID != "" {
			_, inCSV = seenExt[extID]
		} else {
			_, inCSV = seenEmail[email]
		}
		if inCSV {
			continue
		}
		if _, err := pool.Exec(ctx, `
UPDATE "user".users SET deactivated_at = COALESCE(deactivated_at, NOW()), login_blocked = TRUE WHERE id = $1
`, uid); err != nil {
			return count, outcomes, err
		}
		_ = recordUserAudit(ctx, pool, p.OrgID, p.ActorID, adminaudit.EventUserDeactivate, uid, nil, nil)
		outcomes = append(outcomes, RowOutcome{Email: email, Outcome: "deactivated"})
		count++
	}
	return count, outcomes, rows.Err()
}

func existingSnapshot(u *existingUser) []byte {
	return []byte(fmt.Sprintf(`{"email":%q,"firstName":%q,"lastName":%q,"externalId":%q}`,
		u.Email, u.FirstName, u.LastName, u.ExternalID))
}

func rowSnapshot(row ParsedRow) []byte {
	return []byte(fmt.Sprintf(`{"email":%q,"firstName":%q,"lastName":%q,"role":%q,"externalId":%q}`,
		row.Email, row.FirstName, row.LastName, row.Role, row.ExternalID))
}

func recordUserAudit(ctx context.Context, pool *pgxpool.Pool, orgID, actorID uuid.UUID, eventType string, targetID uuid.UUID, before, after []byte) error {
	targetType := "user"
	_, err := adminaudit.Record(ctx, pool, adminaudit.RecordParams{
		OrgID:       &orgID,
		EventType:   eventType,
		ActorID:     actorID,
		TargetType:  &targetType,
		TargetID:    &targetID,
		BeforeValue: before,
		AfterValue:  after,
	})
	return err
}

// WriteResultCSV writes per-row outcomes to path.
func WriteResultCSV(path string, outcomes []RowOutcome) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"row", "email", "outcome", "detail"}); err != nil {
		return err
	}
	for _, o := range outcomes {
		row := ""
		if o.RowNumber > 0 {
			row = fmt.Sprintf("%d", o.RowNumber)
		}
		if err := w.Write([]string{row, o.Email, o.Outcome, o.Detail}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// NormalizeEmail delegates to user package.
func NormalizeEmail(email string) string {
	return user.NormalizeEmail(email)
}
