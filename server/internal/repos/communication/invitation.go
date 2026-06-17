package communication

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	models "github.com/lextures/lextures/server/internal/models/communication"
)

// SendEnrollmentInvitationMessage delivers an inbox invitation with Approve/Decline actions.
func SendEnrollmentInvitationMessage(
	ctx context.Context,
	pool *pgxpool.Pool,
	recipientEmail string,
	courseCode string,
	courseTitle string,
	enrollmentID uuid.UUID,
) (*uuid.UUID, error) {
	subject := fmt.Sprintf("Invitation to %s", courseTitle)
	body := fmt.Sprintf(`You have been invited to join the course "%s" (%s).

Please approve or decline this invitation below. If you approve, your enrollment will be activated and you can access the course. If you decline, the invitation will be removed.`, courseTitle, courseCode)
	meta := models.MessageMetadata{
		Type:         "enrollment_invitation",
		EnrollmentID: enrollmentID.String(),
		CourseCode:   courseCode,
		CourseTitle:  courseTitle,
		Actions: []models.MessageAction{
			{ID: "approve", Label: "Approve", Style: "primary"},
			{ID: "decline", Label: "Decline", Style: "danger"},
		},
	}
	return SendMessageWithMetadata(ctx, pool, PlatformInboxSenderID, recipientEmail, subject, body, &meta)
}

// ResolveEnrollmentInvitationMessages marks open invitation messages resolved for an enrollment.
func ResolveEnrollmentInvitationMessages(ctx context.Context, pool *pgxpool.Pool, recipientID, enrollmentID uuid.UUID, resolved string) error {
	metaPatch, err := json.Marshal(map[string]string{"resolved": resolved})
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE communication.messages
SET metadata = COALESCE(metadata, '{}'::jsonb) || $3::jsonb
WHERE recipient_user_id = $1
  AND metadata->>'type' = 'enrollment_invitation'
  AND metadata->>'enrollmentId' = $2
  AND (metadata->>'resolved' IS NULL OR metadata->>'resolved' = '')
`, recipientID, enrollmentID.String(), string(metaPatch))
	return err
}