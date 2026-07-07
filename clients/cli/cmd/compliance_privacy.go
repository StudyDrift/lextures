package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// --- gdpr ---

var gdprCmd = &cobra.Command{
	Use:   "gdpr",
	Short: "GDPR data-subject access and erasure workflows",
}

var gdprExportFlags struct {
	subject string
	id      string
	out     string
	yes     bool
	wait    bool
}

var gdprExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a completed DSAR data package",
	RunE:  runGDPRExport,
}

var gdprEraseFlags struct {
	subject         string
	confirmSubject  string
	yes             bool
	requestID       string
}

var gdprEraseCmd = &cobra.Command{
	Use:   "erase",
	Short: "Submit or approve a right-to-erasure DSAR (irreversible)",
	RunE:  runGDPRErase,
}

var gdprStatusFlags struct {
	subject string
	queue   bool
}

var gdprStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "List DSAR request status",
	RunE:  runGDPRStatus,
}

// --- ccpa ---

var ccpaCmd = &cobra.Command{
	Use:   "ccpa",
	Short: "CCPA consumer privacy requests",
}

var ccpaExportFlags struct {
	subject string
	typeArg string
	yes     bool
	out     string
}

var ccpaExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Submit a CCPA know/access request",
	RunE:  runCCPAExport,
}

var ccpaEraseFlags struct {
	confirmSubject string
	yes            bool
}

var ccpaEraseCmd = &cobra.Command{
	Use:   "erase",
	Short: "Submit a CCPA deletion request",
	RunE:  runCCPAErase,
}

var ccpaStatusFlags struct {
	queue bool
}

var ccpaStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "List CCPA request status",
	RunE:  runCCPAStatus,
}

// --- ferpa ---

var ferpaCmd = &cobra.Command{
	Use:   "ferpa",
	Short: "FERPA disclosures and record requests",
}

var ferpaDisclosuresListCmd = &cobra.Command{
	Use:   "disclosures list",
	Short: "List FERPA disclosure log entries",
	RunE:  runFerpaDisclosuresList,
}

var ferpaConsentListCmd = &cobra.Command{
	Use:   "consent list",
	Short: "List FERPA consent records (via record requests)",
	RunE:  runFerpaConsentList,
}

// --- coppa ---

var coppaCmd = &cobra.Command{
	Use:   "coppa",
	Short: "COPPA parental consent workflows",
}

var coppaConsentListFlags struct {
	pending bool
}

var coppaConsentListCmd = &cobra.Command{
	Use:   "consent list",
	Short: "List parental consent status from the parent dashboard",
	RunE:  runCoppaConsentList,
}

// --- compliance umbrella ---

var complianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Cross-regulation compliance exports and inventories",
}

var complianceDataInventoryExportFlags struct {
	out string
	yes bool
}

var complianceDataInventoryExportCmd = &cobra.Command{
	Use:   "data-inventory export",
	Short: "Export the platform data inventory (CSV)",
	RunE:  runComplianceDataInventoryExport,
}

var complianceAuditLogExportCmd = &cobra.Command{
	Use:   "audit-log export",
	Short: "Alias for audit-log export",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAuditLogExport(cmd, nil)
	},
}

var complianceAIInferenceExportFlags struct {
	org string
	out string
	yes bool
}

var complianceAIInferenceExportCmd = &cobra.Command{
	Use:   "ai-inference-log export",
	Short: "Export AI inference audit log",
	RunE:  runComplianceAIInferenceExport,
}

// --- soc2 ---

var soc2Cmd = &cobra.Command{
	Use:   "soc2",
	Short: "SOC 2 compliance evidence",
}

var soc2EvidenceExportFlags struct {
	out string
	yes bool
}

var soc2EvidenceExportCmd = &cobra.Command{
	Use:   "evidence export",
	Short: "Export SOC 2 evidence summary",
	RunE:  runSOC2EvidenceExport,
}

var soc2StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show SOC 2 evidence summary",
	RunE:  runSOC2Status,
}

// --- iso ---

var isoCmd = &cobra.Command{
	Use:   "iso",
	Short: "ISO 27001 ISMS controls",
}

var isoControlsListCmd = &cobra.Command{
	Use:   "controls list",
	Short: "List ISO statement-of-applicability controls",
	RunE:  runISOControlsList,
}

// --- pii ---

var piiCmd = &cobra.Command{
	Use:   "pii",
	Short: "PII redaction operations status",
}

var piiRedactStatusCmd = &cobra.Command{
	Use:   "redact status",
	Short: "Show PII redaction configuration status",
	RunE:  runPIIRedactStatus,
}

