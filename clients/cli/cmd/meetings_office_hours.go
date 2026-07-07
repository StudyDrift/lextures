package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/cli"
	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var meetingsCmd = &cobra.Command{Use: "meetings", Short: "Virtual meetings for courses"}

var meetingsListFlags struct{ course string }

var meetingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List meetings for a course",
	RunE:  runMeetingsList,
}

var meetingsCreateFlags struct {
	course   string
	title    string
	start    string
	duration string
	tz       string
	provider string
	file     string
}

var meetingsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a meeting",
	RunE:  runMeetingsCreate,
}

var meetingsUpdateFlags struct {
	file string
}

var meetingsUpdateCmd = &cobra.Command{
	Use:   "update <meeting_id>",
	Short: "Update a meeting",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeetingsUpdate,
}

var meetingsCancelCmd = &cobra.Command{
	Use:   "cancel <meeting_id>",
	Short: "Cancel a meeting",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeetingsCancel,
}

var officeHoursCmd = &cobra.Command{Use: "office-hours", Short: "Office hours availability and slots"}

var officeHoursSetFlags struct {
	course string
	file   string
}

var officeHoursSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Publish availability windows from a JSON file",
	RunE:  runOfficeHoursSet,
}

var officeHoursListFlags struct{ course string }

var officeHoursListCmd = &cobra.Command{
	Use:   "list",
	Short: "List availability windows and slots",
	RunE:  runOfficeHoursList,
}

var officeHoursSlotsFlags struct{ course string }

var officeHoursSlotsCmd = &cobra.Command{
	Use:   "slots",
	Short: "List appointment slots",
	RunE:  runOfficeHoursSlots,
}

var availabilityCmd = &cobra.Command{Use: "availability", Short: "Course availability (alias for office-hours)"}

var availabilityGetFlags struct{ course string }
var availabilityGetCmd = &cobra.Command{Use: "get", Short: "Get availability", RunE: runAvailabilityGet}

var availabilitySetFlags struct {
	course string
	file   string
}
var availabilitySetCmd = &cobra.Command{Use: "set", Short: "Set availability from file", RunE: runAvailabilitySet}

var conferencesCmd = &cobra.Command{Use: "conferences", Short: "Parent-teacher conference slots"}

var conferencesListFlags struct{ teacher string }
var conferencesListCmd = &cobra.Command{Use: "list", Short: "List conference slots", RunE: runConferencesList}

var conferencesCreateFlags struct {
	teacher string
	file    string
}
var conferencesCreateCmd = &cobra.Command{Use: "create", Short: "Create conference availability", RunE: runConferencesCreate}

var conferencesSlotsFlags struct{ teacher string }
var conferencesSlotsCmd = &cobra.Command{Use: "slots", Short: "List conference slots", RunE: runConferencesSlots}

var calendarCmd = &cobra.Command{Use: "calendar", Short: "Personal calendar feed"}

var calendarTokenGetCmd = &cobra.Command{Use: "token get", Short: "Get calendar token status", RunE: runCalendarTokenGet}
var calendarTokenRotateCmd = &cobra.Command{Use: "token rotate", Short: "Rotate calendar token (one-time URL)", RunE: runCalendarTokenRotate}

var calendarExportFlags struct {
	out   string
	token string
}
var calendarExportCmd = &cobra.Command{Use: "export", Short: "Download iCal feed", RunE: runCalendarExport}

var calendarTokenCmd = &cobra.Command{Use: "token", Short: "Calendar feed token"}

func runMeetingsList(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	rows, raw, err := listMeetings(c, meetingsListFlags.course)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tSTART")
	for _, m := range rows {
		start := ""
		if m.ScheduledStart != nil {
			start = *m.ScheduledStart
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.ID, m.Title, m.Status, start)
	}
	return w.Flush()
}

