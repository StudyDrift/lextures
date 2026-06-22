package browsersaml

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beevik/etree"
	samllib "github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	xrv "github.com/mattermost/xml-roundtrip-validator"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/samlidp"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

// HandleSLO processes SAML 2.0 Single Logout (IdP-initiated LogoutRequest, IdP LogoutResponse,
// or SP-initiated logout via GET ?idpId=&nameId=).
func HandleSLO(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, publicWebOrigin string, w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return &HTTPStatusError{http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)}
	}
	_, _ = samlidp.DeleteStaleAuthnState(ctx, pool)
	_, _ = samlidp.DeleteStaleReplayGuard(ctx, pool)

	if hasSAMLResponse(r) {
		return handleIncomingLogoutResponse(ctx, pool, cfg, publicWebOrigin, w, r)
	}
	if hasSAMLRequest(r) {
		return handleIncomingLogoutRequest(ctx, pool, cfg, w, r)
	}
	if r.Method == http.MethodGet && strings.TrimSpace(r.URL.Query().Get("idpId")) != "" {
		return handleSPInitiatedLogout(ctx, pool, cfg, w, r)
	}
	return &HTTPStatusError{http.StatusBadRequest, "Missing SAMLRequest, SAMLResponse, or idpId."}
}

func hasSAMLRequest(r *http.Request) bool {
	if strings.TrimSpace(r.URL.Query().Get("SAMLRequest")) != "" {
		return true
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		return strings.TrimSpace(r.PostFormValue("SAMLRequest")) != ""
	}
	return false
}

func hasSAMLResponse(r *http.Request) bool {
	if strings.TrimSpace(r.URL.Query().Get("SAMLResponse")) != "" {
		return true
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		return strings.TrimSpace(r.PostFormValue("SAMLResponse")) != ""
	}
	return false
}

func decodeSAMLRedirectParam(b64 string) ([]byte, error) {
	compressed, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("cannot decode SAML redirect parameter: %w", err)
	}
	return io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
}

func decodeSAMLPostParam(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}

func extractSAMLRequestXML(r *http.Request) ([]byte, error) {
	if q := strings.TrimSpace(r.URL.Query().Get("SAMLRequest")); q != "" {
		return decodeSAMLRedirectParam(q)
	}
	_ = r.ParseForm()
	if p := strings.TrimSpace(r.PostFormValue("SAMLRequest")); p != "" {
		return decodeSAMLPostParam(p)
	}
	return nil, fmt.Errorf("missing SAMLRequest")
}

func extractSAMLResponseParam(r *http.Request) (string, bool) {
	if q := strings.TrimSpace(r.URL.Query().Get("SAMLResponse")); q != "" {
		return q, true
	}
	_ = r.ParseForm()
	if p := strings.TrimSpace(r.PostFormValue("SAMLResponse")); p != "" {
		return p, true
	}
	return "", false
}

