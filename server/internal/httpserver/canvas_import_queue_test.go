package httpserver

import (
	"testing"

	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
)

func TestIncludeToRepo(t *testing.T) {
	got := includeToRepo(canvasImportInclude{Modules: true, Files: true})
	want := canvasimportjobs.Include{Modules: true, Files: true}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestCanvasImportJobAlreadyTerminal(t *testing.T) {
	t.Parallel()
	if canvasImportJobAlreadyTerminal(canvasimportjobs.StatusCompleted) != true {
		t.Fatal("completed should be terminal")
	}
	if canvasImportJobAlreadyTerminal(canvasimportjobs.StatusFailed) != true {
		t.Fatal("failed should be terminal")
	}
	if canvasImportJobAlreadyTerminal(canvasimportjobs.StatusQueued) {
		t.Fatal("queued should not be terminal")
	}
	if canvasImportJobAlreadyTerminal(canvasimportjobs.StatusProcessing) {
		t.Fatal("processing should not be terminal")
	}
}

func TestTerminalWSMessage_Statuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status   canvasimportjobs.Status
		terminal bool
	}{
		{canvasimportjobs.StatusCompleted, true},
		{canvasimportjobs.StatusFailed, true},
		{canvasimportjobs.StatusProcessing, false},
		{canvasimportjobs.StatusQueued, false},
	}
	for _, tc := range cases {
		job := &canvasimportjobs.Job{Status: tc.status}
		got := job.Status == canvasimportjobs.StatusCompleted || job.Status == canvasimportjobs.StatusFailed
		if got != tc.terminal {
			t.Fatalf("status %q terminal=%v want %v", tc.status, got, tc.terminal)
		}
	}
}
