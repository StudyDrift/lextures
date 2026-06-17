package credentials

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// LinkedInParams are pre-filled certification fields for LinkedIn's add-to-profile flow.
type LinkedInParams struct {
	Name             string `json:"name"`
	OrganizationName string `json:"organizationName"`
	IssueYear        int    `json:"issueYear"`
	IssueMonth       int    `json:"issueMonth"`
	CertURL          string `json:"certUrl"`
	CertID           string `json:"certId"`
	URL              string `json:"url"`
}

// BuildLinkedInParams constructs LinkedIn certification deep-link parameters (plan 15.6).
func BuildLinkedInParams(credentialName, organizationName, verificationURL, certID string, issuedAt time.Time) LinkedInParams {
	org := strings.TrimSpace(organizationName)
	if org == "" {
		org = "Lextures"
	}
	year := issuedAt.UTC().Year()
	month := int(issuedAt.UTC().Month())
	params := LinkedInParams{
		Name:             credentialName,
		OrganizationName: org,
		IssueYear:        year,
		IssueMonth:       month,
		CertURL:          verificationURL,
		CertID:           certID,
	}
	params.URL = BuildLinkedInCertificationURL(params)
	return params
}

// BuildLinkedInCertificationURL returns the LinkedIn add-certification URL.
func BuildLinkedInCertificationURL(p LinkedInParams) string {
	u, _ := url.Parse("https://www.linkedin.com/profile/add")
	q := u.Query()
	q.Set("startTask", "CERTIFICATION_NAME")
	q.Set("name", p.Name)
	q.Set("organizationName", p.OrganizationName)
	q.Set("issueYear", fmt.Sprintf("%d", p.IssueYear))
	q.Set("issueMonth", fmt.Sprintf("%d", p.IssueMonth))
	q.Set("certUrl", p.CertURL)
	q.Set("certId", p.CertID)
	u.RawQuery = q.Encode()
	return u.String()
}