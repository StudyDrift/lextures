package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestRedactLicensePayload(t *testing.T) {
	out := redactLicensePayload(map[string]any{
		"tier": "enterprise",
		"licenseKey": "secret-key-value",
		"maxSeats": 100,
	})
	if out["licenseKey"] != "[REDACTED]" {
		t.Fatalf("key not redacted: %+v", out)
	}
	if out["tier"] != "enterprise" {
		t.Fatalf("tier changed: %+v", out)
	}
}

func TestFormatSeatUsage(t *testing.T) {
	lic := licenseRow{UsedSeats: 40, MaxSeats: 100, PercentUsed: 40}
	if !strings.Contains(formatSeatUsage(lic), "40/100") {
		t.Fatalf("seat usage = %q", formatSeatUsage(lic))
	}
	unlimited := licenseRow{UsedSeats: 10, Unlimited: true, MaxSeats: -1}
	if !strings.Contains(formatSeatUsage(unlimited), "unlimited") {
		t.Fatalf("unlimited = %q", formatSeatUsage(unlimited))
	}
}

func TestLicensesStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin-console/license" {
			_ = json.NewEncoder(w).Encode(licenseRow{
				OrgID: "org-1", Tier: "standard", MaxSeats: 50, UsedSeats: 10, PercentUsed: 20,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	licensesStatusCmd.SetOut(&out)
	if err := licensesStatusCmd.RunE(licensesStatusCmd, nil); err != nil {
		t.Fatalf("licenses status: %v", err)
	}
	if !strings.Contains(out.String(), "10/50") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestEntitlementsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/entitlements" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entitlements": []entitlementRow{{
					ID: "e1", EntitlementType: "course_access", Status: "active",
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	entitlementsListCmd.SetOut(&out)
	if err := entitlementsListCmd.RunE(entitlementsListCmd, nil); err != nil {
		t.Fatalf("entitlements list: %v", err)
	}
	if !strings.Contains(out.String(), "course_access") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRevenueReport_RequiresYes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]int{"pendingCents": 100})
	}))
	defer srv.Close()

	revenueReportFlags.yes = false
	defer func() { revenueReportFlags.yes = false }()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	revenueReportCmd.SetOut(&out)
	if err := revenueReportCmd.RunE(revenueReportCmd, nil); err == nil {
		t.Fatal("expected --yes gate")
	}
}

func TestRevenueReport_WithYes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin/revenue/summary" {
			_ = json.NewEncoder(w).Encode(map[string]int{"totalSalesCents": 5000})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	revenueReportFlags.yes = true
	defer func() { revenueReportFlags.yes = false }()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	revenueReportCmd.SetOut(&out)
	if err := revenueReportCmd.RunE(revenueReportCmd, nil); err != nil {
		t.Fatalf("revenue report: %v", err)
	}
	if !strings.Contains(out.String(), "totalSalesCents") {
		t.Fatalf("output = %q", out.String())
	}
}