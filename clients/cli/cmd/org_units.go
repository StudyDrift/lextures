package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var orgUnitsCmd = &cobra.Command{
	Use:   "org-units",
	Short: "Manage organization unit hierarchy",
}

var orgUnitsListCmd = &cobra.Command{
	Use:   "list <org>",
	Short: "List org units",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgUnitsList,
}

var orgUnitsCreateFlags struct {
	name   string
	typ    string
	parent string
}

var orgUnitsCreateCmd = &cobra.Command{
	Use:   "create <org>",
	Short: "Create an org unit",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgUnitsCreate,
}

var orgUnitsUpdateFlags struct {
	name   string
	typ    string
	status string
	parent string
}

var orgUnitsUpdateCmd = &cobra.Command{
	Use:   "update <org> <unit>",
	Short: "Update an org unit",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgUnitsUpdate,
}

var orgUnitsDeleteCmd = &cobra.Command{
	Use:   "delete <org> <unit>",
	Short: "Delete an org unit",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgUnitsDelete,
}

var orgUnitsMoveFlags struct {
	unit string
}

var orgUnitsMoveCmd = &cobra.Command{
	Use:   "move <org> <parent>",
	Short: "Reparent an org unit under a new parent",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgUnitsMove,
}

func init() {
	orgUnitsCreateCmd.Flags().StringVar(&orgUnitsCreateFlags.name, "name", "", "unit name (required)")
	_ = orgUnitsCreateCmd.MarkFlagRequired("name")
	orgUnitsCreateCmd.Flags().StringVar(&orgUnitsCreateFlags.typ, "type", "other", "unit type")
	orgUnitsCreateCmd.Flags().StringVar(&orgUnitsCreateFlags.parent, "parent", "", "parent unit UUID")

	orgUnitsUpdateCmd.Flags().StringVar(&orgUnitsUpdateFlags.name, "name", "", "unit name")
	orgUnitsUpdateCmd.Flags().StringVar(&orgUnitsUpdateFlags.typ, "type", "", "unit type")
	orgUnitsUpdateCmd.Flags().StringVar(&orgUnitsUpdateFlags.status, "status", "", "unit status")
	orgUnitsUpdateCmd.Flags().StringVar(&orgUnitsUpdateFlags.parent, "parent", "", "new parent unit UUID")

	orgUnitsMoveCmd.Flags().StringVar(&orgUnitsMoveFlags.unit, "unit", "", "unit UUID to move (required)")
	_ = orgUnitsMoveCmd.MarkFlagRequired("unit")

	orgUnitsCmd.AddCommand(
		orgUnitsListCmd,
		orgUnitsCreateCmd,
		orgUnitsUpdateCmd,
		orgUnitsDeleteCmd,
		orgUnitsMoveCmd,
	)
	rootCmd.AddCommand(orgUnitsCmd)
}

func runOrgUnitsList(cmd *cobra.Command, args []string) error {
	body, raw, err := fetchOrgUnits(client.New(Cfg.Server, Cfg.APIKey), args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Units) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No org units.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tPARENT\tTYPE\tNAME\tSTATUS")
	for _, u := range body.Units {
		parent := ""
		if u.ParentUnitID != nil {
			parent = *u.ParentUnitID
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", u.ID, parent, u.UnitType, u.Name, u.Status)
	}
	return w.Flush()
}

func runOrgUnitsCreate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{
		"name":     orgUnitsCreateFlags.name,
		"unitType": orgUnitsCreateFlags.typ,
	}
	if orgUnitsCreateFlags.parent != "" {
		payload["parentUnitId"] = orgUnitsCreateFlags.parent
	}
	body, err := createOrgUnit(client.New(Cfg.Server, Cfg.APIKey), args[0], payload)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Created org unit")
}

func runOrgUnitsUpdate(cmd *cobra.Command, args []string) error {
	payload := map[string]any{}
	if orgUnitsUpdateFlags.name != "" {
		payload["name"] = orgUnitsUpdateFlags.name
	}
	if orgUnitsUpdateFlags.typ != "" {
		payload["unitType"] = orgUnitsUpdateFlags.typ
	}
	if orgUnitsUpdateFlags.status != "" {
		payload["status"] = orgUnitsUpdateFlags.status
	}
	if orgUnitsUpdateFlags.parent != "" {
		payload["parentUnitId"] = orgUnitsUpdateFlags.parent
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one update flag is required")
	}
	body, err := patchOrgUnit(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1], payload)
	if err != nil {
		return err
	}
	return emitRawOrMessage(cmd, body, "Updated org unit")
}

func runOrgUnitsDelete(cmd *cobra.Command, args []string) error {
	if err := deleteOrgUnit(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1]); err != nil {
		return err
	}
	return emitRawOrMessage(cmd, nil, "Deleted org unit")
}

func runOrgUnitsMove(cmd *cobra.Command, args []string) error {
	payload := map[string]any{"childUnitId": orgUnitsMoveFlags.unit}
	body, err := moveOrgUnitChild(client.New(Cfg.Server, Cfg.APIKey), args[0], args[1], payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Moved org unit")
	return nil
}