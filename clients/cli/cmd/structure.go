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

// structureApplySpec is the declarative format for `structure apply --file`.
// Modules are ordered; items within each module are ordered.
type structureApplySpec struct {
	Modules []structureApplyModule `json:"modules"`
}

type structureApplyModule struct {
	ID        string                `json:"id,omitempty"`
	Title     string                `json:"title"`
	Published *bool                 `json:"published,omitempty"`
	Items     []structureApplyItem  `json:"items,omitempty"`
}

type structureApplyItem struct {
	ID        string `json:"id,omitempty"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Published *bool  `json:"published,omitempty"`
	URL       string `json:"url,omitempty"`
}

type structureChangeSummary struct {
	Created int
	Updated int
	Deleted int
}

var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Read and sync course structure (modules and items)",
}

var structureGetFlags struct {
	tree bool
}

var structureGetCmd = &cobra.Command{
	Use:   "get <course_code>",
	Short: "Export the course module tree",
	Args:  cobra.ExactArgs(1),
	RunE:  runStructureGet,
}

var structureApplyFlags struct {
	file    string
	dryRun  bool
}

var structureApplyCmd = &cobra.Command{
	Use:   "apply <course_code>",
	Short: "Declaratively sync course structure from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runStructureApply,
}

func init() {
	structureGetCmd.Flags().BoolVar(&structureGetFlags.tree, "tree", false, "print a human-readable module tree")
	structureApplyCmd.Flags().StringVar(&structureApplyFlags.file, "file", "", "JSON structure file (use - for stdin)")
	structureApplyCmd.Flags().BoolVar(&structureApplyFlags.dryRun, "dry-run", false, "preview changes without applying them")
	_ = structureApplyCmd.MarkFlagRequired("file")

	structureCmd.AddCommand(structureGetCmd, structureApplyCmd)
	rootCmd.AddCommand(structureCmd)
}

func fetchCourseStructure(c *client.Client, courseCode string) (courseStructureBody, []byte, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/structure", nil)
	if err != nil {
		return courseStructureBody{}, nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return courseStructureBody{}, nil, fmt.Errorf("getting structure: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return courseStructureBody{}, nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return courseStructureBody{}, body, apiErrorBody(resp.StatusCode, body)
	}
	var out courseStructureBody
	if err := json.Unmarshal(body, &out); err != nil {
		return courseStructureBody{}, body, fmt.Errorf("decoding response: %w", err)
	}
	return out, body, nil
}

func parseOrderFlag(order string) ([]string, error) {
	order = strings.TrimSpace(order)
	if order == "" {
		return nil, fmt.Errorf("--order is required")
	}
	parts := strings.Split(order, ",")
	var ids []string
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			return nil, fmt.Errorf("invalid --order: empty id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func filterModules(items []structureItemPublic) []structureItemPublic {
	var mods []structureItemPublic
	for _, it := range items {
		if it.Kind == "module" {
			mods = append(mods, it)
		}
	}
	return mods
}

func filterChildren(items []structureItemPublic, moduleID string) []structureItemPublic {
	var children []structureItemPublic
	for _, it := range items {
		if it.ParentID != nil && *it.ParentID == moduleID {
			children = append(children, it)
		}
	}
	return children
}

func printStructureTree(w io.Writer, items []structureItemPublic) error {
	modules := filterModules(items)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, mod := range modules {
		pub := ""
		if !mod.Published {
			pub = " (draft)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s%s\n", mod.ID, mod.Title, pub)
		for _, child := range filterChildren(items, mod.ID) {
			childPub := ""
			if !child.Published {
				childPub = " (draft)"
			}
			_, _ = fmt.Fprintf(tw, "  %s\t[%s] %s%s\n", child.ID, child.Kind, child.Title, childPub)
		}
	}
	return tw.Flush()
}

func runStructureGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, raw, err := fetchCourseStructure(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if structureGetFlags.tree {
		return printStructureTree(cmd.OutOrStdout(), body.Items)
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func parseStructureApplyFile(raw []byte) (structureApplySpec, error) {
	var spec structureApplySpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return structureApplySpec{}, fmt.Errorf("decoding structure file: %w", err)
	}
	// Also accept round-tripped GET /structure output.
	if len(spec.Modules) == 0 {
		var envelope courseStructureBody
		if json.Unmarshal(raw, &envelope) == nil && len(envelope.Items) > 0 {
			spec = structureItemsToApplySpec(envelope.Items)
		}
	}
	if len(spec.Modules) == 0 {
		return structureApplySpec{}, fmt.Errorf("structure file must contain modules or items")
	}
	return spec, nil
}

func structureItemsToApplySpec(items []structureItemPublic) structureApplySpec {
	var spec structureApplySpec
	for _, mod := range filterModules(items) {
		pub := mod.Published
		am := structureApplyModule{
			ID:        mod.ID,
			Title:     mod.Title,
			Published: &pub,
		}
		for _, child := range filterChildren(items, mod.ID) {
			cp := child.Published
			ai := structureApplyItem{
				ID:        child.ID,
				Kind:      child.Kind,
				Title:     child.Title,
				Published: &cp,
			}
			am.Items = append(am.Items, ai)
		}
		spec.Modules = append(spec.Modules, am)
	}
	return spec
}

func findModuleByIDOrTitle(modules []structureItemPublic, id, title string) *structureItemPublic {
	for i := range modules {
		if id != "" && modules[i].ID == id {
			return &modules[i]
		}
	}
	if title != "" {
		for i := range modules {
			if modules[i].Title == title {
				return &modules[i]
			}
		}
	}
	return nil
}

func findChildByIDOrTitle(children []structureItemPublic, id, title, kind string) *structureItemPublic {
	for i := range children {
		if id != "" && children[i].ID == id {
			return &children[i]
		}
	}
	if title != "" {
		for i := range children {
			if children[i].Title == title && (kind == "" || children[i].Kind == kind) {
				return &children[i]
			}
		}
	}
	return nil
}

func computeStructureDiff(current []structureItemPublic, desired structureApplySpec) structureChangeSummary {
	var summary structureChangeSummary
	currentMods := filterModules(current)
	desiredIDs := map[string]bool{}
	desiredTitles := map[string]bool{}

	for _, dm := range desired.Modules {
		if dm.ID != "" {
			desiredIDs[dm.ID] = true
		}
		desiredTitles[dm.Title] = true
		ex := findModuleByIDOrTitle(currentMods, dm.ID, dm.Title)
		if ex == nil {
			summary.Created++
			for range dm.Items {
				summary.Created++
			}
			continue
		}
		if dm.Title != "" && dm.Title != ex.Title {
			summary.Updated++
		}
		if dm.Published != nil && *dm.Published != ex.Published {
			summary.Updated++
		}
		curChildren := filterChildren(current, ex.ID)
		for _, di := range dm.Items {
			ch := findChildByIDOrTitle(curChildren, di.ID, di.Title, di.Kind)
			if ch == nil {
				summary.Created++
				continue
			}
			if di.Title != "" && di.Title != ch.Title {
				summary.Updated++
			}
			if di.Published != nil && *di.Published != ch.Published {
				summary.Updated++
			}
		}
		for _, ch := range curChildren {
			found := false
			for _, di := range dm.Items {
				if (di.ID != "" && di.ID == ch.ID) || (di.Title != "" && di.Title == ch.Title && (di.Kind == "" || di.Kind == ch.Kind)) {
					found = true
					break
				}
			}
			if !found {
				summary.Deleted++
			}
		}
	}
	for _, cm := range currentMods {
		if desiredIDs[cm.ID] {
			continue
		}
		if !desiredTitles[cm.Title] {
			summary.Deleted++
			summary.Deleted += len(filterChildren(current, cm.ID))
		}
	}
	return summary
}

func printChangeSummary(w io.Writer, summary structureChangeSummary, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	_, _ = fmt.Fprintf(w, "%s+ %d created / ~ %d updated / - %d deleted\n",
		prefix, summary.Created, summary.Updated, summary.Deleted)
}

func postStructureReorder(c *client.Client, courseCode string, body map[string]any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}
	req, err := c.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/structure/reorder", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := doWithRetry(c, req)
	if err != nil {
		return nil, fmt.Errorf("reordering structure: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode == http.StatusConflict {
		return respBody, fmt.Errorf("reorder conflict (409): structure changed; re-fetch with `structure get` and retry")
	}
	if resp.StatusCode != http.StatusOK {
		return respBody, apiErrorBody(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func runStructureApply(cmd *cobra.Command, args []string) error {
	raw, err := readInputFile(structureApplyFlags.file)
	if err != nil {
		return fmt.Errorf("reading structure file: %w", err)
	}
	spec, err := parseStructureApplyFile(raw)
	if err != nil {
		return err
	}

	c := client.New(Cfg.Server, Cfg.APIKey)
	current, _, err := fetchCourseStructure(c, args[0])
	if err != nil {
		return err
	}

	summary := computeStructureDiff(current.Items, spec)
	printChangeSummary(cmd.OutOrStdout(), summary, structureApplyFlags.dryRun)

	if structureApplyFlags.dryRun {
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"dryRun":  true,
				"created": summary.Created,
				"updated": summary.Updated,
				"deleted": summary.Deleted,
			})
		}
		return nil
	}

	courseCode := args[0]
	currentMods := filterModules(current.Items)
	moduleIDByDesired := make([]string, 0, len(spec.Modules))

	for _, dm := range spec.Modules {
		var modID string
		ex := findModuleByIDOrTitle(currentMods, dm.ID, dm.Title)
		if ex == nil {
			created, err := createModule(c, courseCode, dm.Title)
			if err != nil {
				return err
			}
			modID = created.ID
			currentMods = append(currentMods, created)
		} else {
			modID = ex.ID
			needsUpdate := (dm.Title != "" && dm.Title != ex.Title) ||
				(dm.Published != nil && *dm.Published != ex.Published)
			if needsUpdate {
				if err := patchModule(c, courseCode, modID, modulePatchOpts{
					title:     dm.Title,
					published: dm.Published,
				}); err != nil {
					return err
				}
			}
		}
		moduleIDByDesired = append(moduleIDByDesired, modID)

		curChildren := filterChildren(current.Items, modID)
		for _, di := range dm.Items {
			ch := findChildByIDOrTitle(curChildren, di.ID, di.Title, di.Kind)
			if ch == nil {
				item, err := addItemToModule(c, courseCode, modID, di.Kind, di.Title, di.URL)
				if err != nil {
					return err
				}
				if di.Published != nil && *di.Published {
					if err := patchStructureItem(c, courseCode, item.ID, structureItemPatchOpts{published: di.Published}); err != nil {
						return err
					}
				}
				continue
			}
			opts := structureItemPatchOpts{}
			if di.Title != "" && di.Title != ch.Title {
				t := di.Title
				opts.title = &t
			}
			if di.Published != nil && *di.Published != ch.Published {
				opts.published = di.Published
			}
			if opts.title != nil || opts.published != nil {
				if err := patchStructureItem(c, courseCode, ch.ID, opts); err != nil {
					return err
				}
			}
		}
		for _, ch := range curChildren {
			keep := false
			for _, di := range dm.Items {
				if (di.ID != "" && di.ID == ch.ID) || (di.Title != "" && di.Title == ch.Title && (di.Kind == "" || di.Kind == ch.Kind)) {
					keep = true
					break
				}
			}
			if !keep {
				if err := deleteStructureItem(c, courseCode, ch.ID); err != nil {
					return err
				}
			}
		}
	}

	for _, cm := range currentMods {
		keep := false
		for _, dm := range spec.Modules {
			if (dm.ID != "" && dm.ID == cm.ID) || (dm.Title != "" && dm.Title == cm.Title) {
				keep = true
				break
			}
		}
		if !keep {
			if err := deleteModule(c, courseCode, cm.ID); err != nil {
				return err
			}
		}
	}

	if len(moduleIDByDesired) > 0 {
		updated, _, err := fetchCourseStructure(c, courseCode)
		if err != nil {
			return err
		}
		childOrder := make(map[string][]string)
		for _, dm := range spec.Modules {
			mod := findModuleByIDOrTitle(filterModules(updated.Items), dm.ID, dm.Title)
			if mod == nil {
				continue
			}
			var order []string
			for _, di := range dm.Items {
				ch := findChildByIDOrTitle(filterChildren(updated.Items, mod.ID), di.ID, di.Title, di.Kind)
				if ch != nil {
					order = append(order, ch.ID)
				}
			}
			if len(order) > 0 {
				childOrder[mod.ID] = order
			}
		}
		_, err = postStructureReorder(c, courseCode, map[string]any{
			"moduleOrder":        moduleIDByDesired,
			"childOrderByModule": childOrder,
		})
		if err != nil {
			return err
		}
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"applied": true,
			"created": summary.Created,
			"updated": summary.Updated,
			"deleted": summary.Deleted,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Applied structure to %s\n", courseCode)
	return nil
}