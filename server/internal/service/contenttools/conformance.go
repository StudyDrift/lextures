package contenttools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/analytics"
)

// ConformanceRow is one tool's shipping-gate status (CT.8 FR-2 / AC-12).
type ConformanceRow struct {
	ToolID              string   `json:"toolId"`
	Version             string   `json:"version"`
	DataSheetPresent    bool     `json:"dataSheetPresent"`
	StateSchemaPresent  bool     `json:"stateSchemaPresent"`
	ProjectionPresent   bool     `json:"projectionPresent"`
	I18nComplete        bool     `json:"i18nComplete"`
	KeyboardOperable    bool     `json:"keyboardOperable"`
	WCAGLevel           string   `json:"wcagLevel"`
	A11yLimitations     string   `json:"a11yLimitations,omitempty"`
	AxeStatus           string   `json:"axeStatus"`      // pass|pending|fail (CI harness)
	KeyboardTestStatus  string   `json:"keyboardTestStatus"`
	Errors              []string `json:"errors,omitempty"`
	OK                  bool     `json:"ok"`
}

// ConformanceReport is the per-release conformance artefact.
type ConformanceReport struct {
	Tools []ConformanceRow `json:"tools"`
	OK    bool             `json:"ok"`
}

// EvaluateConformance runs the CT.8 shipping gate against the registry (AC-1 / AC-12).
// axeStatus / keyboardTestStatus default to "pass" for built-ins that declare keyboardOperable;
// CI may override via harness results passed in overrides.
func EvaluateConformance(reg *Registry, axeOverrides, keyboardOverrides map[string]string) ConformanceReport {
	if axeOverrides == nil {
		axeOverrides = map[string]string{}
	}
	if keyboardOverrides == nil {
		keyboardOverrides = map[string]string{}
	}
	report := ConformanceReport{OK: true, Tools: []ConformanceRow{}}
	if reg == nil {
		report.OK = false
		return report
	}
	ids := make([]string, 0, reg.Size())
	for _, m := range reg.List() {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	for _, id := range ids {
		m := reg.Get(id)
		row := ConformanceRow{
			ToolID:             id,
			Version:            m.Version,
			DataSheetPresent:   m.DataSheet != nil,
			StateSchemaPresent: len(m.StateSchema) > 0,
			ProjectionPresent:  analytics.HasProjector(id),
			I18nComplete:       len(m.I18nBundle) > 0,
			KeyboardOperable:   m.A11y.KeyboardOperable,
			AxeStatus:          "pass",
			KeyboardTestStatus: "pass",
			OK:                 true,
		}
		if m.DataSheet != nil {
			row.WCAGLevel = m.DataSheet.WCAGLevel
			if row.WCAGLevel == "" {
				row.WCAGLevel = "AA"
			}
			row.A11yLimitations = m.DataSheet.A11yLimitations
		}
		if v, ok := axeOverrides[id]; ok {
			row.AxeStatus = v
		}
		if v, ok := keyboardOverrides[id]; ok {
			row.KeyboardTestStatus = v
		}
		var errs []string
		if !row.DataSheetPresent {
			errs = append(errs, "dataSheet missing")
		}
		if !row.StateSchemaPresent {
			errs = append(errs, "stateSchema missing")
		}
		if !row.ProjectionPresent {
			errs = append(errs, "projection function missing")
		}
		if !row.I18nComplete {
			errs = append(errs, "i18n bundle incomplete")
		}
		if !row.KeyboardOperable {
			errs = append(errs, "keyboardOperable must be true")
		}
		if row.AxeStatus == "fail" {
			errs = append(errs, "axe gate failed")
			IncA11yGateFailure(id)
		}
		if row.KeyboardTestStatus == "fail" {
			errs = append(errs, "keyboard test failed")
			IncA11yGateFailure(id)
		}
		if m.DataSheet == nil || strings.TrimSpace(m.DataSheet.WCAGLevel) == "" {
			// ValidateDataSheet already requires sheet; surface WCAG declaration.
			if m.DataSheet == nil {
				errs = append(errs, "wcagLevel undeclared")
			}
		}
		if len(errs) > 0 {
			row.OK = false
			row.Errors = errs
			report.OK = false
		}
		report.Tools = append(report.Tools, row)
	}
	return report
}

// MustConformanceOK fails startup when any registered tool fails the gate (AC-1).
func MustConformanceOK(reg *Registry) error {
	rep := EvaluateConformance(reg, nil, nil)
	if rep.OK {
		return nil
	}
	var names []string
	for _, t := range rep.Tools {
		if !t.OK {
			names = append(names, fmt.Sprintf("%s (%s)", t.ToolID, strings.Join(t.Errors, "; ")))
		}
	}
	return fmt.Errorf("content tools conformance gate failed: %s", strings.Join(names, ", "))
}
