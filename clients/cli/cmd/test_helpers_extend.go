package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/client"
)

func httptestClient(serverURL string) *client.Client {
	return client.New(serverURL, "test-key")
}

func writeTempJSON(t *testing.T, v any) string {
	t.Helper()
	f, err := os.CreateTemp("", "lextures-cli-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := json.NewEncoder(f).Encode(v); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func writeTempJSONL(t *testing.T, line string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "prompts.jsonl")
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}