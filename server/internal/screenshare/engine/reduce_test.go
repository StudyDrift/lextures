package engine

import "testing"

func TestGrantHandOffStopsPrevious(t *testing.T) {
	s := State{Status: StatusOpen, Policy: PolicyRequest, ViewerCap: 50}
	s, evs, err := Reduce(s, ActionGrantPresent, "host1", "student1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.ActivePresenterID != "student1" || s.Status != StatusPresenting {
		t.Fatalf("after grant: %+v", s)
	}
	if len(evs) < 1 {
		t.Fatal("expected events")
	}

	s, evs, err = Reduce(s, ActionGrantPresent, "host1", "student2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.ActivePresenterID != "student2" {
		t.Fatalf("expected hand-off to student2, got %q", s.ActivePresenterID)
	}
	foundChange := false
	for _, e := range evs {
		if e.Type == "present_change" {
			foundChange = true
			if e.Payload["from"] != "student1" || e.Payload["to"] != "student2" {
				t.Fatalf("change payload: %+v", e.Payload)
			}
		}
	}
	if !foundChange {
		t.Fatal("expected present_change on hand-off")
	}
}

func TestGrantIdempotent(t *testing.T) {
	s := State{Status: StatusPresenting, Policy: PolicyRequest, ActivePresenterID: "a"}
	s2, evs, err := Reduce(s, ActionGrantPresent, "host", "a", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s2.ActivePresenterID != "a" {
		t.Fatal("presenter changed on idempotent grant")
	}
	if len(evs) != 0 {
		t.Fatalf("expected no events on idempotent grant, got %v", evs)
	}
}

func TestHostOnlyDeniesRequest(t *testing.T) {
	s := State{Status: StatusOpen, Policy: PolicyHostOnly}
	_, _, err := Reduce(s, ActionRequestPresent, "student", "", nil)
	if err == nil {
		t.Fatal("expected deny")
	}
	_, _, err = Reduce(s, ActionSelfPromote, "student", "", nil)
	if err == nil {
		t.Fatal("expected self-promote deny under host_only")
	}
}

func TestFreeForAllSelfPromote(t *testing.T) {
	s := State{Status: StatusOpen, Policy: PolicyFreeForAll}
	s, _, err := Reduce(s, ActionSelfPromote, "student", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.ActivePresenterID != "student" {
		t.Fatalf("got %q", s.ActivePresenterID)
	}
}

func TestRequestQueuesThenGrant(t *testing.T) {
	s := State{Status: StatusOpen, Policy: PolicyRequest}
	s, _, err := Reduce(s, ActionRequestPresent, "s1", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = Reduce(s, ActionRequestPresent, "s2", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.PendingRequests) != 2 {
		t.Fatalf("pending: %v", s.PendingRequests)
	}
	// Idempotent re-request
	s, _, err = Reduce(s, ActionRequestPresent, "s1", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.PendingRequests) != 2 {
		t.Fatalf("pending after re-request: %v", s.PendingRequests)
	}
	s, _, err = Reduce(s, ActionGrantPresent, "host", "s2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if contains(s.PendingRequests, "s2") {
		t.Fatal("granted user should leave queue")
	}
	if s.ActivePresenterID != "s2" {
		t.Fatal(s.ActivePresenterID)
	}
}

func TestStopAndEnd(t *testing.T) {
	s := State{Status: StatusPresenting, Policy: PolicyRequest, ActivePresenterID: "p"}
	s, evs, err := Reduce(s, ActionStopPresent, "p", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.ActivePresenterID != "" || s.Status != StatusOpen {
		t.Fatalf("%+v", s)
	}
	found := false
	for _, e := range evs {
		if e.Type == "present_stop" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing present_stop")
	}

	s, _, err = Reduce(s, ActionEndSession, "host", "", map[string]any{"by": "host"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusEnded {
		t.Fatal(s.Status)
	}
	_, _, err = Reduce(s, ActionGrantPresent, "host", "x", nil)
	if err == nil {
		t.Fatal("ended session must reject grant")
	}
}

func TestViewerCap(t *testing.T) {
	s := State{ViewerCount: 50, ViewerCap: 50}
	if CanJoinViewer(s) {
		t.Fatal("cap full")
	}
	s.ViewerCount = 49
	if !CanJoinViewer(s) {
		t.Fatal("should allow")
	}
}
