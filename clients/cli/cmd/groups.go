package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/lextures/lextures/clients/cli/internal/client"
	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage course groups and membership",
}

var groupsListCmd = &cobra.Command{
	Use:   "list <course>",
	Short: "List groups in a course",
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupsList,
}

var groupsCreateFlags struct {
	set   string
	name  string
	auto  bool
	size  int
	seed  int64
	count int
}

var groupsCreateCmd = &cobra.Command{
	Use:   "create <course>",
	Short: "Create a group set and groups (optionally auto-assign students)",
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupsCreate,
}

var groupsDeleteFlags struct {
	set   string
	group string
}

var groupsDeleteCmd = &cobra.Command{
	Use:   "delete <course>",
	Short: "Delete a group or entire group set",
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupsDelete,
}

var groupsAddFlags struct {
	group string
	user  string
}

var groupsAddCmd = &cobra.Command{
	Use:   "add <course>",
	Short: "Add a user to a group",
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupsAdd,
}

var groupsRemoveFlags struct {
	group string
	user  string
}

var groupsRemoveCmd = &cobra.Command{
	Use:   "remove <course>",
	Short: "Remove a user from a group",
	Args:  cobra.ExactArgs(1),
	RunE:  runGroupsRemove,
}

var groupsMembersCmd = &cobra.Command{
	Use:   "members <course> <group>",
	Short: "List members of a group",
	Args:  cobra.ExactArgs(2),
	RunE:  runGroupsMembers,
}

func init() {
	groupsCreateCmd.Flags().StringVar(&groupsCreateFlags.set, "set", "", "group set name (required unless --auto creates one)")
	groupsCreateCmd.Flags().StringVar(&groupsCreateFlags.name, "name", "", "single group name (manual create without --auto)")
	groupsCreateCmd.Flags().BoolVar(&groupsCreateFlags.auto, "auto", false, "auto-assign enrolled students into groups")
	groupsCreateCmd.Flags().IntVar(&groupsCreateFlags.size, "size", 4, "target group size when using --auto")
	groupsCreateCmd.Flags().Int64Var(&groupsCreateFlags.seed, "seed", 0, "deterministic shuffle seed for --auto")
	groupsCreateCmd.Flags().IntVar(&groupsCreateFlags.count, "count", 0, "number of groups for --auto (default: ceil(students/size))")

	groupsDeleteCmd.Flags().StringVar(&groupsDeleteFlags.set, "set", "", "group set UUID (required when deleting a set)")
	groupsDeleteCmd.Flags().StringVar(&groupsDeleteFlags.group, "group", "", "group UUID to delete")

	groupsAddCmd.Flags().StringVar(&groupsAddFlags.group, "group", "", "group UUID (required)")
	_ = groupsAddCmd.MarkFlagRequired("group")
	groupsAddCmd.Flags().StringVar(&groupsAddFlags.user, "user", "", "user UUID or email (required)")
	_ = groupsAddCmd.MarkFlagRequired("user")

	groupsRemoveCmd.Flags().StringVar(&groupsRemoveFlags.group, "group", "", "group UUID (required)")
	_ = groupsRemoveCmd.MarkFlagRequired("group")
	groupsRemoveCmd.Flags().StringVar(&groupsRemoveFlags.user, "user", "", "user UUID or email (required)")
	_ = groupsRemoveCmd.MarkFlagRequired("user")

	groupsCmd.AddCommand(
		groupsListCmd,
		groupsCreateCmd,
		groupsDeleteCmd,
		groupsAddCmd,
		groupsRemoveCmd,
		groupsMembersCmd,
	)
	rootCmd.AddCommand(groupsCmd)
}

