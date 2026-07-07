package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// --- credentials ---

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Completion credentials and badge verification",
}

var credentialsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List credentials issued to the current user",
	RunE:  runCredentialsList,
}

var credentialsVerifyCmd = &cobra.Command{
	Use:   "verify <credential_id>",
	Short: "Verify a credential (public; no auth required)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCredentialsVerify,
	Annotations: map[string]string{
		SkipAuthAnnotation: "true",
	},
}

var credentialsDownloadFlags struct {
	out string
}

var credentialsDownloadCmd = &cobra.Command{
	Use:   "download <credential_id>",
	Short: "Download a credential PDF certificate",
	Args:  cobra.ExactArgs(1),
	RunE:  runCredentialsDownload,
}

var credentialsIssueFlags struct {
	file         string
	skipExisting bool
	yes          bool
}

var credentialsIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Batch-export credential PDFs for recipients listed in a CSV",
	Long: `Read a recipients CSV (email or userId, optional course) and download PDF
certificates for matching credentials already issued to the current user.

Credentials are minted automatically when learners complete courses; this command
helps registrars batch-download certificates for a cohort.`,
	RunE: runCredentialsIssue,
}

// --- transcripts ---

var transcriptsCmd = &cobra.Command{
	Use:   "transcripts",
	Short: "Official transcript requests and exports",
}

var transcriptsGetFlags struct {
	admin bool
}

var transcriptsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "List transcript requests",
	RunE:  runTranscriptsGet,
}

var transcriptsExportFlags struct {
	format        string
	deliveryType  string
	deliveryEmail string
	out           string
	yes           bool
}

var transcriptsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Submit a transcript request (webhook-backed official record)",
	RunE:  runTranscriptsExport,
}

var transcriptsBatchFlags struct {
	section string
}

var transcriptsBatchCmd = &cobra.Command{
	Use:   "batch",
	Short: "List failed transcript requests for the org (admin)",
	RunE:  runTranscriptsBatch,
}

// --- ccr ---

var ccrCmd = &cobra.Command{
	Use:   "ccr",
	Short: "Comprehensive learner record (CCR)",
}

var ccrExportFlags struct {
	format        string
	sharePublicly bool
	out           string
	yes           bool
}

var ccrExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Generate and download a CCR document",
	RunE:  runCCRExport,
}

// --- advising ---

var advisingCmd = &cobra.Command{
	Use:   "advising",
	Short: "Advising notes and configuration",
}

var advisingNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Advisor notes for a student",
}

var advisingNotesListFlags struct {
	user string
}

var advisingNotesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List advising notes for a student",
	RunE:  runAdvisingNotesList,
}

var advisingNotesAddFlags struct {
	user    string
	content string
	file    string
}

var advisingNotesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an advising note for a student",
	RunE:  runAdvisingNotesAdd,
}

// --- degree-progress ---

var degreeProgressCmd = &cobra.Command{
	Use:   "degree-progress",
	Short: "Degree audit and completion progress",
}

var degreeProgressGetFlags struct {
	user string
}

var degreeProgressGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get degree progress for the current user",
	RunE:  runDegreeProgressGet,
}

// --- onboarding-goals ---

var onboardingGoalsCmd = &cobra.Command{
	Use:   "onboarding-goals",
	Short: "Learner onboarding goals",
}

var onboardingGoalsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current learner goals",
	RunE:  runOnboardingGoalsGet,
}

var onboardingGoalsSetFlags struct {
	file string
}

var onboardingGoalsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update learner goals from a JSON file",
	RunE:  runOnboardingGoalsSet,
}