// --- ai-disclosure ---

var aiDisclosureCmd = &cobra.Command{
	Use:   "ai-disclosure",
	Short: "AI disclosure and governance configuration",
}

var aiDisclosureGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get org AI disclosure configuration",
	RunE:  runAIDisclosureGet,
}

var aiDisclosureSetFlags struct {
	file string
}

var aiDisclosureSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update org AI disclosure configuration from JSON",
	RunE:  runAIDisclosureSet,
}

// --- dpa ---

var dpaCmd = &cobra.Command{
	Use:   "dpa",
	Short: "Data processing agreement portal",
}

var dpaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List DPA acceptances",
	RunE:  runDPAList,
}

var dpaGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current DPA version",
	RunE:  runDPAGet,
}

// --- research-consent ---

var researchConsentCmd = &cobra.Command{
	Use:   "research-consent",
	Short: "Research consent study management",
}

var researchConsentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List consent studies",
	RunE:  runResearchConsentList,
}

func init() {
	gdprExportCmd.Flags().StringVar(&gdprExportFlags.subject, "subject", "", "filter/export DSAR for subject user UUID")
	gdprExportCmd.Flags().StringVar(&gdprExportFlags.id, "id", "", "completed DSAR request UUID to download")
	gdprExportCmd.Flags().StringVar(&gdprExportFlags.out, "out", ".", "output directory")
	gdprExportCmd.Flags().BoolVar(&gdprExportFlags.yes, "yes", false, "confirm exporting sensitive DSAR data")
	gdprExportCmd.Flags().BoolVar(&gdprExportFlags.wait, "wait", false, "poll until DSAR completes before download")

	gdprEraseCmd.Flags().StringVar(&gdprEraseFlags.subject, "subject", "", "subject user UUID for erasure request")
	gdprEraseCmd.Flags().StringVar(&gdprEraseFlags.confirmSubject, "confirm-subject", "", "must match --subject (double confirmation)")
	gdprEraseCmd.Flags().BoolVar(&gdprEraseFlags.yes, "yes", false, "confirm irreversible erasure")
	gdprEraseCmd.Flags().StringVar(&gdprEraseFlags.requestID, "request-id", "", "approve an existing erasure DSAR (admin)")

	gdprStatusCmd.Flags().StringVar(&gdprStatusFlags.subject, "subject", "", "filter by subject user UUID")
	gdprStatusCmd.Flags().BoolVar(&gdprStatusFlags.queue, "queue", false, "admin queue view")

	ccpaExportCmd.Flags().StringVar(&ccpaExportFlags.subject, "subject", "", "subject user UUID (informational)")
	ccpaExportCmd.Flags().StringVar(&ccpaExportFlags.typeArg, "type", "know_specific", "request type: know_categories, know_specific")
	ccpaExportCmd.Flags().BoolVar(&ccpaExportFlags.yes, "yes", false, "confirm submitting privacy request")
	ccpaExportCmd.Flags().StringVar(&ccpaExportFlags.out, "out", ".", "output directory")

	ccpaEraseCmd.Flags().StringVar(&ccpaEraseFlags.confirmSubject, "confirm-subject", "", "subject UUID confirmation")
	ccpaEraseCmd.Flags().BoolVar(&ccpaEraseFlags.yes, "yes", false, "confirm irreversible deletion request")

	ccpaStatusCmd.Flags().BoolVar(&ccpaStatusFlags.queue, "queue", false, "admin queue view")

	coppaConsentListCmd.Flags().BoolVar(&coppaConsentListFlags.pending, "pending", false, "show only pending consents")

	complianceDataInventoryExportCmd.Flags().StringVar(&complianceDataInventoryExportFlags.out, "out", ".", "output directory")
	complianceDataInventoryExportCmd.Flags().BoolVar(&complianceDataInventoryExportFlags.yes, "yes", false, "confirm export")

	complianceAIInferenceExportCmd.Flags().StringVar(&complianceAIInferenceExportFlags.org, "org", "", "filter by organization UUID")
	complianceAIInferenceExportCmd.Flags().StringVar(&complianceAIInferenceExportFlags.out, "out", ".", "output directory")
	complianceAIInferenceExportCmd.Flags().BoolVar(&complianceAIInferenceExportFlags.yes, "yes", false, "confirm export")

	soc2EvidenceExportCmd.Flags().StringVar(&soc2EvidenceExportFlags.out, "out", ".", "output directory")
	soc2EvidenceExportCmd.Flags().BoolVar(&soc2EvidenceExportFlags.yes, "yes", false, "confirm export")

	aiDisclosureSetCmd.Flags().StringVar(&aiDisclosureSetFlags.file, "file", "", "AI config JSON file")

	gdprCmd.AddCommand(gdprExportCmd, gdprEraseCmd, gdprStatusCmd)
	ccpaCmd.AddCommand(ccpaExportCmd, ccpaEraseCmd, ccpaStatusCmd)
	ferpaCmd.AddCommand(ferpaDisclosuresListCmd, ferpaConsentListCmd)
	coppaCmd.AddCommand(coppaConsentListCmd)
	complianceCmd.AddCommand(complianceDataInventoryExportCmd, complianceAuditLogExportCmd, complianceAIInferenceExportCmd)
	soc2Cmd.AddCommand(soc2EvidenceExportCmd, soc2StatusCmd)
	isoCmd.AddCommand(isoControlsListCmd)
	piiCmd.AddCommand(piiRedactStatusCmd)
	aiDisclosureCmd.AddCommand(aiDisclosureGetCmd, aiDisclosureSetCmd)
	dpaCmd.AddCommand(dpaListCmd, dpaGetCmd)
	researchConsentCmd.AddCommand(researchConsentListCmd)

	rootCmd.AddCommand(gdprCmd, ccpaCmd, ferpaCmd, coppaCmd, complianceCmd, soc2Cmd, isoCmd, piiCmd, aiDisclosureCmd, dpaCmd, researchConsentCmd)
}

