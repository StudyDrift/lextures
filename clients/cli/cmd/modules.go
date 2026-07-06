package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var validItemKinds = map[string]string{
	"assignment":    "assignments",
	"quiz":          "quizzes",
	"content_page":  "content-pages",
	"page":          "content-pages",
	"external_link": "external-links",
	"link":          "external-links",
	"heading":       "headings",
}

type modulePatchOpts struct {
	title       string
	published   *bool
	visibleFrom string
}

type structureItemPatchOpts struct {
	title     *string
	published *bool
}

var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Manage course modules and module items",
}

var modulesListCmd = &cobra.Command{
	Use:   "list <course_code>",
	Short: "List modules in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runModulesList,
}

var modulesCreateFlags struct {
	title string
}

var modulesCreateCmd = &cobra.Command{
	Use:   "create <course_code>",
	Short: "Create a module",
	Args:  cobra.ExactArgs(1),
	RunE:  runModulesCreate,
}

var modulesUpdateFlags struct {
	title       string
	published   string
	visibleFrom string
}

var modulesUpdateCmd = &cobra.Command{
	Use:   "update <course_code> <module_id>",
	Short: "Update a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runModulesUpdate,
}

var modulesDeleteCmd = &cobra.Command{
	Use:   "delete <course_code> <module_id>",
	Short: "Delete or archive a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runModulesDelete,
}

var modulesReorderFlags struct {
	order string
}

var modulesReorderCmd = &cobra.Command{
	Use:   "reorder <course_code>",
	Short: "Reorder modules",
	Args:  cobra.ExactArgs(1),
	RunE:  runModulesReorder,
}

var modulesItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Add, remove, or reorder items within modules",
}

var modulesItemsAddFlags struct {
	typeFlag string
	title    string
	url      string
}

var modulesItemsAddCmd = &cobra.Command{
	Use:   "add <course_code> <module_id>",
	Short: "Add an item to a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runModulesItemsAdd,
}

var modulesItemsRemoveCmd = &cobra.Command{
	Use:   "remove <course_code> <item_id>",
	Short: "Remove (archive) an item from a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runModulesItemsRemove,
}

var modulesItemsReorderFlags struct {
	order string
}

var modulesItemsReorderCmd = &cobra.Command{
	Use:   "reorder <course_code> <module_id>",
	Short: "Reorder items within a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runModulesItemsReorder,
}

var modulesSetRequirementsFlags struct {
	file            string
	completionMode  string
	prerequisites   []string
	unlockAt        string
}

var modulesSetRequirementsCmd = &cobra.Command{
	Use:   "set-requirements <course_code> <module_id>",
	Short: "Set module prerequisites and conditional-release rules",
	Args:  cobra.ExactArgs(2),
	RunE:  runModulesSetRequirements,
}

func init() {
	modulesCreateCmd.Flags().StringVar(&modulesCreateFlags.title, "title", "", "module title (required)")
	_ = modulesCreateCmd.MarkFlagRequired("title")

	modulesUpdateCmd.Flags().StringVar(&modulesUpdateFlags.title, "title", "", "module title")
	modulesUpdateCmd.Flags().StringVar(&modulesUpdateFlags.published, "published", "", "published state: true or false")
	modulesUpdateCmd.Flags().StringVar(&modulesUpdateFlags.visibleFrom, "visible-from", "", "visibility start (RFC3339)")

	modulesReorderCmd.Flags().StringVar(&modulesReorderFlags.order, "order", "", "comma-separated module ids in desired order (required)")
	_ = modulesReorderCmd.MarkFlagRequired("order")

	modulesItemsAddCmd.Flags().StringVar(&modulesItemsAddFlags.typeFlag, "type", "", "item type: assignment, quiz, page, link, heading (required)")
	modulesItemsAddCmd.Flags().StringVar(&modulesItemsAddFlags.title, "title", "", "item title (required)")
	modulesItemsAddCmd.Flags().StringVar(&modulesItemsAddFlags.url, "url", "", "URL (required for link type)")
	_ = modulesItemsAddCmd.MarkFlagRequired("type")
	_ = modulesItemsAddCmd.MarkFlagRequired("title")

	modulesItemsReorderCmd.Flags().StringVar(&modulesItemsReorderFlags.order, "order", "", "comma-separated item ids in desired order (required)")
	_ = modulesItemsReorderCmd.MarkFlagRequired("order")

	modulesSetRequirementsCmd.Flags().StringVar(&modulesSetRequirementsFlags.file, "file", "", "JSON requirements file (use - for stdin)")
	modulesSetRequirementsCmd.Flags().StringVar(&modulesSetRequirementsFlags.completionMode, "completion-mode", "", "completion mode: all_items, one_item, sequential_order")
	modulesSetRequirementsCmd.Flags().StringSliceVar(&modulesSetRequirementsFlags.prerequisites, "prerequisite", nil, "prerequisite module id (repeatable)")
	modulesSetRequirementsCmd.Flags().StringVar(&modulesSetRequirementsFlags.unlockAt, "unlock-at", "", "unlock time (RFC3339)")

	modulesItemsCmd.AddCommand(modulesItemsAddCmd, modulesItemsRemoveCmd, modulesItemsReorderCmd)
	modulesCmd.AddCommand(
		modulesListCmd,
		modulesCreateCmd,
		modulesUpdateCmd,
		modulesDeleteCmd,
		modulesReorderCmd,
		modulesItemsCmd,
		modulesSetRequirementsCmd,
	)
	rootCmd.AddCommand(modulesCmd)
}

