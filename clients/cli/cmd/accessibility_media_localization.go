package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// --- alt-text ---

var altTextCmd = &cobra.Command{
	Use:   "alt-text",
	Short: "Alt-text suggestions and coverage",
}

var altTextGenerateFlags struct {
	course       string
	imageURL     string
	language     string
	wait         bool
	skipExisting bool
}

var altTextGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Suggest alt text for an image URL in a course",
	RunE:  runAltTextGenerate,
}

var altTextListFlags struct {
	course string
}

var altTextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alt-text coverage for a course",
	RunE:  runAltTextList,
}

var altTextSetFlags struct {
	course   string
	imageURL string
	text     string
}

var altTextSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Suggest and apply alt text for one image",
	RunE:  runAltTextSet,
}

// --- captions ---

var captionsCmd = &cobra.Command{
	Use:   "captions",
	Short: "Video captions",
}

var captionsGenerateFlags struct {
	item string
	wait bool
}

var captionsGenerateCmd = &cobra.Command{
	Use:   "generate <file_object_id>",
	Short: "Trigger caption generation for a media file",
	Args:  cobra.ExactArgs(1),
	RunE:  runCaptionsGenerate,
}

var captionsListCmd = &cobra.Command{
	Use:   "list <file_object_id>",
	Short: "List captions for a media file",
	Args:  cobra.ExactArgs(1),
	RunE:  runCaptionsList,
}

var captionsUploadFlags struct {
	file string
	lang string
}

var captionsUploadCmd = &cobra.Command{
	Use:   "upload <file_object_id>",
	Short: "Upload a WebVTT caption track",
	Args:  cobra.ExactArgs(1),
	RunE:  runCaptionsUpload,
}

var captionsDeleteCmd = &cobra.Command{
	Use:   "delete <file_object_id> <caption_id>",
	Short: "Delete a caption track",
	Args:  cobra.ExactArgs(2),
	RunE:  runCaptionsDelete,
}

// --- translations ---

var translationsCmd = &cobra.Command{
	Use:   "translations",
	Short: "Course content translations",
}

var translationsGenerateFlags struct {
	course string
	item   string
	to     string
	wait   bool
}

var translationsGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "AI-draft translations for a course item",
	RunE:  runTranslationsGenerate,
}

var translationsListFlags struct {
	course string
}

var translationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List translation items for a course",
	RunE:  runTranslationsList,
}

var translationsCoverageFlags struct {
	course string
}

var translationsCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "Per-locale translation coverage for a course",
	RunE:  runTranslationsCoverage,
}

var translationsSetFlags struct {
	course string
	item   string
	to     string
	text   string
	file   string
}

var translationsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set translated text for a course item",
	RunE:  runTranslationsSet,
}

// --- accessibility ---

var accessibilityCmd = &cobra.Command{
	Use:   "accessibility",
	Short: "WCAG accessibility checks",
}

var accessibilityCheckCmd = &cobra.Command{
	Use:   "check <course>",
	Short: "Report alt-text coverage and uncovered items",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccessibilityCheck,
}

// --- media ---

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Media transcode and upload helpers",
}

var mediaTranscodeFlags struct {
	wait    bool
	timeout time.Duration
}

var mediaTranscodeCmd = &cobra.Command{
	Use:   "transcode <file_object_id>",
	Short: "Trigger or poll video transcode status",
	Args:  cobra.ExactArgs(1),
	RunE:  runMediaTranscode,
}

// --- tts ---

var ttsCmd = &cobra.Command{
	Use:   "tts",
	Short: "Text-to-speech synthesis",
}

var ttsSynthFlags struct {
	file string
	text string
	out  string
}

var ttsSynthCmd = &cobra.Command{
	Use:   "synth",
	Short: "Synthesize speech from text",
	RunE:  runTTSSynth,
}

// --- reading-level ---

var readingLevelCmd = &cobra.Command{
	Use:   "reading-level",
	Short: "Reading level analysis for course items",
}

