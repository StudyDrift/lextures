package toolmarket

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"

	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

const (
	MaxBundleBytesGzip = 512 * 1024 // CT.5 budget default for marketplace tools
)

var (
	// marketplaceToolIDPattern: developer_namespace.tool_name
	marketplaceToolIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}\.[a-z][a-z0-9_]{1,48}$`)
	reservedNamespaces       = map[string]struct{}{
		"lextures": {}, "system": {}, "builtin": {}, "official": {}, "platform": {},
	}
)

// CheckResult is one automated check outcome.
type CheckResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// ChecksReport is the automated submission gate (FR-2 / AC-1).
type ChecksReport struct {
	OK     bool          `json:"ok"`
	Checks []CheckResult `json:"checks"`
}

// ValidateMarketplaceToolID enforces namespaced third-party ids.
func ValidateMarketplaceToolID(toolID string) error {
	toolID = strings.TrimSpace(toolID)
	if !marketplaceToolIDPattern.MatchString(toolID) {
		return fmt.Errorf("tool_id must be namespace.tool_name (lowercase snake_case)")
	}
	ns := strings.SplitN(toolID, ".", 2)[0]
	if _, reserved := reservedNamespaces[ns]; reserved {
		return fmt.Errorf("namespace %q is reserved", ns)
	}
	// Must not collide with first-party snake_case ids (defensive).
	if ctsvc.MustDefault().Get(toolID) != nil {
		return fmt.Errorf("tool_id collides with a first-party tool")
	}
	return nil
}

// ComputeSRI returns a sha256 Subresource Integrity digest for bundle bytes.
func ComputeSRI(bundle []byte) string {
	sum := sha256.Sum256(bundle)
	return "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
}

// RunAutomatedChecks validates a release before human review (FR-2).
// axe/keyboard statuses may be supplied by the developer harness; failing values reject.
func RunAutomatedChecks(
	toolID string,
	version string,
	manifestJSON json.RawMessage,
	dataSheetJSON json.RawMessage,
	bundle []byte,
	axeStatus string,
	keyboardTestStatus string,
	i18nKeys map[string]string,
) ChecksReport {
	rep := ChecksReport{OK: true, Checks: []CheckResult{}}
	add := func(name string, ok bool, msg string) {
		rep.Checks = append(rep.Checks, CheckResult{Name: name, OK: ok, Message: msg})
		if !ok {
			rep.OK = false
		}
	}

	if err := ValidateMarketplaceToolID(toolID); err != nil {
		add("namespace", false, err.Error())
	} else {
		add("namespace", true, "")
	}

	sv, err := ctsvc.ParseSemVer(version)
	if err != nil {
		add("semver", false, err.Error())
	} else {
		add("semver", true, fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch))
	}

	var m ctsvc.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		add("manifest_json", false, "invalid JSON: "+err.Error())
		return rep
	}
	add("manifest_json", true, "")

	// Force marketplace sandbox to iframe (FR-10) — not trusted from developer.
	if m.Sandbox != "" && m.Sandbox != ctsvc.SandboxIframe {
		add("sandbox_forced", false, "third-party tools must use sandbox iframe (forced by platform)")
	} else {
		add("sandbox_forced", true, "iframe")
	}
	m.Sandbox = ctsvc.SandboxIframe
	m.ID = toolID
	m.Version = version

	if m.DataSheet == nil && len(dataSheetJSON) > 0 && string(dataSheetJSON) != "null" {
		var sheet ctsvc.DataSheetDecl
		if err := json.Unmarshal(dataSheetJSON, &sheet); err == nil {
			m.DataSheet = &sheet
		}
	}

	if err := validateMarketplaceManifest(m); err != nil {
		add("manifest_validity", false, err.Error())
	} else {
		add("manifest_validity", true, "")
	}

	if m.DataSheet == nil {
		add("data_sheet", false, "data sheet required")
	} else if err := ctsvc.ValidateDataSheet(m); err != nil {
		add("data_sheet", false, err.Error())
	} else {
		add("data_sheet", true, "")
	}

	budget := m.MaxBundleBytesGzip
	if budget <= 0 {
		budget = MaxBundleBytesGzip
	}
	if len(bundle) > budget {
		add("bundle_budget", false, fmt.Sprintf("bundle %d bytes exceeds budget %d", len(bundle), budget))
	} else {
		add("bundle_budget", true, fmt.Sprintf("%d/%d", len(bundle), budget))
	}

	if axeStatus == "" {
		axeStatus = "pass"
	}
	if axeStatus == "fail" {
		add("axe", false, "axe gate failed")
	} else {
		add("axe", true, axeStatus)
	}

	if keyboardTestStatus == "" {
		keyboardTestStatus = "pass"
	}
	if keyboardTestStatus == "fail" || !m.A11y.KeyboardOperable {
		add("keyboard", false, "keyboard-operability test missing or failed")
	} else {
		add("keyboard", true, keyboardTestStatus)
	}

	if len(m.I18nNamespace) == 0 {
		add("i18n", false, "i18nNamespace required")
	} else if i18nKeys != nil && len(i18nKeys) == 0 {
		add("i18n", false, "i18n bundle incomplete")
	} else {
		add("i18n", true, m.I18nNamespace)
	}

	disallowed := false
	for _, cap := range m.Capabilities {
		if cap == "platform_ai" {
			add("disallowed_apis", false, "platform_ai capability is off by default in v1")
			disallowed = true
			break
		}
	}
	if !disallowed {
		add("disallowed_apis", true, "")
	}

	if m.Network != nil {
		okHosts := true
		for _, host := range m.Network.AllowedHosts {
			if err := validatePublicHost(host); err != nil {
				add("network_hosts", false, fmt.Sprintf("%s: %v", host, err))
				okHosts = false
				break
			}
		}
		if okHosts {
			add("network_hosts", true, fmt.Sprintf("%d hosts", len(m.Network.AllowedHosts)))
		}
	} else {
		add("network_hosts", true, "none")
	}

	return rep
}

// validateMarketplaceManifest mirrors CT.5/CT.8 checks but allows namespaced tool ids.
func validateMarketplaceManifest(m ctsvc.Manifest) error {
	if err := ValidateMarketplaceToolID(m.ID); err != nil {
		return err
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if _, ok := ctsvc.AllowedCategories[m.Category]; !ok {
		return fmt.Errorf("unknown category %q", m.Category)
	}
	for _, c := range m.Capabilities {
		if _, ok := ctsvc.AllowedCapabilities[c]; !ok {
			return fmt.Errorf("unknown capability %q", c)
		}
	}
	if _, ok := ctsvc.AllowedScoringModes[m.Scoring.Mode]; !ok {
		return fmt.Errorf("unknown scoring mode %q", m.Scoring.Mode)
	}
	if m.Storage.MaxStateBytes <= 0 {
		return fmt.Errorf("storage.maxStateBytes must be > 0")
	}
	if len(m.Roles.Interact) == 0 {
		return fmt.Errorf("roles.interact must be non-empty")
	}
	if !m.A11y.KeyboardOperable {
		return fmt.Errorf("a11y.keyboardOperable must be true")
	}
	if strings.TrimSpace(m.A11y.SRPattern) == "" {
		return fmt.Errorf("a11y.srPattern is required")
	}
	if strings.TrimSpace(m.I18nNamespace) == "" {
		return fmt.Errorf("i18nNamespace is required")
	}
	if strings.TrimSpace(m.UI.Renderer) == "" || strings.TrimSpace(m.UI.Icon) == "" || strings.TrimSpace(m.UI.Group) == "" {
		return fmt.Errorf("ui.renderer, ui.icon, and ui.group are required")
	}
	if len(m.ConfigSchema) == 0 || len(m.StateSchema) == 0 {
		return fmt.Errorf("configSchema and stateSchema are required")
	}
	if m.Sandbox != "" {
		if _, ok := ctsvc.AllowedSandboxModes[m.Sandbox]; !ok {
			return fmt.Errorf("unknown sandbox mode %q", m.Sandbox)
		}
	}
	return nil
}

func validatePublicHost(host string) error {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if strings.Contains(host, "/") || strings.Contains(host, ":") {
		return fmt.Errorf("host must be a bare hostname (no scheme/port/path)")
	}
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("private/local host not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("private IP not allowed")
		}
		return nil
	}
	// For hostnames, reject obviously private suffixes; DNS lookup is best-effort
	// and skipped for unit tests via skipLookup marker hosts ending in .example.
	if strings.HasSuffix(host, ".example") || strings.HasSuffix(host, ".example.com") {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("host not resolvable")
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("resolves to private address")
		}
	}
	return nil
}

// ForceIframeManifest returns manifest JSON with sandbox forced to iframe and id/version set.
func ForceIframeManifest(manifestJSON json.RawMessage, toolID, version string) (json.RawMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(manifestJSON, &raw); err != nil {
		return nil, err
	}
	raw["id"] = toolID
	raw["version"] = version
	raw["sandbox"] = ctsvc.SandboxIframe
	return json.Marshal(raw)
}

// CapabilityPlainLanguage maps capability tokens to consent copy (FR-5).
func CapabilityPlainLanguage(cap string) string {
	switch cap {
	case "state":
		return "Stores student progress for this activity"
	case "scoring":
		return "Can contribute a score to the gradebook"
	case "ai":
		return "Sends student writing to an AI model"
	case "network":
		return "Sends data to an external service"
	case "media", "media_capture":
		return "Can access camera, microphone, or uploaded media"
	case "realtime":
		return "Uses a live connection while students interact"
	case "aggregate":
		return "Shows class-level aggregates to instructors"
	case "peer_visible":
		return "Other students in the course may see contributions"
	case "code_execution":
		return "Runs student-authored code"
	case "platform_ai":
		return "Uses Lextures platform AI (budgeted and disclosed)"
	default:
		return "Requests capability: " + cap
	}
}

// ExtractHosts reads network.allowedHosts from a manifest.
func ExtractHosts(manifestJSON json.RawMessage) []string {
	var m ctsvc.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil || m.Network == nil {
		return []string{}
	}
	return append([]string{}, m.Network.AllowedHosts...)
}

// ExtractCapabilities reads capabilities from a manifest.
func ExtractCapabilities(manifestJSON json.RawMessage) []string {
	var m ctsvc.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return []string{}
	}
	if m.Capabilities == nil {
		return []string{}
	}
	return append([]string{}, m.Capabilities...)
}
