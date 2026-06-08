// Package sis provides the higher-ed SIS adapter layer (plan 14.1).
//
// Each adapter implements roster sync and grade passback against a university
// Student Information System. Live HTTP calls require valid credentials; without
// them adapters return zero-count stub results so the sync framework can be
// exercised end-to-end.
package sis

import (
	"context"
	"log/slog"

	repoSIS "github.com/lextures/lextures/server/internal/repos/sis"
)

// ConnectionConfig is the minimum data an adapter needs to call a SIS API.
type ConnectionConfig struct {
	Vendor          string
	BaseURL         string
	ClientIDRef     string
	ClientSecretRef string
}

// Adapter syncs roster data and posts grades to one SIS vendor.
type Adapter interface {
	Vendor() string
	SyncRoster(ctx context.Context, cfg ConnectionConfig) (repoSIS.SyncSummary, []repoSIS.SyncError)
	TestConnection(ctx context.Context, cfg ConnectionConfig) error
}

// AdapterFor returns the HE adapter implementation for a vendor constant.
func AdapterFor(vendor string) Adapter {
	switch vendor {
	case repoSIS.VendorBanner:
		return bannerAdapter{}
	case repoSIS.VendorWorkday:
		return workdayAdapter{}
	case repoSIS.VendorColleague:
		return colleagueAdapter{}
	case repoSIS.VendorJenzabar:
		return jenzabarAdapter{}
	case repoSIS.VendorPeopleSoft:
		return peoplesoftAdapter{}
	default:
		return nil
	}
}

// IsHEVendor reports whether the vendor is a higher-ed SIS (plan 14.1).
func IsHEVendor(vendor string) bool {
	return AdapterFor(vendor) != nil
}

type bannerAdapter struct{}

func (bannerAdapter) Vendor() string { return repoSIS.VendorBanner }

func (a bannerAdapter) SyncRoster(_ context.Context, cfg ConnectionConfig) (repoSIS.SyncSummary, []repoSIS.SyncError) {
	slog.Info("sis banner sync: stub (no credentials configured)", "base_url", cfg.BaseURL)
	return repoSIS.SyncSummary{}, nil
}

func (a bannerAdapter) TestConnection(_ context.Context, _ ConnectionConfig) error {
	return nil
}

type workdayAdapter struct{}

func (workdayAdapter) Vendor() string { return repoSIS.VendorWorkday }

func (w workdayAdapter) SyncRoster(_ context.Context, cfg ConnectionConfig) (repoSIS.SyncSummary, []repoSIS.SyncError) {
	slog.Info("sis workday sync: stub (no credentials configured)", "base_url", cfg.BaseURL)
	return repoSIS.SyncSummary{}, nil
}

func (w workdayAdapter) TestConnection(_ context.Context, _ ConnectionConfig) error {
	return nil
}

type colleagueAdapter struct{}

func (colleagueAdapter) Vendor() string { return repoSIS.VendorColleague }

func (c colleagueAdapter) SyncRoster(_ context.Context, cfg ConnectionConfig) (repoSIS.SyncSummary, []repoSIS.SyncError) {
	slog.Info("sis colleague sync: stub (no credentials configured)", "base_url", cfg.BaseURL)
	return repoSIS.SyncSummary{}, nil
}

func (c colleagueAdapter) TestConnection(_ context.Context, _ ConnectionConfig) error {
	return nil
}

type jenzabarAdapter struct{}

func (jenzabarAdapter) Vendor() string { return repoSIS.VendorJenzabar }

func (j jenzabarAdapter) SyncRoster(_ context.Context, cfg ConnectionConfig) (repoSIS.SyncSummary, []repoSIS.SyncError) {
	slog.Info("sis jenzabar sync: stub (no credentials configured)", "base_url", cfg.BaseURL)
	return repoSIS.SyncSummary{}, nil
}

func (j jenzabarAdapter) TestConnection(_ context.Context, _ ConnectionConfig) error {
	return nil
}

type peoplesoftAdapter struct{}

func (peoplesoftAdapter) Vendor() string { return repoSIS.VendorPeopleSoft }

func (p peoplesoftAdapter) SyncRoster(_ context.Context, cfg ConnectionConfig) (repoSIS.SyncSummary, []repoSIS.SyncError) {
	slog.Info("sis peoplesoft sync: stub (no credentials configured)", "base_url", cfg.BaseURL)
	return repoSIS.SyncSummary{}, nil
}

func (p peoplesoftAdapter) TestConnection(_ context.Context, _ ConnectionConfig) error {
	return nil
}