func runGroupsList(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	body, raw, err := fetchCourseGroups(c, args[0])
	if err != nil {
		return err
	}
	if globalFlags.jsonOut {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	if len(body.Groups) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No groups.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSET\tNAME\tMEMBERS")
	for _, g := range body.Groups {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", g.ID, g.GroupSetID, g.Name, g.MemberCount)
	}
	return w.Flush()
}

func runGroupsCreate(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]

	setName := strings.TrimSpace(groupsCreateFlags.set)
	if setName == "" && !groupsCreateFlags.auto {
		return fmt.Errorf("--set is required unless --auto is used")
	}
	if setName == "" {
		setName = "Project Teams"
	}

	setID, setBody, err := postEnrollmentGroupSet(c, course, setName)
	if err != nil {
		return err
	}

	if !groupsCreateFlags.auto {
		groupName := strings.TrimSpace(groupsCreateFlags.name)
		if groupName == "" {
			groupName = "Group 1"
		}
		groupID, groupBody, err := postEnrollmentGroup(c, course, setID, groupName)
		if err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"groupSetId": setID,
				"groupId":    groupID,
				"set":        json.RawMessage(setBody),
				"group":      json.RawMessage(groupBody),
			})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created group set %q (%s) with group %q (%s)\n",
			setName, setID, groupName, groupID)
		return nil
	}

	rows, err := fetchEnrollments(c, course)
	if err != nil {
		return err
	}
	students := studentEnrollmentIDs(rows)
	if len(students) == 0 {
		return fmt.Errorf("no student enrollments found in course %s", course)
	}

	groupCount, ordered := planAutoAssignGroups(students, groupsCreateFlags.size, groupsCreateFlags.seed)
	if groupsCreateFlags.count > 0 {
		groupCount = groupsCreateFlags.count
	}
	if groupCount < 1 {
		groupCount = 1
	}

	groupIDs := make([]string, 0, groupCount)
	for i := 0; i < groupCount; i++ {
		name := fmt.Sprintf("Team %d", i+1)
		id, _, err := postEnrollmentGroup(c, course, setID, name)
		if err != nil {
			return err
		}
		groupIDs = append(groupIDs, id)
	}
	if err := assignStudentsRoundRobin(c, course, setID, groupIDs, ordered); err != nil {
		return err
	}

	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"groupSetId": setID,
			"groupIds":   groupIDs,
			"assigned":   len(ordered),
			"size":       groupsCreateFlags.size,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created group set %q with %d groups and assigned %d students\n",
		setName, len(groupIDs), len(ordered))
	return nil
}

func runGroupsDelete(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	if strings.TrimSpace(groupsDeleteFlags.group) != "" {
		if err := deleteEnrollmentGroup(c, course, groupsDeleteFlags.group); err != nil {
			return err
		}
		if globalFlags.jsonOut {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deletedGroup": groupsDeleteFlags.group})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted group %s\n", groupsDeleteFlags.group)
		return nil
	}
	if strings.TrimSpace(groupsDeleteFlags.set) == "" {
		return fmt.Errorf("either --group or --set is required")
	}
	if err := deleteEnrollmentGroupSet(c, course, groupsDeleteFlags.set); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"deletedSet": groupsDeleteFlags.set})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted group set %s\n", groupsDeleteFlags.set)
	return nil
}

func runGroupsAdd(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]

	tree, _, err := fetchEnrollmentGroupsTree(c, course)
	if err != nil {
		return err
	}
	_, set := findGroupInTree(tree, groupsAddFlags.group)
	if set == nil {
		return fmt.Errorf("group %q not found", groupsAddFlags.group)
	}

	userID, _, err := resolveUserID(c, groupsAddFlags.user)
	if err != nil {
		return err
	}
	rows, err := fetchEnrollments(c, course)
	if err != nil {
		return err
	}
	enrollmentID, err := enrollmentIDForUser(rows, userID)
	if err != nil {
		return err
	}
	if err := putEnrollmentGroupMembership(c, course, enrollmentID, set.ID, groupsAddFlags.group); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"user":         userID,
			"group":        groupsAddFlags.group,
			"enrollmentId": enrollmentID,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %s to group %s\n", groupsAddFlags.user, groupsAddFlags.group)
	return nil
}

func runGroupsRemove(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]

	tree, _, err := fetchEnrollmentGroupsTree(c, course)
	if err != nil {
		return err
	}
	_, set := findGroupInTree(tree, groupsRemoveFlags.group)
	if set == nil {
		return fmt.Errorf("group %q not found", groupsRemoveFlags.group)
	}

	userID, _, err := resolveUserID(c, groupsRemoveFlags.user)
	if err != nil {
		return err
	}
	rows, err := fetchEnrollments(c, course)
	if err != nil {
		return err
	}
	enrollmentID, err := enrollmentIDForUser(rows, userID)
	if err != nil {
		return err
	}
	if err := removeEnrollmentGroupMembership(c, course, enrollmentID, set.ID); err != nil {
		return err
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
			"user":         userID,
			"group":        groupsRemoveFlags.group,
			"enrollmentId": enrollmentID,
		})
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %s from group %s\n", groupsRemoveFlags.user, groupsRemoveFlags.group)
	return nil
}

func runGroupsMembers(cmd *cobra.Command, args []string) error {
	c := client.New(Cfg.Server, Cfg.APIKey)
	course := args[0]
	groupID := args[1]

	tree, raw, err := fetchEnrollmentGroupsTree(c, course)
	if err != nil {
		return err
	}
	group, set := findGroupInTree(tree, groupID)
	if group == nil {
		return fmt.Errorf("group %q not found", groupID)
	}
	if globalFlags.jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"groupSetId":     set.ID,
			"groupId":        group.ID,
			"enrollmentIds":  group.EnrollmentIDs,
			"tree":           json.RawMessage(raw),
		})
	}
	if len(group.EnrollmentIDs) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No members.")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ENROLLMENT_ID")
	for _, id := range group.EnrollmentIDs {
		_, _ = fmt.Fprintf(w, "%s\n", id)
	}
	return w.Flush()
}