func runMeetingsCreate(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	var payload map[string]any
	var err error
	if meetingsCreateFlags.file != "" {
		payload, err = cli.ReadJSONFile(meetingsCreateFlags.file)
	} else {
		tz := meetingsCreateFlags.tz
		if tz == "" {
			tz = cliRuntime.tz
		}
		payload, err = buildMeetingPayload(meetingsCreateFlags.title, meetingsCreateFlags.start, meetingsCreateFlags.duration, tz, meetingsCreateFlags.provider)
	}
	if err != nil {
		return err
	}
	raw, err := createMeeting(c, meetingsCreateFlags.course, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	var m meetingRow
	_ = json.Unmarshal(raw, &m)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created meeting %s\n", m.ID)
	return nil
}

func runMeetingsUpdate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(meetingsUpdateFlags.file)
	if err != nil {
		return err
	}
	raw, err := patchMeeting(c, args[0], payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
	}
	return err
}

func runMeetingsCancel(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	status := "cancelled"
	raw, err := patchMeeting(c, args[0], map[string]any{"status": status})
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Meeting cancelled.")
	return nil
}

func runOfficeHoursSet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(officeHoursSetFlags.file)
	if err != nil {
		return err
	}
	raw, err := setOfficeHoursAvailability(c, officeHoursSetFlags.course, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Availability published.")
	return nil
}

func runOfficeHoursList(cmd *cobra.Command, _ []string) error {
	return runAvailabilityGet(cmd, nil)
}

func runOfficeHoursSlots(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	slots, raw, err := getOfficeHoursAvailability(c, officeHoursSlotsFlags.course)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTART\tEND\tSTATUS")
	for _, s := range slots {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.StartTime, s.EndTime, s.Status)
	}
	return w.Flush()
}

func runAvailabilityGet(cmd *cobra.Command, _ []string) error {
	course := availabilityGetFlags.course
	if course == "" {
		course = officeHoursListFlags.course
	}
	c := client.New(Cfg.Server, Cfg.APIKey)
	_, raw, err := getOfficeHoursAvailability(c, course)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runAvailabilitySet(cmd *cobra.Command, _ []string) error {
	officeHoursSetFlags.course = availabilitySetFlags.course
	officeHoursSetFlags.file = availabilitySetFlags.file
	return runOfficeHoursSet(cmd, nil)
}

func runConferencesList(cmd *cobra.Command, _ []string) error {
	conferencesSlotsFlags.teacher = conferencesListFlags.teacher
	return runConferencesSlots(cmd, nil)
}

func runConferencesCreate(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	payload, err := cli.ReadJSONFile(conferencesCreateFlags.file)
	if err != nil {
		return err
	}
	raw, err := createConferenceAvailability(c, conferencesCreateFlags.teacher, payload)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Conference availability created.")
	return nil
}

func runConferencesSlots(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	slots, raw, err := listConferenceSlots(c, conferencesSlotsFlags.teacher)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTART\tEND\tSTATUS")
	for _, s := range slots {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.StartTime, s.EndTime, s.Status)
	}
	return w.Flush()
}

func runCalendarTokenGet(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := getCalendarToken(c)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(raw)
	return err
}

func runCalendarTokenRotate(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := rotateCalendarToken(c)
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		cli.RedactSecrets(out)
		if u, ok := out["feedUrl"].(string); ok && strings.Contains(u, "token=") {
			out["feedUrl"] = "[redacted — use rotate output in human mode]"
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	var out struct {
		FeedURL string `json:"feedUrl"`
		Token   string `json:"token"`
	}
	_ = json.Unmarshal(raw, &out)
	_, _ = fmt.Fprintln(os.Stderr, "Save this URL now — the token will not be shown again on re-fetch.")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.FeedURL)
	return nil
}

