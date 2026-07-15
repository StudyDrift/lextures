// Package institutioninquiry stores marketing "Request information" leads.
// Write-only from the public endpoint for now (email hook comes later).
package institutioninquiry

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Inquiry is one institution_inquiries row for insert.
type Inquiry struct {
	OrganizationType  string
	OrganizationName  string
	ContactName       string
	Email             string
	Role              *string
	EnrollmentSize    string
	HostingPreference string
	Message           string
	IPAddress         *string
	UserAgent         *string
}

// Insert persists one lead and returns its id.
func Insert(ctx context.Context, db *pgxpool.Pool, in Inquiry) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO institution_inquiries (
			organization_type, organization_name, contact_name, email, role,
			enrollment_size, hosting_preference, message, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`,
		in.OrganizationType,
		in.OrganizationName,
		in.ContactName,
		strings.ToLower(strings.TrimSpace(in.Email)),
		in.Role,
		in.EnrollmentSize,
		in.HostingPreference,
		in.Message,
		in.IPAddress,
		in.UserAgent,
	).Scan(&id)
	return id, err
}