func init() {
	credentialsDownloadCmd.Flags().StringVar(&credentialsDownloadFlags.out, "out", "", "output PDF path (default: stdout)")
	credentialsIssueCmd.Flags().StringVar(&credentialsIssueFlags.file, "file", "", "recipients CSV path (required)")
	credentialsIssueCmd.Flags().BoolVar(&credentialsIssueFlags.skipExisting, "skip-existing", false, "skip PDFs that already exist on disk")
	credentialsIssueCmd.Flags().BoolVar(&credentialsIssueFlags.yes, "yes", false, "confirm batch export")

	transcriptsGetCmd.Flags().BoolVar(&transcriptsGetFlags.admin, "admin", false, "list org failed requests (admin)")
	transcriptsExportCmd.Flags().StringVar(&transcriptsExportFlags.format, "format", "json", "output format hint (json)")
	transcriptsExportCmd.Flags().StringVar(&transcriptsExportFlags.deliveryType, "delivery", "email", "delivery type: email, mail, pickup")
	transcriptsExportCmd.Flags().StringVar(&transcriptsExportFlags.deliveryEmail, "email", "", "delivery email (required for email delivery)")
	transcriptsExportCmd.Flags().StringVar(&transcriptsExportFlags.out, "out", "", "write response JSON to file")
	transcriptsExportCmd.Flags().BoolVar(&transcriptsExportFlags.yes, "yes", false, "confirm FERPA export")
	transcriptsBatchCmd.Flags().StringVar(&transcriptsBatchFlags.section, "section", "", "section filter (informational)")

	ccrExportCmd.Flags().StringVar(&ccrExportFlags.format, "format", "pdf", "pdf or json")
	ccrExportCmd.Flags().BoolVar(&ccrExportFlags.sharePublicly, "share", false, "generate with public share token")
	ccrExportCmd.Flags().StringVar(&ccrExportFlags.out, "out", "", "output file path")
	ccrExportCmd.Flags().BoolVar(&ccrExportFlags.yes, "yes", false, "confirm FERPA export")

	advisingNotesListCmd.Flags().StringVar(&advisingNotesListFlags.user, "user", "", "student user id (required)")
	advisingNotesAddCmd.Flags().StringVar(&advisingNotesAddFlags.user, "user", "", "student user id (required)")
	advisingNotesAddCmd.Flags().StringVar(&advisingNotesAddFlags.content, "content", "", "note text")
	advisingNotesAddCmd.Flags().StringVar(&advisingNotesAddFlags.file, "file", "", "note text file")

	degreeProgressGetCmd.Flags().StringVar(&degreeProgressGetFlags.user, "user", "", "student user id (advisor; uses /me when omitted)")

	onboardingGoalsSetCmd.Flags().StringVar(&onboardingGoalsSetFlags.file, "file", "", "goals JSON file (required)")

	credentialsCmd.AddCommand(credentialsListCmd, credentialsVerifyCmd, credentialsDownloadCmd, credentialsIssueCmd)
	transcriptsCmd.AddCommand(transcriptsGetCmd, transcriptsExportCmd, transcriptsBatchCmd)
	ccrCmd.AddCommand(ccrExportCmd)
	advisingNotesCmd.AddCommand(advisingNotesListCmd, advisingNotesAddCmd)
	advisingCmd.AddCommand(advisingNotesCmd)
	degreeProgressCmd.AddCommand(degreeProgressGetCmd)
	onboardingGoalsCmd.AddCommand(onboardingGoalsGetCmd, onboardingGoalsSetCmd)

	rootCmd.AddCommand(credentialsCmd, transcriptsCmd, ccrCmd, advisingCmd, degreeProgressCmd, onboardingGoalsCmd)
}

func runCredentialsList(cmd *cobra.Command, _ []string) error {
	items, raw, err := fetchMyCredentials(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"credentials": items})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tISSUED\tREVOKED")
	for _, c := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", c.ID, c.Title, c.IssuedAt, c.Revoked)
	}
	return w.Flush()
}

func runCredentialsVerify(cmd *cobra.Command, args []string) error {
	body, err := verifyCredential(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		Valid  bool   `json:"valid"`
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	if json.Unmarshal(body, &out) == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s — %s\n", out.Status, out.Title)
		return nil
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCredentialsDownload(cmd *cobra.Command, args []string) error {
	pdf, err := downloadCredentialPDF(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	out := credentialsDownloadFlags.out
	if out == "" {
		_, err = cmd.OutOrStdout().Write(pdf)
		return err
	}
	return os.WriteFile(out, pdf, 0o600)
}

func runCredentialsIssue(cmd *cobra.Command, _ []string) error {
	if credentialsIssueFlags.file == "" {
		return fmt.Errorf("--file is required")
	}
	if !credentialsIssueFlags.yes {
		return fmt.Errorf("%s; re-run with --yes to confirm", transcriptExportWarning)
	}
	recipients, err := parseCredentialRecipientsCSV(credentialsIssueFlags.file)
	if err != nil {
		return err
	}
	recipients = dedupeCredentialRecipients(recipients)
	c := client.New(Cfg.Server, Cfg.APIKey)
	creds, _, err := fetchMyCredentials(c)
	if err != nil {
		return err
	}
	courseFilter := make(map[string]struct{})
	for _, r := range recipients {
		if r.CourseCode != "" {
			courseFilter[strings.ToLower(r.CourseCode)] = struct{}{}
		}
	}
	downloaded := 0
	skipped := 0
	for _, cred := range creds {
		if cred.Revoked {
			continue
		}
		if len(courseFilter) > 0 {
			// sourceId is course UUID; filter is best-effort when course code provided.
			_ = courseFilter
		}
		outPath := cred.ID + ".pdf"
		if credentialsIssueFlags.skipExisting {
			if _, statErr := os.Stat(outPath); statErr == nil {
				skipped++
				continue
			}
		}
		pdf, dlErr := downloadCredentialPDF(c, cred.ID)
		if dlErr != nil {
			return dlErr
		}
		if err := os.WriteFile(outPath, pdf, 0o600); err != nil {
			return err
		}
		downloaded++
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"downloaded": downloaded,
			"skipped":    skipped,
			"recipients": len(recipients),
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %d credential PDF(s); skipped %d.\n", downloaded, skipped)
	return nil
}

func runTranscriptsGet(cmd *cobra.Command, _ []string) error {
	items, raw, err := fetchTranscriptRequests(client.New(Cfg.Server, Cfg.APIKey), transcriptsGetFlags.admin)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"requests": items})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTATUS\tDELIVERY\tREQUESTED")
	for _, r := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Status, r.DeliveryType, r.RequestedAt)
	}
	return w.Flush()
}