var readingLevelGetFlags struct {
	course string
}

var readingLevelGetCmd = &cobra.Command{
	Use:   "get <item_id>",
	Short: "Get FKGL/FRE reading level for an item",
	Args:  cobra.ExactArgs(1),
	RunE:  runReadingLevelGet,
}

func init() {
	altTextGenerateCmd.Flags().StringVar(&altTextGenerateFlags.course, "course", "", "course code (required)")
	altTextGenerateCmd.Flags().StringVar(&altTextGenerateFlags.imageURL, "image-url", "", "image URL (required)")
	altTextGenerateCmd.Flags().StringVar(&altTextGenerateFlags.language, "language", "en", "language hint")
	altTextGenerateCmd.Flags().BoolVar(&altTextGenerateFlags.wait, "wait", false, "wait for suggestion")
	altTextGenerateCmd.Flags().BoolVar(&altTextGenerateFlags.skipExisting, "skip-existing", false, "skip when suggestion exists")

	altTextListCmd.Flags().StringVar(&altTextListFlags.course, "course", "", "course code (required)")
	altTextSetCmd.Flags().StringVar(&altTextSetFlags.course, "course", "", "course code (required)")
	altTextSetCmd.Flags().StringVar(&altTextSetFlags.imageURL, "image-url", "", "image URL (required)")
	altTextSetCmd.Flags().StringVar(&altTextSetFlags.text, "text", "", "alt text (optional; uses AI when empty)")

	captionsGenerateCmd.Flags().BoolVar(&captionsGenerateFlags.wait, "wait", false, "poll until captions ready")
	captionsUploadCmd.Flags().StringVar(&captionsUploadFlags.file, "file", "", "WebVTT file (required)")
	captionsUploadCmd.Flags().StringVar(&captionsUploadFlags.lang, "lang", "en", "caption language")

	translationsGenerateCmd.Flags().StringVar(&translationsGenerateFlags.course, "course", "", "course code (required)")
	translationsGenerateCmd.Flags().StringVar(&translationsGenerateFlags.item, "item", "", "structure item id (required)")
	translationsGenerateCmd.Flags().StringVar(&translationsGenerateFlags.to, "to", "", "target locale (required)")
	translationsGenerateCmd.Flags().BoolVar(&translationsGenerateFlags.wait, "wait", false, "wait for draft")

	translationsListCmd.Flags().StringVar(&translationsListFlags.course, "course", "", "course code (required)")
	translationsCoverageCmd.Flags().StringVar(&translationsCoverageFlags.course, "course", "", "course code (required)")

	translationsSetCmd.Flags().StringVar(&translationsSetFlags.course, "course", "", "course code (required)")
	translationsSetCmd.Flags().StringVar(&translationsSetFlags.item, "item", "", "structure item id (required)")
	translationsSetCmd.Flags().StringVar(&translationsSetFlags.to, "to", "", "target locale (required)")
	translationsSetCmd.Flags().StringVar(&translationsSetFlags.text, "text", "", "translated text")
	translationsSetCmd.Flags().StringVar(&translationsSetFlags.file, "file", "", "translated text file")

	mediaTranscodeCmd.Flags().BoolVar(&mediaTranscodeFlags.wait, "wait", false, "poll until transcode completes")
	mediaTranscodeCmd.Flags().DurationVar(&mediaTranscodeFlags.timeout, "timeout", 10*time.Minute, "wait timeout")

	ttsSynthCmd.Flags().StringVar(&ttsSynthFlags.file, "file", "", "text file")
	ttsSynthCmd.Flags().StringVar(&ttsSynthFlags.text, "text", "", "inline text")
	ttsSynthCmd.Flags().StringVar(&ttsSynthFlags.out, "out", "", "output audio path")

	readingLevelGetCmd.Flags().StringVar(&readingLevelGetFlags.course, "course", "", "course code (required)")

	altTextCmd.AddCommand(altTextGenerateCmd, altTextListCmd, altTextSetCmd)
	captionsCmd.AddCommand(captionsGenerateCmd, captionsListCmd, captionsUploadCmd, captionsDeleteCmd)
	translationsCmd.AddCommand(translationsGenerateCmd, translationsListCmd, translationsCoverageCmd, translationsSetCmd)
	accessibilityCmd.AddCommand(accessibilityCheckCmd)
	mediaCmd.AddCommand(mediaTranscodeCmd)
	ttsCmd.AddCommand(ttsSynthCmd)
	readingLevelCmd.AddCommand(readingLevelGetCmd)

	rootCmd.AddCommand(altTextCmd, captionsCmd, translationsCmd, accessibilityCmd, mediaCmd, ttsCmd, readingLevelCmd)
}

