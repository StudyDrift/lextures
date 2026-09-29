// Managed learner identity helpers (Homeschool #640).
package user

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const managedEmailDomain = "managed.lextures.invalid"

// ManagedEmail returns the synthetic email for a managed learner user id.
// Pattern matches system identities (@system.lextures.invalid); never shown in parent UI.
func ManagedEmail(userID uuid.UUID) string {
	return fmt.Sprintf("%s@%s", userID.String(), managedEmailDomain)
}

// IsManagedEmail reports whether email uses the managed.lextures.invalid domain.
func IsManagedEmail(email string) bool {
	em := strings.ToLower(strings.TrimSpace(email))
	return strings.HasSuffix(em, "@"+managedEmailDomain)
}