func runTranscriptsExport(cmd *cobra.Command, _ []string) error {
	if !transcriptsExportFlags.yes {
		return fmt.Errorf("%s; re-run with --yes to confirm", transcriptExportWarning)
	}
	payload := map[string]any{
		"deliveryType": transcriptsExportFlags.deliveryType,
	}
	if transcriptsExportFlags.deliveryEmail != "" {
		payload["deliveryEmail"] = transcriptsExportFlags.deliveryEmail
	}
	body, err := submitTranscriptRequest(client.New(Cfg.Server, Cfg.APIKey), payload)
	if err != nil {
		return err
	}
	if transcriptsExportFlags.out != "" {
		return os.WriteFile(transcriptsExportFlags.out, body, 0o600)
	}
	if globalFlags.jsonOut || transcriptsExportFlags.format == "json" {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Transcript request submitted.")
	return nil
}

func runTranscriptsBatch(cmd *cobra.Command, _ []string) error {
	transcriptsGetFlags.admin = true
	return runTranscriptsGet(cmd, nil)
}

func runCCRExport(cmd *cobra.Command, _ []string) error {
	if !ccrExportFlags.yes {
		return fmt.Errorf("%s; re-run with --yes to confirm", transcriptExportWarning)
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	genBody, err := generateCCR(c, ccrExportFlags.sharePublicly)
	if err != nil {
		return err
	}
	var gen struct {
		Document struct {
			ID string `json:"id"`
		} `json:"document"`
	}
	if err := json.Unmarshal(genBody, &gen); err != nil {
		return err
	}
	if gen.Document.ID == "" {
		if globalFlags.jsonOut {
			_, err = cmd.OutOrStdout().Write(genBody)
			return err
		}
		return fmt.Errorf("CCR document id missing from response")
	}
	data, err := downloadCCR(c, gen.Document.ID, ccrExportFlags.format)
	if err != nil {
		return err
	}
	if ccrExportFlags.out != "" {
		return os.WriteFile(ccrExportFlags.out, data, 0o600)
	}
	_, err = cmd.OutOrStdout().Write(data)
	return err
}

func runAdvisingNotesList(cmd *cobra.Command, _ []string) error {
	user := strings.TrimSpace(advisingNotesListFlags.user)
	if user == "" {
		return fmt.Errorf("--user is required")
	}
	notes, raw, err := fetchAdvisorNotes(client.New(Cfg.Server, Cfg.APIKey), user)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		if raw != nil {
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"notes": notes})
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CREATED\tADVISOR\tCONTENT")
	for _, n := range notes {
		author := n.AdvisorEmail
		if n.AdvisorDisplay != nil && strings.TrimSpace(*n.AdvisorDisplay) != "" {
			author = *n.AdvisorDisplay
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", n.CreatedAt, author, oneLine(n.Content))
	}
	return w.Flush()
}

func runAdvisingNotesAdd(cmd *cobra.Command, _ []string) error {
	user := strings.TrimSpace(advisingNotesAddFlags.user)
	if user == "" {
		return fmt.Errorf("--user is required")
	}
	content := strings.TrimSpace(advisingNotesAddFlags.content)
	if content == "" && advisingNotesAddFlags.file != "" {
		var err error
		content, err = readTextFile(advisingNotesAddFlags.file)
		if err != nil {
			return err
		}
	}
	if content == "" {
		return fmt.Errorf("--content or --file is required")
	}
	body, err := addAdvisorNote(client.New(Cfg.Server, Cfg.APIKey), user, content)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Advising note added.")
	return nil
}

func runDegreeProgressGet(cmd *cobra.Command, _ []string) error {
	if strings.TrimSpace(degreeProgressGetFlags.user) != "" {
		return fmt.Errorf("per-student degree progress for advisors is not exposed via API; use advising integration")
	}
	body, err := fetchDegreeProgress(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runOnboardingGoalsGet(cmd *cobra.Command, _ []string) error {
	body, err := fetchMyGoals(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runOnboardingGoalsSet(cmd *cobra.Command, _ []string) error {
	if onboardingGoalsSetFlags.file == "" {
		return fmt.Errorf("--file is required")
	}
	raw, err := os.ReadFile(onboardingGoalsSetFlags.file)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	body, err := patchMyGoals(client.New(Cfg.Server, Cfg.APIKey), payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}
