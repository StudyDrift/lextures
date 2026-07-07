package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var portfoliosCmd = &cobra.Command{Use: "portfolios", Short: "E-portfolios and artifacts"}

var portfoliosListCmd = &cobra.Command{Use: "list", Short: "List portfolios", RunE: runPortfoliosList}

var portfoliosGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a portfolio with artifacts",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortfoliosGet,
}

var portfoliosCreateFlags struct {
	title string
	file  string
}
var portfoliosCreateCmd = &cobra.Command{Use: "create", Short: "Create a portfolio", RunE: runPortfoliosCreate}

var portfoliosDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a portfolio",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortfoliosDelete,
}

var portfoliosAddArtifactFlags struct {
	file       string
	link       string
	reflection string
	title      string
}
var portfoliosAddArtifactCmd = &cobra.Command{
	Use:   "add-artifact <portfolio_id>",
	Short: "Add an artifact (file, link, or reflection)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortfoliosAddArtifact,
}

var portfoliosRemoveArtifactCmd = &cobra.Command{
	Use:   "remove-artifact <portfolio_id> <artifact_id>",
	Short: "Remove an artifact",
	Args:  cobra.ExactArgs(2),
	RunE:  runPortfoliosRemoveArtifact,
}

var portfoliosPublishFlags struct{ yes bool }
var portfoliosPublishCmd = &cobra.Command{
	Use:   "publish <id>",
	Short: "Publish a portfolio (public link)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortfoliosPublish,
}

var portfoliosUnpublishCmd = &cobra.Command{
	Use:   "unpublish <id>",
	Short: "Unpublish a portfolio",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortfoliosUnpublish,
}

var portfoliosExportFlags struct{ out string }
var portfoliosExportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Export portfolio metadata and artifact list",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortfoliosExport,
}

func runPortfoliosList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listPortfolios(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tPUBLIC")
	for _, p := range rows {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%v\n", p.ID, p.Title, p.IsPublic)
	}
	return w.Flush()
}

func runPortfoliosGet(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getPortfolio(c, args[0])
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runPortfoliosCreate(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	var payload map[string]any
	var err error
	if portfoliosCreateFlags.file != "" {
		payload, err = cli.ReadJSONFile(portfoliosCreateFlags.file)
	} else {
		payload = map[string]any{"title": portfoliosCreateFlags.title}
	}
	if err != nil {
		return err
	}
	raw, err := createPortfolio(c, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runPortfoliosDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := deletePortfolio(c, args[0]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Portfolio deleted.")
	return nil
}

func runPortfoliosAddArtifact(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	pid := args[0]
	if portfoliosAddArtifactFlags.file != "" {
		raw, err := uploadPortfolioArtifact(c, pid, portfoliosAddArtifactFlags.file, portfoliosAddArtifactFlags.title)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	payload := map[string]any{"title": portfoliosAddArtifactFlags.title}
	if portfoliosAddArtifactFlags.link != "" {
		payload["artifactType"] = "link"
		payload["externalUrl"] = portfoliosAddArtifactFlags.link
	} else if portfoliosAddArtifactFlags.reflection != "" {
		payload["artifactType"] = "reflection"
		text, err := cli.ReadTextFile(portfoliosAddArtifactFlags.reflection)
		if err != nil {
			return err
		}
		payload["textContent"] = text
	} else {
		return fmt.Errorf("provide --file, --link, or --reflection")
	}
	if payload["title"] == "" {
		payload["title"] = "Artifact"
	}
	raw, err := createPortfolioArtifact(c, pid, payload)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runPortfoliosRemoveArtifact(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	if err := deletePortfolioArtifact(c, args[0], args[1]); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]bool{"ok": true})
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Artifact removed.")
	return nil
}

func runPortfoliosPublish(cmd *cobra.Command, args []string) error {
	if err := cli.RequireYes(portfoliosPublishFlags.yes, "publishing makes portfolio content publicly accessible"); err != nil {
		return err
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	pub := true
	raw, err := patchPortfolio(c, args[0], map[string]any{"isPublic": pub})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	var detail struct {
		Portfolio portfolioRow `json:"portfolio"`
	}
	_ = json.Unmarshal(raw, &detail)
	slug := ""
	if detail.Portfolio.Slug != nil {
		slug = *detail.Portfolio.Slug
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Published. Public slug: %s\n", slug)
	return nil
}

func runPortfoliosUnpublish(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	pub := false
	raw, err := patchPortfolio(c, args[0], map[string]any{"isPublic": pub})
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runPortfoliosExport(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := exportPortfolio(c, args[0])
	if err != nil {
		return err
	}
	out := portfoliosExportFlags.out
	if out == "" || out == "-" {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	return os.WriteFile(out, raw, 0o644)
}

func init() {
	portfoliosCreateCmd.Flags().StringVar(&portfoliosCreateFlags.title, "title", "", "portfolio title")
	portfoliosCreateCmd.Flags().StringVar(&portfoliosCreateFlags.file, "file", "", "portfolio JSON")
	portfoliosAddArtifactCmd.Flags().StringVar(&portfoliosAddArtifactFlags.file, "file", "", "artifact file to upload")
	portfoliosAddArtifactCmd.Flags().StringVar(&portfoliosAddArtifactFlags.link, "link", "", "external link URL")
	portfoliosAddArtifactCmd.Flags().StringVar(&portfoliosAddArtifactFlags.reflection, "reflection", "", "reflection text file")
	portfoliosAddArtifactCmd.Flags().StringVar(&portfoliosAddArtifactFlags.title, "title", "", "artifact title")
	portfoliosPublishCmd.Flags().BoolVar(&portfoliosPublishFlags.yes, "yes", false, "confirm public publish")
	portfoliosExportCmd.Flags().StringVar(&portfoliosExportFlags.out, "out", "-", "output file")

	portfoliosCmd.AddCommand(
		portfoliosListCmd, portfoliosGetCmd, portfoliosCreateCmd, portfoliosDeleteCmd,
		portfoliosAddArtifactCmd, portfoliosRemoveArtifactCmd,
		portfoliosPublishCmd, portfoliosUnpublishCmd, portfoliosExportCmd,
	)
	rootCmd.AddCommand(portfoliosCmd)
}