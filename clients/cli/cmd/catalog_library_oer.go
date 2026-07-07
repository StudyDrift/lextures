package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

// --- catalog ---

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Public course catalog and institutional sections",
}

var catalogListFlags struct {
	org    string
	public bool
	q      string
	limit  int
}

var catalogListCmd = &cobra.Command{
	Use:   "list",
	Short: "List catalog courses or institutional sections",
	RunE:  runCatalogList,
}

var catalogGetFlags struct {
	public bool
}

var catalogGetCmd = &cobra.Command{
	Use:   "get <course_or_slug>",
	Short: "Get a catalog listing for a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runCatalogGet,
}

var catalogPublishFlags struct {
	course   string
	category string
	language string
	price    int
	slug     string
	file     string
}

var catalogPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a course to the public catalog",
	RunE:  runCatalogPublish,
}

var catalogUnpublishFlags struct {
	course string
}

var catalogUnpublishCmd = &cobra.Command{
	Use:   "unpublish",
	Short: "Remove a course from the public catalog",
	RunE:  runCatalogUnpublish,
}

// --- library ---

var libraryCmd = &cobra.Command{
	Use:   "library",
	Short: "Institutional library and Alma search",
}

var librarySearchFlags struct {
	q string
}

var librarySearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the Alma library catalog",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLibrarySearch,
}

var libraryListFlags struct {
	org string
	q   string
}

var libraryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List org library books",
	RunE:  runLibraryList,
}

var libraryLinkFlags struct {
	course   string
	module   string
	title    string
	url      string
	resource string
}

var libraryLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link a library resource into a course module",
	RunE:  runLibraryLink,
}

var libraryUnlinkFlags struct {
	course string
	item   string
}

var libraryUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Record library unlink (metadata patch placeholder)",
	RunE:  runLibraryUnlink,
}

// --- oer ---

var oerCmd = &cobra.Command{
	Use:   "oer",
	Short: "Open educational resources",
}

var oerSearchFlags struct {
	provider string
	q        string
	subject  string
	level    string
}

var oerSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search OER providers",
	RunE:  runOERSearch,
}

var oerProvidersCmd = &cobra.Command{
	Use:   "providers list",
	Short: "List enabled OER providers",
	RunE:  runOERProvidersList,
}

var oerLinkFlags struct {
	course   string
	module   string
	provider string
	title    string
	url      string
}

var oerLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Attach an OER resource to a course module",
	RunE:  runOERLink,
}

// --- textbooks ---

var textbooksCmd = &cobra.Command{
	Use:   "textbooks",
	Short: "Bookstore textbook resources",
}

var textbooksListFlags struct {
	course string
	item   string
}

var textbooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "Get textbook resource metadata for a course item",
	RunE:  runTextbooksList,
}

var textbooksSetFlags struct {
	course string
	item   string
	file   string
}

var textbooksSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update textbook resource metadata from JSON",
	RunE:  runTextbooksSet,
}

var inclusiveAccessCmd = &cobra.Command{
	Use:   "inclusive-access",
	Short: "Inclusive access program settings",
}

var inclusiveAccessGetFlags struct {
	course string
}

var inclusiveAccessGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get inclusive access settings for a course",
	RunE:  runInclusiveAccessGet,
}

var inclusiveAccessSetFlags struct {
	course string
	file   string
}

var inclusiveAccessSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set inclusive access settings from JSON",
	RunE:  runInclusiveAccessSet,
}

