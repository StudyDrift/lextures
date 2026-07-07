package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestPlanAutoAssignGroups_SeedDeterministic(t *testing.T) {
	students := make([]string, 20)
	for i := range students {
		students[i] = fmtStudentID(i)
	}
	count1, order1 := planAutoAssignGroups(students, 4, 42)
	count2, order2 := planAutoAssignGroups(students, 4, 42)
	if count1 != 5 || count2 != 5 {
		t.Fatalf("group count = %d,%d; want 5", count1, count2)
	}
	if strings.Join(order1, ",") != strings.Join(order2, ",") {
		t.Fatal("seeded shuffle should be deterministic")
	}
}

func TestPlanAutoAssignGroups_SizeFour(t *testing.T) {
	students := make([]string, 20)
	for i := range students {
		students[i] = fmtStudentID(i)
	}
	count, _ := planAutoAssignGroups(students, 4, 1)
	if count != 5 {
		t.Fatalf("group count = %d; want 5", count)
	}
}

func fmtStudentID(i int) string {
	return "00000000-0000-4000-8000-" + strings.Repeat("0", 11) + string(rune('a'+i))
}

func TestGroupsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/groups") {
			_ = json.NewEncoder(w).Encode(groupsListBody{
				Groups: []groupPublic{{
					ID: "g1", GroupSetID: "s1", Name: "Team 1", MemberCount: 4,
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	groupsListCmd.SetOut(&out)
	if err := groupsListCmd.RunE(groupsListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("groups list: %v", err)
	}
	if !strings.Contains(out.String(), "Team 1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestGroupsCreate_AutoAssign(t *testing.T) {
	var membershipCalls int
	groupCounter := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/enrollment-groups/sets"):
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "set-1"})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/sets/set-1/groups"):
			groupCounter++
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "group-" + string(rune('0'+groupCounter))})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/enrollments"):
			rows := make([]enrollmentRow, 8)
			for i := range rows {
				rows[i] = enrollmentRow{ID: "enr-" + string(rune('a'+i)), UserID: "user-" + string(rune('a'+i)), Role: "student"}
			}
			_ = json.NewEncoder(w).Encode(enrollmentsListBody{Enrollments: rows})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/memberships"):
			membershipCalls++
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	groupsCreateFlags.auto = true
	groupsCreateFlags.set = "Teams"
	groupsCreateFlags.size = 4
	groupsCreateFlags.seed = 7
	defer func() {
		groupsCreateFlags.auto = false
		groupsCreateFlags.set = ""
		groupsCreateFlags.size = 4
		groupsCreateFlags.seed = 0
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	groupsCreateCmd.SetOut(&out)
	if err := groupsCreateCmd.RunE(groupsCreateCmd, []string{"CS101"}); err != nil {
		t.Fatalf("groups create: %v", err)
	}
	if membershipCalls != 8 {
		t.Fatalf("membership calls = %d; want 8", membershipCalls)
	}
}

func TestDiscussionsExport_RequiresYes(t *testing.T) {
	discussionsExportFlags.yes = false
	discussionsExportInput = strings.NewReader("n\n")
	defer func() {
		discussionsExportFlags.yes = false
		discussionsExportInput = nil
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: "http://127.0.0.1:9", APIKey: "test-key"}
	var out bytes.Buffer
	discussionsExportCmd.SetOut(&out)
	err := discussionsExportCmd.RunE(discussionsExportCmd, []string{"CS101"})
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Fatalf("expected abort, got %v", err)
	}
}