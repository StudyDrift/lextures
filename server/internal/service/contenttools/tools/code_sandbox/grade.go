package code_sandbox

import (
	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

// MapRunnerStatus converts a codeexecution test status into a tool RunStatus.
func MapRunnerStatus(resp codeexecution.RunResponse, primary codeexecution.TestResult) RunStatus {
	if resp.CompileError != "" {
		return StatusCompileError
	}
	if resp.TimedOut || primary.Status == codeexecution.StatusTLE {
		return StatusTimeout
	}
	if primary.Status == codeexecution.StatusMLE {
		return StatusMemory
	}
	if primary.Status == codeexecution.StatusCE {
		return StatusCompileError
	}
	if primary.Status == codeexecution.StatusRE {
		return StatusRuntimeError
	}
	if primary.Status == codeexecution.StatusPass || primary.Passed {
		return StatusOK
	}
	if primary.Status == codeexecution.StatusFail {
		// Fail on expected-output mismatch still counts as ok execution for Run.
		return StatusOK
	}
	return StatusError
}

// GradeCheck maps a full RunResponse against configured tests into learner-safe results.
func GradeCheck(cfg Config, resp codeexecution.RunResponse) (results []CheckTestResult, passed, total int, status RunStatus, stdout, stderr string) {
	total = len(cfg.Tests)
	byID := map[string]codeexecution.TestResult{}
	for _, r := range resp.Results {
		byID[r.TestCaseID] = r
	}

	status = StatusOK
	if resp.CompileError != "" {
		status = StatusCompileError
		stderr = TruncateOutput(resp.CompileError, cfg.OutputLimitBytes)
	} else if resp.TimedOut {
		status = StatusTimeout
	}

	results = make([]CheckTestResult, 0, len(cfg.Tests))
	for _, tc := range cfg.Tests {
		tr, ok := byID[tc.ID]
		entry := CheckTestResult{
			ID:     tc.ID,
			Name:   tc.Name,
			Hidden: tc.Hidden,
		}
		if !ok {
			entry.Passed = false
			if status == StatusOK {
				status = StatusError
			}
		} else {
			entry.Passed = tr.Passed
			if tr.Status == codeexecution.StatusCE {
				status = StatusCompileError
			} else if tr.Status == codeexecution.StatusTLE && status == StatusOK {
				status = StatusTimeout
			} else if tr.Status == codeexecution.StatusRE && status == StatusOK {
				status = StatusRuntimeError
			} else if tr.Status == codeexecution.StatusMLE && status == StatusOK {
				status = StatusMemory
			}
			if stdout == "" && tr.ActualOutput != "" && !tc.Hidden {
				stdout = TruncateOutput(tr.ActualOutput, cfg.OutputLimitBytes)
			}
			if stderr == "" && tr.Stderr != "" {
				stderr = TruncateOutput(tr.Stderr, cfg.OutputLimitBytes)
			}
		}
		if !entry.Passed && tc.Feedback != "" {
			entry.Feedback = tc.Feedback
		}
		if entry.Passed {
			passed++
		}
		results = append(results, entry)
	}

	if status == StatusOK && passed < total {
		// Execution succeeded but some assertions failed — keep status ok for history.
		status = StatusOK
	}
	return results, passed, total, status, stdout, stderr
}

// ToRunnerTests converts config tests to codeexecution.TestCase values.
func ToRunnerTests(cfg Config) []codeexecution.TestCase {
	out := make([]codeexecution.TestCase, 0, len(cfg.Tests))
	for _, tc := range cfg.Tests {
		out = append(out, codeexecution.TestCase{
			ID:             tc.ID,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			IsHidden:       tc.Hidden,
		})
	}
	return out
}

// SampleRunTest builds a single synthetic test for the Run action.
func SampleRunTest(stdin string) []codeexecution.TestCase {
	return []codeexecution.TestCase{{
		ID:             "_sample",
		Input:          stdin,
		ExpectedOutput: "", // unused; we ignore pass/fail for Run
		IsHidden:       false,
	}}
}