func init() {
	catalogListCmd.Flags().StringVar(&catalogListFlags.org, "org", "", "org id (informational for sections list)")
	catalogListCmd.Flags().BoolVar(&catalogListFlags.public, "public", false, "read public catalog (no auth)")
	catalogListCmd.Flags().StringVar(&catalogListFlags.q, "q", "", "search query")
	catalogListCmd.Flags().IntVar(&catalogListFlags.limit, "limit", 50, "page size")

	catalogGetCmd.Flags().BoolVar(&catalogGetFlags.public, "public", false, "fetch public catalog detail by slug")

	catalogPublishCmd.Flags().StringVar(&catalogPublishFlags.course, "course", "", "course code (required)")
	catalogPublishCmd.Flags().StringVar(&catalogPublishFlags.category, "category", "", "catalog category")
	catalogPublishCmd.Flags().StringVar(&catalogPublishFlags.language, "language", "en", "catalog language")
	catalogPublishCmd.Flags().IntVar(&catalogPublishFlags.price, "price", 0, "price in cents")
	catalogPublishCmd.Flags().StringVar(&catalogPublishFlags.slug, "slug", "", "public slug")
	catalogPublishCmd.Flags().StringVar(&catalogPublishFlags.file, "file", "", "listing JSON file")

	catalogUnpublishCmd.Flags().StringVar(&catalogUnpublishFlags.course, "course", "", "course code (required)")

	librarySearchCmd.Flags().StringVar(&librarySearchFlags.q, "q", "", "search query")

	libraryListCmd.Flags().StringVar(&libraryListFlags.org, "org", "", "org id (required)")
	libraryListCmd.Flags().StringVar(&libraryListFlags.q, "q", "", "filter titles client-side")

	libraryLinkCmd.Flags().StringVar(&libraryLinkFlags.course, "course", "", "course code (required)")
	libraryLinkCmd.Flags().StringVar(&libraryLinkFlags.module, "module", "", "module id (required)")
	libraryLinkCmd.Flags().StringVar(&libraryLinkFlags.title, "title", "", "resource title (required)")
	libraryLinkCmd.Flags().StringVar(&libraryLinkFlags.url, "url", "", "source URL (required)")
	libraryLinkCmd.Flags().StringVar(&libraryLinkFlags.resource, "resource", "article", "resource type")

	libraryUnlinkCmd.Flags().StringVar(&libraryUnlinkFlags.course, "course", "", "course code (required)")
	libraryUnlinkCmd.Flags().StringVar(&libraryUnlinkFlags.item, "item", "", "structure item id (required)")

	oerSearchCmd.Flags().StringVar(&oerSearchFlags.provider, "provider", "", "OER provider id (required)")
	oerSearchCmd.Flags().StringVar(&oerSearchFlags.q, "q", "", "search query")
	oerSearchCmd.Flags().StringVar(&oerSearchFlags.subject, "subject", "", "subject filter")
	oerSearchCmd.Flags().StringVar(&oerSearchFlags.level, "level", "", "level filter")

	oerLinkCmd.Flags().StringVar(&oerLinkFlags.course, "course", "", "course code (required)")
	oerLinkCmd.Flags().StringVar(&oerLinkFlags.module, "module", "", "module id (required)")
	oerLinkCmd.Flags().StringVar(&oerLinkFlags.provider, "provider", "", "OER provider (required)")
	oerLinkCmd.Flags().StringVar(&oerLinkFlags.title, "title", "", "resource title (required)")
	oerLinkCmd.Flags().StringVar(&oerLinkFlags.url, "url", "", "resource URL (required)")

	textbooksListCmd.Flags().StringVar(&textbooksListFlags.course, "course", "", "course code (required)")
	textbooksListCmd.Flags().StringVar(&textbooksListFlags.item, "item", "", "structure item id (required)")

	textbooksSetCmd.Flags().StringVar(&textbooksSetFlags.course, "course", "", "course code (required)")
	textbooksSetCmd.Flags().StringVar(&textbooksSetFlags.item, "item", "", "structure item id (required)")
	textbooksSetCmd.Flags().StringVar(&textbooksSetFlags.file, "file", "", "metadata JSON file (required)")

	inclusiveAccessGetCmd.Flags().StringVar(&inclusiveAccessGetFlags.course, "course", "", "course code (required)")
	inclusiveAccessSetCmd.Flags().StringVar(&inclusiveAccessSetFlags.course, "course", "", "course code (required)")
	inclusiveAccessSetCmd.Flags().StringVar(&inclusiveAccessSetFlags.file, "file", "", "settings JSON file (required)")

	catalogCmd.AddCommand(catalogListCmd, catalogGetCmd, catalogPublishCmd, catalogUnpublishCmd)
	libraryCmd.AddCommand(librarySearchCmd, libraryListCmd, libraryLinkCmd, libraryUnlinkCmd)
	oerCmd.AddCommand(oerSearchCmd, oerProvidersCmd, oerLinkCmd)
	textbooksCmd.AddCommand(textbooksListCmd, textbooksSetCmd)
	inclusiveAccessCmd.AddCommand(inclusiveAccessGetCmd, inclusiveAccessSetCmd)

	rootCmd.AddCommand(catalogCmd, libraryCmd, oerCmd, textbooksCmd, inclusiveAccessCmd)
}