func validateItemKind(kind string) (string, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	segment, ok := validItemKinds[kind]
	if !ok {
		return "", fmt.Errorf("invalid --type %q: must be one of assignment, quiz, page, link, heading", kind)
	}
	return segment, nil
}

func createModule(c *client.Client, courseCode, title string) (structureItemPublic, error) {
	body, _ := json.Marshal(map[string]string{"title": title})
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/structure/modules", bytes.NewReader(body))
	if err != nil {
		return structureItemPublic{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return structureItemPublic{}, fmt.Errorf("creating module: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return structureItemPublic{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return structureItemPublic{}, apiErrorBody(resp.StatusCode, respBody)
	}
	var item structureItemPublic
	if err := json.Unmarshal(respBody, &item); err != nil {
		return structureItemPublic{}, fmt.Errorf("decoding response: %w", err)
	}
	return item, nil
}

func patchModule(c *client.Client, courseCode, moduleID string, opts modulePatchOpts) error {
	body := map[string]any{}
	if opts.title != "" {
		body["title"] = opts.title
	}
	if opts.published != nil {
		body["published"] = *opts.published
	}
	if opts.visibleFrom != "" {
		body["visibleFrom"] = opts.visibleFrom
	}
	if len(body) == 0 {
		return fmt.Errorf("no fields to update")
	}
	raw, _ := json.Marshal(body)
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating module: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return nil
}

func deleteModule(c *client.Client, courseCode, moduleID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/courses/"+courseCode+"/structure/modules/"+moduleID, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("deleting module: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return nil
}

func addItemToModule(c *client.Client, courseCode, moduleID, kind, title, url string) (structureItemPublic, error) {
	segment, err := validateItemKind(kind)
	if err != nil {
		return structureItemPublic{}, err
	}
	var body map[string]string
	if segment == "external-links" {
		if strings.TrimSpace(url) == "" {
			return structureItemPublic{}, fmt.Errorf("url is required for link items")
		}
		body = map[string]string{"title": title, "url": url}
	} else {
		body = map[string]string{"title": title}
	}
	raw, _ := json.Marshal(body)
	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/%s", courseCode, moduleID, segment)
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return structureItemPublic{}, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return structureItemPublic{}, fmt.Errorf("adding item: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return structureItemPublic{}, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return structureItemPublic{}, apiErrorBody(resp.StatusCode, respBody)
	}
	var item structureItemPublic
	if err := json.Unmarshal(respBody, &item); err != nil {
		return structureItemPublic{}, fmt.Errorf("decoding response: %w", err)
	}
	return item, nil
}

func patchStructureItem(c *client.Client, courseCode, itemID string, opts structureItemPatchOpts) error {
	body := map[string]any{}
	if opts.title != nil {
		body["title"] = *opts.title
	}
	if opts.published != nil {
		body["published"] = *opts.published
	}
	if len(body) == 0 {
		return nil
	}
	raw, _ := json.Marshal(body)
	req, err := c.NewRequest(http.MethodPatch, "/api/v1/courses/"+courseCode+"/structure/items/"+itemID, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("updating item: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return nil
}

func deleteStructureItem(c *client.Client, courseCode, itemID string) error {
	req, err := c.NewRequest(http.MethodDelete, "/api/v1/courses/"+courseCode+"/structure/items/"+itemID, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("removing item: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return nil
}

func runModulesList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchCourseStructure(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	modules := filterModules(body.Items)
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(modules)
	}
	if len(modules) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No modules.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tPUBLISHED\tORDER")
	for _, m := range modules {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\t%d\n", m.ID, m.Title, m.Published, m.SortOrder)
	}
	_ = w.Flush()
	_ = raw
	return nil
}

func runModulesCreate(cmd *cobra.Command, args []string) error {
	item, err := createModule(client.New(Cfg.Server, Cfg.APIKey), args[0], modulesCreateFlags.title)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created module %s (%s)\n", item.Title, item.ID)
	return nil
}

func runModulesUpdate(cmd *cobra.Command, args []string) error {
	opts := modulePatchOpts{title: modulesUpdateFlags.title, visibleFrom: modulesUpdateFlags.visibleFrom}
	if modulesUpdateFlags.published != "" {
		switch modulesUpdateFlags.published {
		case "true":
			v := true
			opts.published = &v
		case "false":
			v := false
			opts.published = &v
		default:
			return fmt.Errorf("--published must be true or false")
		}
	}
	if err := patchModule(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1], opts); err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, fmt.Sprintf("Updated module %s", args[1]))
}

func runModulesDelete(cmd *cobra.Command, args []string) error {
	if err := deleteModule(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, fmt.Sprintf("Deleted module %s", args[1]))
}

func runModulesReorder(cmd *cobra.Command, args []string) error {
	ids, err := parseOrderFlag(modulesReorderFlags.order)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	current, _, err := fetchCourseStructure(c, args[0])
	if err != nil {
		return err
	}
	childOrder := make(map[string][]string)
	for _, modID := range ids {
		var children []string
		for _, ch := range filterChildren(current.Items, modID) {
			children = append(children, ch.ID)
		}
		if len(children) > 0 {
			childOrder[modID] = children
		}
	}
	_, err = postStructureReorder(c, args[0], map[string]any{
		"moduleOrder":        ids,
		"childOrderByModule": childOrder,
	})
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, fmt.Sprintf("Reordered %d modules in %s", len(ids), args[0]))
}

func runModulesItemsAdd(cmd *cobra.Command, args []string) error {
	item, err := addItemToModule(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1],
		modulesItemsAddFlags.typeFlag, modulesItemsAddFlags.title, modulesItemsAddFlags.url)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(item)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %s %s (%s)\n", item.Kind, item.Title, item.ID)
	return nil
}

func runModulesItemsRemove(cmd *cobra.Command, args []string) error {
	if err := deleteStructureItem(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, fmt.Sprintf("Removed item %s", args[1]))
}

func runModulesItemsReorder(cmd *cobra.Command, args []string) error {
	ids, err := parseOrderFlag(modulesItemsReorderFlags.order)
	if err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	current, _, err := fetchCourseStructure(c, args[0])
	if err != nil {
		return err
	}
	moduleOrder := make([]string, 0)
	for _, m := range filterModules(current.Items) {
		moduleOrder = append(moduleOrder, m.ID)
	}
	_, err = postStructureReorder(c, args[0], map[string]any{
		"moduleOrder":        moduleOrder,
		"childOrderByModule": map[string][]string{args[1]: ids},
	})
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, fmt.Sprintf("Reordered %d items in module %s", len(ids), args[1]))
}

