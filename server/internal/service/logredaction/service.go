// Package logredaction implements ops visibility for PII log redaction (plan 10.14).
package logredaction

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// ReadPermission gates the redaction status endpoint.
const ReadPermission = "compliance:redaction:read:*"

// Status is returned by GET /api/v1/internal/ops/redaction-status.
type Status struct {
	RedactionEnabled   bool              `json:"redactionEnabled"`
	DisableRedaction   bool              `json:"disableRedaction"`
	AppEnv             string            `json:"appEnv"`
	RegisteredFields   []string          `json:"registeredFields"`
	PIIRedactionsTotal map[string]uint64 `json:"piiRedactionsTotal"`
}

// CheckRead returns true when the user may read redaction status.
func CheckRead(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, ReadPermission)
}

// BuildStatus assembles the current redaction configuration and metrics.
func BuildStatus(disableRedaction bool, appEnv string, extraFields []string) Status {
	reg := logging.NewFieldRegistry(extraFields...)
	return Status{
		RedactionEnabled:   !disableRedaction,
		DisableRedaction:   disableRedaction,
		AppEnv:             appEnv,
		RegisteredFields:   reg.Names(),
		PIIRedactionsTotal: logging.GlobalRedactionMetrics.Snapshot(),
	}
}