func runCatalogList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	q := url.Values{}
	if catalogListFlags.q != "" {
		q.Set("q", catalogListFlags.q)
	}
	if catalogListFlags.limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", catalogListFlags.limit))
	}
	var body []byte
	var err error
	if catalogListFlags.public {
		body, err = fetchPublicCatalog(c, q)
	} else {
		body, err = fetchCatalogSections(c, q)
	}
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCatalogGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	ref := args[0]
	if catalogGetFlags.public {
		path := "/api/v1/public/catalog/courses/" + url.PathEscape(ref)
		req, err := c.NewRequest("GET", path, nil)
		if err != nil {
			return err
		}
		resp, err := doWithRetry(c, req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := readResponseBody(resp)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return apiErrorBody(resp.StatusCode, body)
		}
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	listing, body, err := fetchCourseCatalogListing(c, ref)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "public=%v slug=%s price=%d\n", listing.IsPublic, listing.Slug, listing.PriceCents)
	return nil
}

func runCatalogPublish(cmd *cobra.Command, _ []string) error {
	course := strings.TrimSpace(catalogPublishFlags.course)
	if course == "" {
		return fmt.Errorf("--course is required")
	}
	listing := catalogListing{IsPublic: true, Language: catalogPublishFlags.language, PriceCents: catalogPublishFlags.price}
	if catalogPublishFlags.category != "" {
		cat := catalogPublishFlags.category
		listing.Category = &cat
	}
	if catalogPublishFlags.slug != "" {
		listing.Slug = catalogPublishFlags.slug
	}
	if catalogPublishFlags.file != "" {
		raw, err := os.ReadFile(catalogPublishFlags.file)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &listing); err != nil {
			return err
		}
		listing.IsPublic = true
	}
	body, err := putCourseCatalogListing(client.New(Cfg.Server, Cfg.APIKey), course, listing)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runCatalogUnpublish(cmd *cobra.Command, _ []string) error {
	course := strings.TrimSpace(catalogUnpublishFlags.course)
	if course == "" {
		return fmt.Errorf("--course is required")
	}
	listing, _, err := fetchCourseCatalogListing(client.New(Cfg.Server, Cfg.APIKey), course)
	if err != nil {
		return err
	}
	listing.IsPublic = false
	body, err := putCourseCatalogListing(client.New(Cfg.Server, Cfg.APIKey), course, listing)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runLibrarySearch(cmd *cobra.Command, args []string) error {
	q := librarySearchFlags.q
	if len(args) > 0 {
		q = args[0]
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return fmt.Errorf("search query is required")
	}
	body, err := searchLibrary(client.New(Cfg.Server, Cfg.APIKey), q)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runLibraryList(cmd *cobra.Command, _ []string) error {
	org := strings.TrimSpace(libraryListFlags.org)
	if org == "" {
		return fmt.Errorf("--org is required")
	}
	books, raw, err := fetchOrgLibrary(client.New(Cfg.Server, Cfg.APIKey), org, nil)
	if err != nil {
		return err
	}
	books = filterLibraryBooks(books, libraryListFlags.q)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"books": books})
	}
	if libraryListFlags.q == "" && raw != nil && !globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE")
	for _, b := range books {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", b.ID, b.Title)
	}
	return w.Flush()
}

