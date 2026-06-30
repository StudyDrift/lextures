package csvimport

import "fmt"

// Profile identifies a CSV column layout.
type Profile string

const (
	ProfileLexturesNative Profile = "lextures_native"
	ProfileOneRosterV12   Profile = "oneroster_v1.2"
)

// ParseProfile normalizes a profile query/form value.
func ParseProfile(s string) (Profile, error) {
	switch Profile(s) {
	case ProfileLexturesNative, ProfileOneRosterV12:
		return Profile(s), nil
	case "":
		return ProfileLexturesNative, nil
	default:
		return "", fmt.Errorf("unknown import profile %q", s)
	}
}

// ColumnMap returns logical field -> CSV header names for a profile.
func (p Profile) ColumnMap() map[string][]string {
	switch p {
	case ProfileOneRosterV12:
		return map[string][]string{
			"email":       {"email", "emailaddress"},
			"first_name":  {"givenname", "given_name"},
			"last_name":   {"familyname", "family_name"},
			"role":        {"role"},
			"external_id": {"sourcedid", "sourced_id", "external_id"},
		}
	default:
		return map[string][]string{
			"email":       {"email"},
			"first_name":  {"first_name", "firstname", "givenname"},
			"last_name":   {"last_name", "lastname", "familyname"},
			"role":        {"role"},
			"external_id": {"external_id", "sourcedid", "sourced_id"},
		}
	}
}
