package contenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// SyncDataSheets mirrors registry Tool Data Sheets into course.content_tool_data_sheets.
func SyncDataSheets(ctx context.Context, pool *pgxpool.Pool, reg *Registry) error {
	if pool == nil || reg == nil {
		return nil
	}
	for _, m := range reg.List() {
		if m.DataSheet == nil {
			return fmt.Errorf("tool %s: missing dataSheet at sync", m.ID)
		}
		collects, err := json.Marshal(m.DataSheet.Collects)
		if err != nil {
			return err
		}
		aiJSON := []byte("{}")
		if m.DataSheet.AITransparency != nil {
			aiJSON, err = json.Marshal(m.DataSheet.AITransparency)
			if err != nil {
				return err
			}
		}
		procs := m.DataSheet.Processors
		if procs == nil {
			procs = []string{}
		}
		level := m.DataSheet.WCAGLevel
		if level == "" {
			level = "AA"
		}
		var lim *string
		if m.DataSheet.A11yLimitations != "" {
			s := m.DataSheet.A11yLimitations
			lim = &s
		}
		row := ctrepo.DataSheetRow{
			ToolID:             m.ID,
			Version:            m.Version,
			CollectsJSON:       collects,
			LeavesPlatform:     m.DataSheet.LeavesPlatform,
			Processors:         procs,
			Visibility:         m.DataSheet.Visibility,
			WCAGLevel:          level,
			A11yLimitations:    lim,
			AITransparencyJSON: aiJSON,
		}
		if err := ctrepo.UpsertDataSheet(ctx, pool, row); err != nil {
			return err
		}
	}
	slog.Info("contenttools.data_sheets_synced", "tools", reg.Size())
	return nil
}

// DataSheetPublic is the trust-centre shape.
type DataSheetPublic struct {
	ToolID           string                     `json:"toolId"`
	Version          string                     `json:"version"`
	Name             string                     `json:"name"`
	Collects         map[string]DataSheetCollect `json:"collects"`
	LeavesPlatform   bool                       `json:"leavesPlatform"`
	Processors       []string                   `json:"processors"`
	Visibility       string                     `json:"visibility"`
	WCAGLevel        string                     `json:"wcagLevel"`
	A11yLimitations  string                     `json:"a11yLimitations,omitempty"`
	AITransparency   *AITransparencyDecl        `json:"aiTransparency,omitempty"`
	Capabilities     []string                   `json:"capabilities"`
}