func validateIDPSignature(certPEM string, root *etree.Element) error {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return fmt.Errorf("invalid IdP certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	store := dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}}
	ctx := dsig.NewDefaultValidationContext(&store)
	ctx.IdAttribute = "ID"
	if _, err := ctx.Validate(root); err != nil {
		return fmt.Errorf("invalid SAML signature: %w", err)
	}
	return nil
}

type parsedLogoutRequest struct {
	ID      string
	Issuer  string
	NameID  string
	Instant time.Time
}

func readLogoutRequestRoot(xmlBytes []byte) (*etree.Element, error) {
	if err := xrv.Validate(bytes.NewReader(xmlBytes)); err != nil {
		return nil, fmt.Errorf("invalid logout request XML: %w", err)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlBytes); err != nil {
		return nil, err
	}
	root := doc.Root()
	if root == nil || !strings.HasSuffix(root.Tag, "LogoutRequest") {
		return nil, fmt.Errorf("expected LogoutRequest root element")
	}
	return root, nil
}

func parseLogoutRequestRoot(root *etree.Element, idpCertPEM, expectedDestination string) (*parsedLogoutRequest, error) {
	if strings.TrimSpace(idpCertPEM) != "" {
		if err := validateIDPSignature(idpCertPEM, root); err != nil {
			return nil, err
		}
	}
	dest := root.SelectAttrValue("Destination", "")
	if expectedDestination != "" && dest != "" && dest != expectedDestination {
		return nil, fmt.Errorf("destination mismatch")
	}
	inst := root.SelectAttrValue("IssueInstant", "")
	if inst == "" {
		return nil, fmt.Errorf("missing IssueInstant")
	}
	issued, err := time.Parse(time.RFC3339, inst)
	if err != nil {
		issued, err = time.Parse("2006-01-02T15:04:05Z", inst)
		if err != nil {
			return nil, fmt.Errorf("invalid IssueInstant")
		}
	}
	if issued.Add(samllib.MaxIssueDelay).Before(time.Now().UTC()) {
		return nil, fmt.Errorf("logout request expired")
	}
	id := root.SelectAttrValue("ID", "")
	if id == "" {
		return nil, fmt.Errorf("missing request ID")
	}
	issuer := strings.TrimSpace(elementText(root, "Issuer"))
	if issuer == "" {
		return nil, fmt.Errorf("missing Issuer")
	}
	nameID := strings.TrimSpace(elementText(root, "NameID"))
	if nameID == "" {
		return nil, fmt.Errorf("missing NameID")
	}
	return &parsedLogoutRequest{ID: id, Issuer: issuer, NameID: nameID, Instant: issued}, nil
}

func elementText(root *etree.Element, local string) string {
	for _, el := range root.ChildElements() {
		if el.Tag == local {
			return el.Text()
		}
	}
	if el := root.FindElement(".//" + local); el != nil {
		return el.Text()
	}
	return ""
}

func serviceProviderForIDP(cfg config.Config, row *samlidp.IDPRow) (*samllib.ServiceProvider, error) {
	xmlStr, err := IDPMetadataXMLFromRow(row.EntityID, row.SSOURL, row.IDPCertPem, row.SLOURL)
	if err != nil {
		return nil, err
	}
	meta, err := ParseIDPMetadata(xmlStr)
	if err != nil {
		return nil, err
	}
	return ServiceProvider(cfg, meta)
}

func revokeUserByNameID(ctx context.Context, pool *pgxpool.Pool, nameID string) error {
	email := user.NormalizeEmail(strings.TrimSpace(nameID))
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("NameID is not an email address")
	}
	urow, err := user.FindByEmailCI(ctx, pool, email)
	if err != nil {
		return err
	}
	if urow == nil {
		return nil
	}
	uid, err := uuid.Parse(urow.ID)
	if err != nil {
		return err
	}
	return authservice.RevokeAllSessionsForUser(ctx, pool, uid)
}