func runGDPRExport(cmd *cobra.Command, _ []string) error {
	if !gdprExportFlags.yes {
		return fmt.Errorf("%s", complianceExportWarning)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	requestID := gdprExportFlags.id

	if requestID == "" {
		id, _, err := submitGDPRDSAR(c, "access")
		if err != nil {
			return err
		}
		requestID = id
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "submitted DSAR request_id=%s\n", requestID)
		if gdprExportFlags.wait {
			if err := waitForDSARComplete(c, requestID, 2*time.Minute); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("DSAR %s submitted; re-run with --id %s --wait after completion", requestID, requestID)
		}
	} else if gdprExportFlags.wait {
		if err := waitForDSARComplete(c, requestID, 2*time.Minute); err != nil {
			return err
		}
	}

	if gdprExportFlags.subject != "" {
		requests, _, err := listGDPRDSARs(c, true)
		if err != nil {
			return err
		}
		requests = filterDSARsBySubject(requests, gdprExportFlags.subject)
		for _, r := range requests {
			if r.Status == "completed" && (r.RequestType == "access" || r.RequestType == "portability") {
				requestID = r.ID
				break
			}
		}
	}

	body, err := downloadGDPRDSAR(c, requestID)
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(gdprExportFlags.out, "gdpr-dsar-"+requestID+".json", body)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "downloaded DSAR package to %s (request_id=%s)\n", path, requestID)
	return nil
}

func runGDPRErase(cmd *cobra.Command, _ []string) error {
	subject := strings.TrimSpace(gdprEraseFlags.subject)
	if gdprEraseFlags.requestID != "" {
		if !gdprEraseFlags.yes {
			return fmt.Errorf("erasure approval requires --yes")
		}
		body, err := patchGDPRDSAR(client.New(Cfg.Server, Cfg.APIKey), gdprEraseFlags.requestID, "approved", "")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "approved erasure request_id=%s\n", gdprEraseFlags.requestID)
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(body)
		}
		return err
	}
	if subject == "" {
		return fmt.Errorf("--subject is required")
	}
	if gdprEraseFlags.confirmSubject != subject {
		return fmt.Errorf("erasure refused: --confirm-subject must match --subject %q", subject)
	}
	if !gdprEraseFlags.yes {
		return fmt.Errorf("irreversible erasure requires --yes")
	}
	id, _, err := submitGDPRDSAR(client.New(Cfg.Server, Cfg.APIKey), "erasure")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "submitted erasure DSAR request_id=%s for subject=%s\n", id, subject)
	return nil
}

func runGDPRStatus(cmd *cobra.Command, _ []string) error {
	requests, raw, err := listGDPRDSARs(client.New(Cfg.Server, Cfg.APIKey), gdprStatusFlags.queue)
	if err != nil {
		return err
	}
	requests = filterDSARsBySubject(requests, gdprStatusFlags.subject)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"requests": requests})
	}
	if gdprStatusFlags.subject == "" && len(requests) == 0 && raw != nil {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tUSER\tTYPE\tSTATUS\tDUE")
	for _, r := range requests {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.UserID, r.RequestType, r.Status, r.DueAt)
	}
	return w.Flush()
}

