// Package transcriptconsent provides versioned FERPA release-authorization text
// and payload hashing for transcript order e-signatures (T04).
package transcriptconsent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// CurrentTextVersion is the authorization text version pinned into new signatures.
const CurrentTextVersion = "ferpa-release-v1"

// ScopeFullAcademicRecord is the default records scope for a transcript release.
const ScopeFullAcademicRecord = "full_academic_record"

// PurposeTranscriptRelease is the default purpose string.
const PurposeTranscriptRelease = "Official transcript release to the named recipients"

// AuthorizationText returns the localized FERPA release text for a version + locale.
// Unknown versions return an error so signed exports stay pinned to known copy.
func AuthorizationText(version, locale string) (string, error) {
	loc := normalizeLocale(locale)
	switch version {
	case CurrentTextVersion:
		switch loc {
		case "es":
			return authorizationTextV1ES, nil
		case "fr":
			return authorizationTextV1FR, nil
		default:
			return authorizationTextV1EN, nil
		}
	default:
		return "", fmt.Errorf("unknown consent text version %q", version)
	}
}

func normalizeLocale(locale string) string {
	loc := strings.ToLower(strings.TrimSpace(locale))
	if i := strings.IndexByte(loc, '-'); i > 0 {
		loc = loc[:i]
	}
	if i := strings.IndexByte(loc, '_'); i > 0 {
		loc = loc[:i]
	}
	switch loc {
	case "es", "fr", "en":
		return loc
	default:
		return "en"
	}
}

// RecipientSnapshot is the recipient list stored on a consent row and hashed.
type RecipientSnapshot struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// Payload is the canonical authorization content hashed for tamper evidence.
type Payload struct {
	OrderID      string              `json:"orderId"`
	UserID       string              `json:"userId"`
	SignerID     string              `json:"signerId"`
	SignerRole   string              `json:"signerRole"`
	Recipients   []RecipientSnapshot `json:"recipients"`
	Scope        string              `json:"scope"`
	Purpose      string              `json:"purpose"`
	TextVersion  string              `json:"textVersion"`
	Locale       string              `json:"locale"`
	Agree        bool                `json:"agree"`
}

// HashPayload returns the SHA-256 hex digest of the canonical JSON payload.
func HashPayload(p Payload) (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

const authorizationTextV1EN = `FERPA RELEASE AUTHORIZATION

I authorize the disclosure of my education records as described below, pursuant to the Family Educational Rights and Privacy Act (FERPA), 34 CFR §99.30.

I understand that:
1. The records to be disclosed are my official academic transcript (or the variant specified in this order).
2. The purpose of the disclosure is to fulfill this transcript order to the recipients listed.
3. Only the recipients named in this authorization may receive the records under this release.
4. I have the right to revoke this authorization in writing before the records are delivered.
5. This authorization is voluntary; declining to sign will prevent release to third-party recipients.

By signing, I confirm that I am the eligible student (or an authorized parent/guardian of a minor student) and that I intend this electronic signature to be my legally binding authorization.`

const authorizationTextV1ES = `AUTORIZACIÓN DE DIVULGACIÓN FERPA

Autorizo la divulgación de mis registros educativos según se describe a continuación, de conformidad con la Ley de Derechos Educativos y Privacidad Familiar (FERPA), 34 CFR §99.30.

Entiendo que:
1. Los registros a divulgar son mi expediente académico oficial (o la variante indicada en este pedido).
2. El propósito de la divulgación es cumplir este pedido de expediente a los destinatarios listados.
3. Solo los destinatarios nombrados en esta autorización pueden recibir los registros bajo esta liberación.
4. Tengo derecho a revocar esta autorización por escrito antes de que se entreguen los registros.
5. Esta autorización es voluntaria; negarme a firmar impedirá la liberación a terceros.

Al firmar, confirmo que soy el estudiante elegible (o un padre/tutor autorizado de un estudiante menor) y que esta firma electrónica es mi autorización legalmente vinculante.`

const authorizationTextV1FR = `AUTORISATION DE DIVULGATION FERPA

J'autorise la divulgation de mes dossiers scolaires telle que décrite ci-dessous, conformément à la Family Educational Rights and Privacy Act (FERPA), 34 CFR §99.30.

Je comprends que :
1. Les dossiers à divulguer sont mon relevé de notes officiel (ou la variante indiquée dans cette commande).
2. Le but de la divulgation est d'exécuter cette commande de relevé aux destinataires listés.
3. Seuls les destinataires nommés dans cette autorisation peuvent recevoir les dossiers.
4. J'ai le droit de révoquer cette autorisation par écrit avant la livraison des dossiers.
5. Cette autorisation est volontaire ; refuser de signer empêchera la divulgation à des tiers.

En signant, je confirme que je suis l'étudiant admissible (ou un parent/tuteur autorisé d'un mineur) et que cette signature électronique constitue mon autorisation juridiquement contraignante.`