func handleIncomingLogoutRequest(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, w http.ResponseWriter, r *http.Request) error {
	xmlBytes, err := extractSAMLRequestXML(r)
	if err != nil {
		return &HTTPStatusError{http.StatusBadRequest, err.Error()}
	}
	root, err := readLogoutRequestRoot(xmlBytes)
	if err != nil {
		return &HTTPStatusError{http.StatusBadRequest, "Invalid LogoutRequest: " + err.Error()}
	}
	issuer := strings.TrimSpace(elementText(root, "Issuer"))
	idpRow, err := samlidp.GetIDPByEntityID(ctx, pool, issuer)
	if err != nil {
		return err
	}
	if idpRow == nil {
		return &HTTPStatusError{http.StatusBadRequest, "Unknown SAML IdP issuer."}
	}
	sloDest := strings.TrimRight(cfg.SAMLPublicBaseURL, "/") + "/auth/saml/slo"
	parsed, err := parseLogoutRequestRoot(root, idpRow.IDPCertPem, sloDest)
	if err != nil {
		return &HTTPStatusError{http.StatusBadRequest, "Invalid LogoutRequest: " + err.Error()}
	}
	okReplay, err := samlidp.RecordReplay(ctx, pool, "slo:"+parsed.ID)
	if err != nil {
		return err
	}
	if !okReplay {
		return &HTTPStatusError{http.StatusConflict, "Logout request replay detected."}
	}
	if err := revokeUserByNameID(ctx, pool, parsed.NameID); err != nil {
		return &HTTPStatusError{http.StatusBadRequest, err.Error()}
	}

	sp, err := serviceProviderForIDP(cfg, idpRow)
	if err != nil {
		return err
	}
	relay := strings.TrimSpace(r.URL.Query().Get("RelayState"))
	if relay == "" {
		_ = r.ParseForm()
		relay = strings.TrimSpace(r.PostFormValue("RelayState"))
	}
	formHTML, err := sp.MakePostLogoutResponse(parsed.ID, relay)
	if err != nil {
		return fmt.Errorf("could not build LogoutResponse: %w", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(formHTML)
	return nil
}

func handleIncomingLogoutResponse(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, publicWebOrigin string, w http.ResponseWriter, r *http.Request) error {
	def, err := samlidp.GetDefaultIdP(ctx, pool)
	if err != nil {
		return err
	}
	if def == nil {
		return &HTTPStatusError{http.StatusBadRequest, "No SAML IdP configured."}
	}
	sp, err := serviceProviderForIDP(cfg, def)
	if err != nil {
		return err
	}
	param, ok := extractSAMLResponseParam(r)
	if !ok {
		return &HTTPStatusError{http.StatusBadRequest, "Missing SAMLResponse."}
	}
	var validateErr error
	if strings.TrimSpace(r.URL.Query().Get("SAMLResponse")) != "" {
		validateErr = sp.ValidateLogoutResponseRedirect(param)
	} else {
		validateErr = sp.ValidateLogoutResponseForm(param)
	}
	if validateErr != nil {
		return &HTTPStatusError{http.StatusBadRequest, "Invalid LogoutResponse: " + validateErr.Error()}
	}
	pub := strings.TrimRight(strings.TrimSpace(publicWebOrigin), "/")
	if pub == "" {
		pub = "/"
	}
	body := fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"/><title>Signed out</title>
<meta http-equiv="refresh" content="0;url=%s"/></head>
<body><p>You have been signed out. <a href="%s">Continue</a></p></body></html>`,
		html.EscapeString(pub+"/"), html.EscapeString(pub+"/"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
	return nil
}

func handleSPInitiatedLogout(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, w http.ResponseWriter, r *http.Request) error {
	idpStr := strings.TrimSpace(r.URL.Query().Get("idpId"))
	idpUUID, err := uuid.Parse(idpStr)
	if err != nil {
		return &HTTPStatusError{http.StatusBadRequest, "Invalid idpId."}
	}
	nameID := strings.TrimSpace(r.URL.Query().Get("nameId"))
	if nameID == "" {
		return &HTTPStatusError{http.StatusBadRequest, "Missing nameId query parameter."}
	}
	row, err := samlidp.GetIDPByID(ctx, pool, idpUUID)
	if err != nil {
		return err
	}
	if row == nil {
		return &HTTPStatusError{http.StatusNotFound, "IdP not found."}
	}
	if row.SLOURL == nil || strings.TrimSpace(*row.SLOURL) == "" {
		return &HTTPStatusError{http.StatusBadRequest, "IdP has no Single Logout URL configured."}
	}
	if err := revokeUserByNameID(ctx, pool, nameID); err != nil {
		return &HTTPStatusError{http.StatusBadRequest, err.Error()}
	}
	sp, err := serviceProviderForIDP(cfg, row)
	if err != nil {
		return err
	}
	relay := strings.TrimSpace(r.URL.Query().Get("RelayState"))
	redirectURL, err := sp.MakeRedirectLogoutRequest(nameID, relay)
	if err != nil {
		return fmt.Errorf("could not start IdP logout: %w", err)
	}
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	return nil
}