func runModulesSetRequirements(cmd *cobra.Command, args []string) error {
	var body map[string]any
	if modulesSetRequirementsFlags.file != "" {
		raw, err := readInputFile(modulesSetRequirementsFlags.file)
		if err != nil {
			return fmt.Errorf("reading requirements file: %w", err)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return fmt.Errorf("decoding requirements file: %w", err)
		}
	} else {
		body = map[string]any{}
		if modulesSetRequirementsFlags.completionMode != "" {
			body["completionMode"] = modulesSetRequirementsFlags.completionMode
		}
		if len(modulesSetRequirementsFlags.prerequisites) > 0 {
			body["prerequisiteModuleIds"] = modulesSetRequirementsFlags.prerequisites
		}
		if modulesSetRequirementsFlags.unlockAt != "" {
			body["unlockAt"] = modulesSetRequirementsFlags.unlockAt
		}
		if len(body) == 0 {
			return fmt.Errorf("provide --file or at least one of --completion-mode, --prerequisite, --unlock-at")
		}
	}
	raw, _ := json.Marshal(body)
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := fmt.Sprintf("/api/v1/courses/%s/structure/modules/%s/requirements", args[0], args[1])
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return fmt.Errorf("setting requirements: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, respBody)
	}
	return emitRawOrMessage(cmd, respBody, fmt.Sprintf("Updated requirements for module %s", args[1]))
}