func runAltTextGenerate(cmd *cobra.Command, _ []string) error {
	if altTextGenerateFlags.course == "" || altTextGenerateFlags.imageURL == "" {
		return fmt.Errorf("--course and --image-url are required")
	}
	body, err := suggestAltText(client.New(Cfg.Server, Cfg.APIKey), altTextGenerateFlags.course, altTextGenerateFlags.imageURL, altTextGenerateFlags.language)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAltTextList(cmd *cobra.Command, _ []string) error {
	course := strings.TrimSpace(altTextListFlags.course)
	if course == "" {
		return fmt.Errorf("--course is required")
	}
	body, err := fetchCourseAccessibility(client.New(Cfg.Server, Cfg.APIKey), course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAltTextSet(cmd *cobra.Command, _ []string) error {
	if altTextSetFlags.course == "" || altTextSetFlags.imageURL == "" {
		return fmt.Errorf("--course and --image-url are required")
	}
	text := altTextSetFlags.text
	if text == "" {
		body, err := suggestAltText(client.New(Cfg.Server, Cfg.APIKey), altTextSetFlags.course, altTextSetFlags.imageURL, "en")
		if err != nil {
			return err
		}
		var out struct {
			Suggestion string `json:"suggestion"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return err
		}
		text = out.Suggestion
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"imageUrl": altTextSetFlags.imageURL,
			"altText":  text,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Alt text: %s\n", text)
	return nil
}

func runCaptionsGenerate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := retriggerCaptions(c, args[0])
	if err != nil {
		return err
	}
	if captionsGenerateFlags.wait {
		for i := 0; i < 60; i++ {
			list, listErr := listCaptions(c, args[0])
			if listErr == nil {
				var tracks []any
				if json.Unmarshal(list, &tracks) == nil && len(tracks) > 0 {
					_, err = cmd.OutOrStdout().Write(list)
					return err
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCaptionsList(cmd *cobra.Command, args []string) error {
	body, err := listCaptions(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCaptionsUpload(cmd *cobra.Command, args []string) error {
	if captionsUploadFlags.file == "" {
		return fmt.Errorf("--file is required")
	}
	vtt, err := os.ReadFile(captionsUploadFlags.file)
	if err != nil {
		return err
	}
	if !isWebVTT(vtt) {
		return fmt.Errorf("caption file must be WebVTT (starts with WEBVTT)")
	}
	body, err := importCaptionVTT(client.New(Cfg.Server, Cfg.APIKey), args[0], vtt, captionsUploadFlags.lang)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCaptionsDelete(cmd *cobra.Command, args []string) error {
	if err := deleteCaption(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deleted": args[1]})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted caption %s\n", args[1])
	return nil
}

func runTranslationsGenerate(cmd *cobra.Command, _ []string) error {
	if translationsGenerateFlags.course == "" || translationsGenerateFlags.item == "" || translationsGenerateFlags.to == "" {
		return fmt.Errorf("--course, --item, and --to are required")
	}
	if err := validateLocale(translationsGenerateFlags.to); err != nil {
		return err
	}
	body, err := draftCourseTranslation(client.New(Cfg.Server, Cfg.APIKey), translationsGenerateFlags.course, translationsGenerateFlags.item, translationsGenerateFlags.to)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTranslationsList(cmd *cobra.Command, _ []string) error {
	course := strings.TrimSpace(translationsListFlags.course)
	if course == "" {
		return fmt.Errorf("--course is required")
	}
	body, err := listCourseTranslations(client.New(Cfg.Server, Cfg.APIKey), course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTranslationsCoverage(cmd *cobra.Command, _ []string) error {
	course := strings.TrimSpace(translationsCoverageFlags.course)
	if course == "" {
		return fmt.Errorf("--course is required")
	}
	body, err := fetchTranslationCoverage(client.New(Cfg.Server, Cfg.APIKey), course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTranslationsSet(cmd *cobra.Command, _ []string) error {
	if translationsSetFlags.course == "" || translationsSetFlags.item == "" || translationsSetFlags.to == "" {
		return fmt.Errorf("--course, --item, and --to are required")
	}
	if err := validateLocale(translationsSetFlags.to); err != nil {
		return err
	}
	text := translationsSetFlags.text
	if text == "" && translationsSetFlags.file != "" {
		raw, err := os.ReadFile(translationsSetFlags.file)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(string(raw))
	}
	if text == "" {
		return fmt.Errorf("--text or --file is required")
	}
	body, err := setCourseTranslation(client.New(Cfg.Server, Cfg.APIKey), translationsSetFlags.course, translationsSetFlags.item, translationsSetFlags.to, text)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runAccessibilityCheck(cmd *cobra.Command, args []string) error {
	body, err := fetchCourseAccessibility(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		AltTextCoverage struct {
			WithAlt int `json:"withAlt"`
			Total   int `json:"total"`
			Percent int `json:"percent"`
		} `json:"altTextCoverage"`
	}
	if json.Unmarshal(body, &out) == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Alt-text coverage: %d%% (%d/%d images)\n",
			out.AltTextCoverage.Percent, out.AltTextCoverage.WithAlt, out.AltTextCoverage.Total)
		return nil
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runMediaTranscode(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if _, err := retranscodeFile(c, args[0]); err != nil {
		// non-admin callers may only poll status
		_ = err
	}
	if mediaTranscodeFlags.wait {
		body, err := waitForTranscode(c, args[0], mediaTranscodeFlags.timeout)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	body, err := fetchTranscodeStatus(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTTSSynth(cmd *cobra.Command, _ []string) error {
	text := strings.TrimSpace(ttsSynthFlags.text)
	if text == "" && ttsSynthFlags.file != "" {
		raw, err := os.ReadFile(ttsSynthFlags.file)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(string(raw))
	}
	if text == "" {
		return fmt.Errorf("--text or --file is required")
	}
	audio, err := synthesizeTTS(client.New(Cfg.Server, Cfg.APIKey), text)
	if err != nil {
		return err
	}
	if ttsSynthFlags.out != "" {
		return os.WriteFile(ttsSynthFlags.out, audio, 0o600)
	}
	_, err = cmd.OutOrStdout().Write(audio)
	return err
}

func runReadingLevelGet(cmd *cobra.Command, args []string) error {
	course := strings.TrimSpace(readingLevelGetFlags.course)
	if course == "" {
		return fmt.Errorf("--course is required")
	}
	body, err := fetchItemReadingLevel(client.New(Cfg.Server, Cfg.APIKey), course, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out struct {
		FKGL *float64 `json:"fkgl"`
		FRE  *float64 `json:"fre"`
	}
	if json.Unmarshal(body, &out) == nil && out.FKGL != nil {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintf(w, "FKGL\t%.1f\n", *out.FKGL)
		if out.FRE != nil {
			_, _ = fmt.Fprintf(w, "FRE\t%.1f\n", *out.FRE)
		}
		return w.Flush()
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}