func runCalendarExport(cmd *cobra.Command, _ []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	raw, err := exportCalendarICal(c, calendarExportFlags.token)
	if err != nil {
		return err
	}
	if calendarExportFlags.out == "" || calendarExportFlags.out == "-" {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	return os.WriteFile(calendarExportFlags.out, raw, 0o644)
}

func init() {
	meetingsListCmd.Flags().StringVar(&meetingsListFlags.course, "course", "", "course code")
	_ = meetingsListCmd.MarkFlagRequired("course")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.course, "course", "", "course code")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.title, "title", "", "meeting title")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.start, "start", "", "start time (RFC3339)")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.duration, "duration", "", "duration in minutes")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.tz, "tz", "", "timezone for --start")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.provider, "provider", "jitsi", "video provider")
	meetingsCreateCmd.Flags().StringVar(&meetingsCreateFlags.file, "file", "", "meeting JSON file")
	_ = meetingsCreateCmd.MarkFlagRequired("course")
	meetingsUpdateCmd.Flags().StringVar(&meetingsUpdateFlags.file, "file", "", "patch JSON")
	_ = meetingsUpdateCmd.MarkFlagRequired("file")

	officeHoursSetCmd.Flags().StringVar(&officeHoursSetFlags.course, "course", "", "course code")
	officeHoursSetCmd.Flags().StringVar(&officeHoursSetFlags.file, "file", "", "availability JSON")
	_ = officeHoursSetCmd.MarkFlagRequired("course")
	_ = officeHoursSetCmd.MarkFlagRequired("file")
	officeHoursListCmd.Flags().StringVar(&officeHoursListFlags.course, "course", "", "course code")
	_ = officeHoursListCmd.MarkFlagRequired("course")
	officeHoursSlotsCmd.Flags().StringVar(&officeHoursSlotsFlags.course, "course", "", "course code")
	_ = officeHoursSlotsCmd.MarkFlagRequired("course")

	availabilityGetCmd.Flags().StringVar(&availabilityGetFlags.course, "course", "", "course code")
	_ = availabilityGetCmd.MarkFlagRequired("course")
	availabilitySetCmd.Flags().StringVar(&availabilitySetFlags.course, "course", "", "course code")
	availabilitySetCmd.Flags().StringVar(&availabilitySetFlags.file, "file", "", "availability JSON")
	_ = availabilitySetCmd.MarkFlagRequired("course")
	_ = availabilitySetCmd.MarkFlagRequired("file")
	availabilityCmd.AddCommand(availabilityGetCmd, availabilitySetCmd)

	conferencesListCmd.Flags().StringVar(&conferencesListFlags.teacher, "teacher", "", "teacher user id")
	_ = conferencesListCmd.MarkFlagRequired("teacher")
	conferencesCreateCmd.Flags().StringVar(&conferencesCreateFlags.teacher, "teacher", "", "teacher user id")
	conferencesCreateCmd.Flags().StringVar(&conferencesCreateFlags.file, "file", "", "availability JSON")
	_ = conferencesCreateCmd.MarkFlagRequired("teacher")
	_ = conferencesCreateCmd.MarkFlagRequired("file")
	conferencesSlotsCmd.Flags().StringVar(&conferencesSlotsFlags.teacher, "teacher", "", "teacher user id")
	_ = conferencesSlotsCmd.MarkFlagRequired("teacher")

	calendarExportCmd.Flags().StringVar(&calendarExportFlags.out, "out", "-", "output file (- for stdout)")
	calendarExportCmd.Flags().StringVar(&calendarExportFlags.token, "token", "", "calendar feed token (optional)")
	calendarTokenCmd.AddCommand(calendarTokenGetCmd, calendarTokenRotateCmd)

	meetingsCmd.AddCommand(meetingsListCmd, meetingsCreateCmd, meetingsUpdateCmd, meetingsCancelCmd)
	officeHoursCmd.AddCommand(officeHoursSetCmd, officeHoursListCmd, officeHoursSlotsCmd)
	conferencesCmd.AddCommand(conferencesListCmd, conferencesCreateCmd, conferencesSlotsCmd)
	calendarCmd.AddCommand(calendarTokenCmd, calendarExportCmd)

	rootCmd.AddCommand(meetingsCmd, officeHoursCmd, availabilityCmd, conferencesCmd, calendarCmd)
}