package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var customFieldsCmd = &cobra.Command{
	Use:   "custom-fields",
	Short: "Manage custom profile field definitions",
}

var customFieldsListFlags struct {
	entity string
	org    string
}

var customFieldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List custom field definitions",
	RunE:  runCustomFieldsList,
}

var customFieldsCreateFlags struct {
	entity   string
	key      string
	label    string
	typ      string
	required bool
	org      string
}

var customFieldsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a custom field definition",
	RunE:  runCustomFieldsCreate,
}

var customFieldsUpdateFlags struct {
	label    string
	required bool
	org      string
}

var customFieldsUpdateCmd = &cobra.Command{
	Use:   "update <field>",
	Short: "Update a custom field definition",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomFieldsUpdate,
}

var customFieldsDeleteFlags struct {
	org string
}

var customFieldsDeleteCmd = &cobra.Command{
	Use:   "delete <field>",
	Short: "Delete a custom field definition",
	Args:  cobra.ExactArgs(1),
	RunE:  runCustomFieldsDelete,
}

func init() {
	customFieldsListCmd.Flags().StringVar(&customFieldsListFlags.entity, "entity", "user", "entity type: user, course, enrollment")
	customFieldsListCmd.Flags().StringVar(&customFieldsListFlags.org, "org", "", "target org UUID (global admin)")

	customFieldsCreateCmd.Flags().StringVar(&customFieldsCreateFlags.entity, "entity", "user", "entity type")
	customFieldsCreateCmd.Flags().StringVar(&customFieldsCreateFlags.key, "key", "", "field key (required)")
	_ = customFieldsCreateCmd.MarkFlagRequired("key")
	customFieldsCreateCmd.Flags().StringVar(&customFieldsCreateFlags.label, "label", "", "field label (required)")
	_ = customFieldsCreateCmd.MarkFlagRequired("label")
	customFieldsCreateCmd.Flags().StringVar(&customFieldsCreateFlags.typ, "type", "text", "field type")
	customFieldsCreateCmd.Flags().BoolVar(&customFieldsCreateFlags.required, "required", false, "mark field required")
	customFieldsCreateCmd.Flags().StringVar(&customFieldsCreateFlags.org, "org", "", "target org UUID (global admin)")

	customFieldsUpdateCmd.Flags().StringVar(&customFieldsUpdateFlags.label, "label", "", "field label")
	customFieldsUpdateCmd.Flags().BoolVar(&customFieldsUpdateFlags.required, "required", false, "mark field required")
	customFieldsUpdateCmd.Flags().StringVar(&customFieldsUpdateFlags.org, "org", "", "target org UUID (global admin)")

	customFieldsDeleteCmd.Flags().StringVar(&customFieldsDeleteFlags.org, "org", "", "target org UUID (global admin)")

	customFieldsCmd.AddCommand(customFieldsListCmd, customFieldsCreateCmd, customFieldsUpdateCmd, customFieldsDeleteCmd)
	rootCmd.AddCommand(customFieldsCmd)
}

func customFieldsPath(orgID, suffix string) string {
	path := "/api/v1/admin-console/custom-fields" + suffix
	if orgID != "" {
		path += "?orgId=" + url.QueryEscape(orgID)
	}
	return path
}

func runCustomFieldsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	params := url.Values{}
	if customFieldsListFlags.org != "" {
		params.Set("orgId", customFieldsListFlags.org)
	}
	if customFieldsListFlags.entity != "" {
		params.Set("entity_type", customFieldsListFlags.entity)
	}
	path := "/api/v1/admin-console/custom-fields"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var items []map[string]any
	if err := json.Unmarshal(body, &items); err != nil {
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tENTITY\tKEY\tLABEL\tTYPE")
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n", item["id"], item["entityType"], item["key"], item["label"], item["fieldType"])
	}
	return w.Flush()
}

func runCustomFieldsCreate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{
		"entityType": customFieldsCreateFlags.entity,
		"key":        customFieldsCreateFlags.key,
		"label":      customFieldsCreateFlags.label,
		"fieldType":  customFieldsCreateFlags.typ,
		"isRequired": customFieldsCreateFlags.required,
	}
	raw, _ := json.Marshal(payload)
	c := client.New(Cfg.Server, Cfg.APIKey)
	req, err := c.NewRequest(http.MethodPost, customFieldsPath(customFieldsCreateFlags.org, ""), bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return emitRawOrMessage(cmd, body, "Created custom field")
}

func runCustomFieldsUpdate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{}
	if customFieldsUpdateFlags.label != "" {
		payload["label"] = customFieldsUpdateFlags.label
	}
	payload["isRequired"] = customFieldsUpdateFlags.required
	raw, _ := json.Marshal(payload)
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := customFieldsPath(customFieldsUpdateFlags.org, "/"+url.PathEscape(args[0]))
	req, err := c.NewRequest(http.MethodPut, path, bytes.NewReader(raw))
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
	if resp.StatusCode != http.StatusOK {
		return apiErrorBody(resp.StatusCode, body)
	}
	return emitRawOrMessage(cmd, body, "Updated custom field")
}

func runCustomFieldsDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	path := customFieldsPath(customFieldsDeleteFlags.org, "/"+url.PathEscape(args[0]))
	req, err := c.NewRequest(http.MethodDelete, path, nil)
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiErrorBody(resp.StatusCode, body)
	}
	return emitRawOrMessage(cmd, nil, "Deleted custom field")
}