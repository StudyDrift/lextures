package codeexecution

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultTimeLimitMs = 2000
	maxOutputBytes     = 64 * 1024
	maxCodeBytes       = 64 * 1024
)

// RunTests executes student code against each test case with resource caps.
func (s Service) RunTests(ctx context.Context, req RunRequest) (RunResponse, error) {
	if ctx == nil {
		return RunResponse{}, fmt.Errorf("context is nil")
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return RunResponse{}, fmt.Errorf("submission code is empty")
	}
	if len(code) > maxCodeBytes {
		return RunResponse{}, fmt.Errorf("submission code exceeds size limit")
	}
	if len(req.Tests) == 0 {
		return RunResponse{}, fmt.Errorf("test suite is empty")
	}

	runtime := normalizeRuntime(req.Runtime)
	cmdName, fileExt, err := runtimeCommand(runtime)
	if err != nil {
		return RunResponse{Results: []TestResult{{
			Status: StatusRE, Stderr: err.Error(),
		}}}, nil
	}

	dir, err := os.MkdirTemp("", "lextures-code-*")
	if err != nil {
		return RunResponse{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	srcPath := filepath.Join(dir, "submission"+fileExt)
	if writeErr := os.WriteFile(srcPath, []byte(code), 0o600); writeErr != nil {
		return RunResponse{}, writeErr
	}

	out := RunResponse{Results: make([]TestResult, 0, len(req.Tests))}
	for _, tc := range req.Tests {
		limitMs := tc.TimeLimitMs
		if limitMs <= 0 {
			limitMs = defaultTimeLimitMs
		}
		result := runOneTest(ctx, cmdName, srcPath, tc, limitMs)
		out.Results = append(out.Results, result)
		if result.Status == StatusCE {
			out.CompileError = result.Stderr
			break
		}
		if result.Status == StatusTLE {
			out.TimedOut = true
		}
	}
	return out, nil
}

func runOneTest(parent context.Context, cmdName, srcPath string, tc TestCase, limitMs int) TestResult {
	result := TestResult{
		TestCaseID:     tc.ID,
		ExpectedOutput: tc.ExpectedOutput,
		Status:         StatusPending,
	}
	runCtx, cancel := context.WithTimeout(parent, time.Duration(limitMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(runCtx, cmdName, srcPath)
	cmd.Stdin = strings.NewReader(tc.Input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	result.ExecutionMs = int(time.Since(start).Milliseconds())

	if runCtx.Err() == context.DeadlineExceeded {
		result.Status = StatusTLE
		result.Stderr = truncateOutput(stderr.String(), 4096)
		return result
	}

	stdoutStr := truncateOutput(stdout.String(), maxOutputBytes)
	stderrStr := truncateOutput(stderr.String(), 4096)
	result.ActualOutput = stdoutStr
	result.Stderr = stderrStr

	if runErr != nil {
		if isCompileError(stderrStr, cmdName) {
			result.Status = StatusCE
			return result
		}
		result.Status = StatusRE
		return result
	}

	if strings.TrimSpace(stdoutStr) == strings.TrimSpace(tc.ExpectedOutput) {
		result.Status = StatusPass
		result.Passed = true
	} else {
		result.Status = StatusFail
	}
	return result
}

func isCompileError(stderr, cmdName string) bool {
	s := strings.ToLower(stderr)
	if s == "" {
		return false
	}
	switch cmdName {
	case "python3", "python":
		return strings.Contains(s, "syntaxerror") || strings.Contains(s, "indentationerror") ||
			strings.Contains(s, "nameerror") && strings.Contains(s, "line")
	case "node":
		return strings.Contains(s, "syntaxerror")
	default:
		return strings.Contains(s, "error")
	}
}

func normalizeRuntime(runtime string) string {
	r := strings.TrimSpace(strings.ToLower(runtime))
	switch {
	case r == "" || strings.HasPrefix(r, "python"):
		return "python3"
	case strings.HasPrefix(r, "javascript") || strings.HasPrefix(r, "node"):
		return "javascript"
	default:
		return r
	}
}

func runtimeCommand(runtime string) (cmdName, fileExt string, err error) {
	switch runtime {
	case "python3", "python":
		if path, lookErr := exec.LookPath("python3"); lookErr == nil {
			return path, ".py", nil
		}
		if path, lookErr := exec.LookPath("python"); lookErr == nil {
			return path, ".py", nil
		}
		return "", "", fmt.Errorf("python runtime is not available")
	case "javascript", "node":
		if path, lookErr := exec.LookPath("node"); lookErr == nil {
			return path, ".js", nil
		}
		return "", "", fmt.Errorf("javascript runtime is not available")
	default:
		return "", "", fmt.Errorf("unsupported runtime %q", runtime)
	}
}

func truncateOutput(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
