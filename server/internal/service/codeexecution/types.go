package codeexecution

// Status values for a single test-case execution.
const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusTLE     = "tle"
	StatusMLE     = "mle"
	StatusRE      = "re"
	StatusCE      = "ce"
	StatusPending = "pending"
)

// TestCase is one instructor-defined input/output pair.
type TestCase struct {
	ID             string
	Input          string
	ExpectedOutput string
	IsHidden       bool
	TimeLimitMs    int
	MemoryLimitKb  int
}

// TestResult is the outcome of running one test case.
type TestResult struct {
	TestCaseID     string `json:"testCaseId"`
	Status         string `json:"status"`
	Passed         bool   `json:"passed"`
	ActualOutput   string `json:"actualOutput"`
	ExpectedOutput string `json:"expectedOutput"`
	Stderr         string `json:"stderr,omitempty"`
	ExecutionMs    int    `json:"executionMs,omitempty"`
}

// RunRequest executes student code against a test suite.
type RunRequest struct {
	Runtime string
	Code    string
	Tests   []TestCase
}

// RunResponse aggregates per-test outcomes and run-level failures.
type RunResponse struct {
	Results      []TestResult
	CompileError string
	TimedOut     bool
}
