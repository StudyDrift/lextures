package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/lextures/lextures/clients/cli/internal/config"
	"github.com/spf13/cobra"
)

var cliRuntime struct {
	output    string
	quiet     bool
	noColor   bool
	noHeaders bool
	tz        string
	all       bool
	wait      bool
	timeout   int
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Args:  cobra.ExactArgs(1),
	RunE:  runCompletion,
	Annotations: map[string]string{SkipAuthAnnotation: "true"},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and manage CLI configuration",
	Annotations: map[string]string{SkipAuthAnnotation: "true"},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get config values (server, profile, api_key redacted)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resolved configuration",
	RunE:  runConfigList,
}

var configEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Show environment diagnostics for CLI troubleshooting",
	RunE:  runConfigEnv,
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current authenticated user (alias for me get)",
	RunE:  runWhoami,
}

func outputOpts(cmd *cobra.Command) cli.Options {
	return cli.Options{
		Format:    cli.ParseFormat(cliRuntime.output, globalFlags.jsonOut),
		Quiet:     cliRuntime.quiet,
		NoHeaders: cliRuntime.noHeaders,
		NoColor:   cliRuntime.noColor || !isTerminalWriter(cmd.OutOrStdout()),
		Stdout:    cmd.OutOrStdout(),
		Stderr:    cmd.ErrOrStderr(),
	}
}

func isTerminalWriter(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		st, err := f.Stat()
		return err == nil && (st.Mode()&os.ModeCharDevice) != 0
	}
	return false
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	case "zsh":
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	default:
		return fmt.Errorf("unsupported shell %q (use bash, zsh, or fish)", args[0])
	}
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		return fmt.Errorf("config not loaded")
	}
	key := ""
	if len(args) > 0 {
		key = strings.ToLower(args[0])
	}
	redacted := map[string]any{
		"server":  Cfg.Server,
		"profile": globalFlags.profile,
		"json":    Cfg.JSON,
	}
	if Cfg.APIKey != "" {
		redacted["api_key"] = "[set]"
	}
	if key == "" {
		return outputOpts(cmd).WriteJSON(nil, redacted)
	}
	val, ok := redacted[key]
	if !ok {
		return fmt.Errorf("unknown config key %q", key)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{key: val})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%v\n", key, val)
	return nil
}

func runConfigList(cmd *cobra.Command, _ []string) error {
	return runConfigGet(cmd, nil)
}

func runConfigEnv(cmd *cobra.Command, _ []string) error {
	env := map[string]string{}
	for _, k := range []string{"LEXTURES_SERVER", "LEXTURES_API_KEY", "LEXTURES_JSON", "LEXTURES_DEBUG", "HOME"} {
		if v := os.Getenv(k); v != "" {
			if strings.Contains(strings.ToLower(k), "key") || strings.Contains(strings.ToLower(k), "secret") {
				env[k] = "[set]"
			} else {
				env[k] = v
			}
		}
	}
	cfgPath := globalFlags.configFile
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = home + "/.lextures.yaml"
	}
	out := map[string]any{
		"config_file": cfgPath,
		"env":         env,
		"version":     Version,
	}
	return outputOpts(cmd).WriteJSON(nil, out)
}

func runWhoami(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, err := fetchMeProfile(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}
	var me meProfile
	_ = json.Unmarshal(body, &me)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s <%s>\n", displayName(me), me.Email)
	if me.Org != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "org: %s\n", me.Org.Name)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cliRuntime.output, "output", "",
		"output format: table, json, ndjson, csv (--json is an alias for --output json)")
	rootCmd.PersistentFlags().BoolVar(&cliRuntime.quiet, "quiet", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&cliRuntime.noColor, "no-color", false, "disable color output")
	rootCmd.PersistentFlags().BoolVar(&cliRuntime.noHeaders, "no-headers", false, "omit table/CSV headers")
	rootCmd.PersistentFlags().StringVar(&cliRuntime.tz, "tz", "",
		"timezone for date parsing (IANA name, defaults to local)")
	rootCmd.PersistentFlags().BoolVar(&cliRuntime.all, "all", false, "fetch all pages when listing")
	rootCmd.PersistentFlags().BoolVar(&cliRuntime.wait, "wait", false, "wait for async jobs to complete")
	rootCmd.PersistentFlags().IntVar(&cliRuntime.timeout, "timeout", 300, "timeout in seconds for --wait")

	configCmd.AddCommand(configGetCmd, configListCmd, configEnvCmd)
	rootCmd.AddCommand(completionCmd, configCmd, whoamiCmd)
}

// ensureConfigLoaded is a noop guard used by config commands after PersistentPreRunE.
func ensureConfigLoaded() error {
	if Cfg == nil {
		cfg, err := config.Load(config.LoadOptions{
			ConfigFile: globalFlags.configFile,
			Profile:    globalFlags.profile,
			Server:     globalFlags.server,
			APIKey:     globalFlags.apiKey,
			JSON:       globalFlags.jsonOut,
		})
		if err != nil {
			return err
		}
		Cfg = cfg
	}
	return nil
}