func runCCPAExport(cmd *cobra.Command, _ []string) error {
	if !ccpaExportFlags.yes {
		return fmt.Errorf("%s", complianceExportWarning)
	}
	id, body, err := submitCCPARequest(client.New(Cfg.Server, Cfg.APIKey), ccpaExportFlags.typeArg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "submitted CCPA request_id=%s\n", id)
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
	}
	return err
}

func runCCPAErase(cmd *cobra.Command, _ []string) error {
	if ccpaEraseFlags.confirmSubject == "" {
		return fmt.Errorf("erasure refused: --confirm-subject is required")
	}
	if !ccpaEraseFlags.yes {
		return fmt.Errorf("irreversible deletion requires --yes")
	}
	id, body, err := submitCCPARequest(client.New(Cfg.Server, Cfg.APIKey), "delete")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "submitted CCPA deletion request_id=%s confirm_subject=%s\n", id, ccpaEraseFlags.confirmSubject)
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
	}
	return err
}

func runCCPAStatus(cmd *cobra.Command, _ []string) error {
	body, err := listCCPARequests(client.New(Cfg.Server, Cfg.APIKey), ccpaStatusFlags.queue)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runFerpaDisclosuresList(cmd *cobra.Command, _ []string) error {
	body, err := fetchFerpaDisclosureLog(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runFerpaConsentList(cmd *cobra.Command, _ []string) error {
	body, err := fetchComplianceGET(client.New(Cfg.Server, Cfg.APIKey), "/api/v1/compliance/ferpa/record-requests")
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCoppaConsentList(cmd *cobra.Command, _ []string) error {
	body, err := fetchCoppaParentDashboard(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if !coppaConsentListFlags.pending || globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Children []struct {
			UserID  string `json:"userId"`
			Status  string `json:"status"`
			Pending bool   `json:"pending"`
		} `json:"children"`
	}
	if json.Unmarshal(body, &out) != nil {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "USER\tSTATUS")
	for _, ch := range out.Children {
		if ch.Pending || ch.Status == "pending" {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", ch.UserID, ch.Status)
		}
	}
	return w.Flush()
}

func runComplianceDataInventoryExport(cmd *cobra.Command, _ []string) error {
	if !complianceDataInventoryExportFlags.yes {
		return fmt.Errorf("%s", complianceExportWarning)
	}
	body, _, err := fetchComplianceExport(client.New(Cfg.Server, Cfg.APIKey), "/api/v1/compliance/data-inventory/export.csv")
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(complianceDataInventoryExportFlags.out, "data-inventory.csv", body)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported data inventory to %s\n", path)
	return nil
}

func runComplianceAIInferenceExport(cmd *cobra.Command, _ []string) error {
	if !complianceAIInferenceExportFlags.yes {
		return fmt.Errorf("%s", complianceExportWarning)
	}
	body, err := fetchAIInferenceLog(client.New(Cfg.Server, Cfg.APIKey), complianceAIInferenceExportFlags.org)
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(complianceAIInferenceExportFlags.out, "ai-inference-log.json", body)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported AI inference log to %s\n", path)
	return nil
}

func runSOC2EvidenceExport(cmd *cobra.Command, _ []string) error {
	if !soc2EvidenceExportFlags.yes {
		return fmt.Errorf("%s", complianceExportWarning)
	}
	body, err := fetchSOC2Evidence(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	path, err := writeComplianceExport(soc2EvidenceExportFlags.out, "soc2-evidence.json", body)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "exported SOC 2 evidence to %s\n", path)
	return nil
}

func runSOC2Status(cmd *cobra.Command, _ []string) error {
	body, err := fetchSOC2Evidence(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runISOControlsList(cmd *cobra.Command, _ []string) error {
	body, err := fetchISOControls(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runPIIRedactStatus(cmd *cobra.Command, _ []string) error {
	body, err := fetchPIIRedactionStatus(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAIDisclosureGet(cmd *cobra.Command, _ []string) error {
	body, err := fetchAdminAIConfig(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAIDisclosureSet(cmd *cobra.Command, _ []string) error {
	payload, err := readComplianceInputFile(aiDisclosureSetFlags.file)
	if err != nil {
		return err
	}
	body, err := putComplianceJSON(client.New(Cfg.Server, Cfg.APIKey), "/api/v1/admin/ai-config", payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runDPAList(cmd *cobra.Command, _ []string) error {
	body, err := fetchDPAAcceptances(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runDPAGet(cmd *cobra.Command, _ []string) error {
	body, err := fetchDPACurrent(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runResearchConsentList(cmd *cobra.Command, _ []string) error {
	body, err := fetchResearchConsentStudies(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}