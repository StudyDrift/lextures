package gradingagent

import (
	"bytes"
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	markitdown "github.com/conductor-oss/markitdown"
)

func TestJoinSubmissions_doubleNewlineDelimited(t *testing.T) {
	got := JoinSubmissions([]string{" first ", "", "second", " third "})
	want := "first\n\nsecond\n\nthird"
	if got != want {
		t.Fatalf("JoinSubmissions = %q want %q", got, want)
	}
}

func TestSubmissionMarkdownConverter_plainText(t *testing.T) {
	result, err := submissionMarkdownConverter.ConvertReader(bytes.NewReader([]byte("Hello from the student essay.")), markitdown.StreamInfo{
		Extension: ".txt",
		Filename:  "essay.txt",
		MIMEType:  "text/plain",
	})
	if err != nil {
		t.Fatalf("ConvertReader: %v", err)
	}
	if result.Markdown != "Hello from the student essay." {
		t.Fatalf("markdown = %q", result.Markdown)
	}
}

func TestSubstituteWorkflowPromptVariables_joinsSubmissions(t *testing.T) {
	g := samplePromptVariableGraph()
	prompt := "Files:\n$StudentSubmission.Submissions"
	resolved := SubstituteWorkflowPromptVariables(&g, "ai1", prompt, PromptVariableContext{
		Submissions: []string{"Essay part one", "Essay part two"},
	})
	want := "Files:\nEssay part one\n\nEssay part two"
	if resolved != want {
		t.Fatalf("resolved = %q want %q", resolved, want)
	}
}

func TestLoadSubmissionTextForSubmission_requiresAttachment(t *testing.T) {
	svc := &Service{}
	_, err := svc.LoadSubmissionTextForSubmission(context.Background(), "C-TEST", &moduleassignmentsubmissions.SubmissionRow{})
	if err == nil {
		t.Fatal("expected error without attachment")
	}
}