func runLibraryLink(cmd *cobra.Command, _ []string) error {
	if libraryLinkFlags.course == "" || libraryLinkFlags.module == "" {
		return fmt.Errorf("--course and --module are required")
	}
	if libraryLinkFlags.title == "" || libraryLinkFlags.url == "" {
		return fmt.Errorf("--title and --url are required")
	}
	payload := map[string]any{
		"title":        libraryLinkFlags.title,
		"resourceType": libraryLinkFlags.resource,
		"sourceUrl":    libraryLinkFlags.url,
		"metadata":     map[string]any{},
	}
	body, err := linkLibraryResource(client.New(Cfg.Server, Cfg.APIKey), libraryLinkFlags.course, libraryLinkFlags.module, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runLibraryUnlink(cmd *cobra.Command, _ []string) error {
	if libraryUnlinkFlags.course == "" || libraryUnlinkFlags.item == "" {
		return fmt.Errorf("--course and --item are required")
	}
	payload := map[string]any{"metadata": map[string]any{"unlinked": true}}
	body, err := patchLibraryResource(client.New(Cfg.Server, Cfg.APIKey), libraryUnlinkFlags.course, libraryUnlinkFlags.item, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runOERSearch(cmd *cobra.Command, _ []string) error {
	if oerSearchFlags.provider == "" {
		return fmt.Errorf("--provider is required")
	}
	q := url.Values{}
	q.Set("provider", oerSearchFlags.provider)
	if oerSearchFlags.q != "" {
		q.Set("q", oerSearchFlags.q)
	}
	if oerSearchFlags.subject != "" {
		q.Set("subject", oerSearchFlags.subject)
	}
	if oerSearchFlags.level != "" {
		q.Set("level", oerSearchFlags.level)
	}
	body, err := searchOER(client.New(Cfg.Server, Cfg.APIKey), q)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runOERProvidersList(cmd *cobra.Command, _ []string) error {
	body, err := fetchOERProviders(client.New(Cfg.Server, Cfg.APIKey))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runOERLink(cmd *cobra.Command, _ []string) error {
	if oerLinkFlags.course == "" || oerLinkFlags.module == "" || oerLinkFlags.provider == "" {
		return fmt.Errorf("--course, --module, and --provider are required")
	}
	if oerLinkFlags.title == "" || oerLinkFlags.url == "" {
		return fmt.Errorf("--title and --url are required")
	}
	payload := map[string]any{
		"title":    oerLinkFlags.title,
		"url":      oerLinkFlags.url,
		"provider": oerLinkFlags.provider,
	}
	body, err := linkOERResource(client.New(Cfg.Server, Cfg.APIKey), oerLinkFlags.course, oerLinkFlags.module, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTextbooksList(cmd *cobra.Command, _ []string) error {
	if textbooksListFlags.course == "" || textbooksListFlags.item == "" {
		return fmt.Errorf("--course and --item are required")
	}
	body, err := fetchTextbookResource(client.New(Cfg.Server, Cfg.APIKey), textbooksListFlags.course, textbooksListFlags.item)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runTextbooksSet(cmd *cobra.Command, _ []string) error {
	if textbooksSetFlags.course == "" || textbooksSetFlags.item == "" || textbooksSetFlags.file == "" {
		return fmt.Errorf("--course, --item, and --file are required")
	}
	raw, err := os.ReadFile(textbooksSetFlags.file)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	body, err := patchTextbookResource(client.New(Cfg.Server, Cfg.APIKey), textbooksSetFlags.course, textbooksSetFlags.item, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runInclusiveAccessGet(cmd *cobra.Command, _ []string) error {
	if inclusiveAccessGetFlags.course == "" {
		return fmt.Errorf("--course is required")
	}
	body, err := fetchInclusiveAccess(client.New(Cfg.Server, Cfg.APIKey), inclusiveAccessGetFlags.course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}

func runInclusiveAccessSet(cmd *cobra.Command, _ []string) error {
	if inclusiveAccessSetFlags.course == "" || inclusiveAccessSetFlags.file == "" {
		return fmt.Errorf("--course and --file are required")
	}
	raw, err := os.ReadFile(inclusiveAccessSetFlags.file)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	body, err := setInclusiveAccess(client.New(Cfg.Server, Cfg.APIKey), inclusiveAccessSetFlags.course, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(body)
	return err
}
