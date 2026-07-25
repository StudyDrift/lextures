package adaptivecontent

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

// Review reason labels for units-to-review ranking (AC.9 FR-1).
const (
	ReviewReasonRegressing      = "regressing"
	ReviewReasonLowFidelity     = "low_fidelity"
	ReviewReasonInsufficientData = "insufficient_data"
)

// RefreshCourseReport refreshes coverage + course report matview for one course (AC.9).
func RefreshCourseReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	if pool == nil {
		return nil
	}
	if err := acrepo.RefreshCoverageForCourse(ctx, pool, courseID); err != nil {
		return err
	}
	if err := acrepo.RefreshCourseReportMaterializedView(ctx, pool); err != nil {
		// Concurrent refresh can fail on empty/first populate — fall back to non-concurrent.
		if _, err2 := pool.Exec(ctx, `REFRESH MATERIALIZED VIEW analytics.adaptive_content_course_report`); err2 != nil {
			return err
		}
	}
	return nil
}

// RefreshAllCourseReports refreshes coverage for every ACE-enabled course and the matview.
func RefreshAllCourseReports(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return nil
	}
	ids, err := acrepo.ListCourseIDsWithAdaptiveContentEnabled(ctx, pool)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := acrepo.RefreshCoverageForCourse(ctx, pool, id); err != nil {
			slog.Error("adaptivecontent: coverage refresh failed", "course_id", id, "err", err)
		}
	}
	if err := acrepo.RefreshCourseReportMaterializedView(ctx, pool); err != nil {
		if _, err2 := pool.Exec(ctx, `REFRESH MATERIALIZED VIEW analytics.adaptive_content_course_report`); err2 != nil {
			return err
		}
	}
	return nil
}

func coveragePct(adapted, eligible int) float64 {
	if eligible <= 0 {
		return 0
	}
	return float64(adapted) / float64(eligible) * 100
}

func mapUnitEffectiveness(
	cache *acrepo.EffectivenessCacheRow,
	modes []acrepo.ModeEffectivenessRow,
	variants []acrepo.VariantEffectivenessRow,
	unitID uuid.UUID,
) acmodel.UnitEffectiveness {
	out := acmodel.UnitEffectiveness{
		UnitID:        unitID,
		Verdict:       VerdictInsufficientData,
		ByMode:        []acmodel.ModeEffectiveness{},
		ByVariant:     []acmodel.VariantEffectiveness{},
		SmallCellMinN: SmallCellMinN,
		MinNPerArm:    MinNPerArm,
	}
	if cache != nil {
		out.NTreatment = cache.NTreatment
		out.NHoldout = cache.NHoldout
		out.MeanLiftTreatment = cache.MeanLiftTreatment
		out.MeanLiftHoldout = cache.MeanLiftHoldout
		out.TreatmentMinusHoldout = cache.TreatmentMinusHoldout
		out.DiffStdError = cache.DiffStdError
		out.MeanMasteryDeltaTreatment = cache.MeanMasteryDeltaTreatment
		out.MeanMasteryDeltaHoldout = cache.MeanMasteryDeltaHoldout
		out.Verdict = cache.Verdict
		t := cache.RefreshedAt
		out.RefreshedAt = &t
	}
	for _, m := range modes {
		out.ByMode = append(out.ByMode, acmodel.ModeEffectiveness{
			EmphasisMode: m.EmphasisMode,
			N:            m.N,
			MeanLift:     m.MeanLift,
		})
	}
	for _, v := range variants {
		out.ByVariant = append(out.ByVariant, acmodel.VariantEffectiveness{
			VariantID: v.VariantID,
			N:         v.N,
			MeanLift:  v.MeanLift,
		})
	}
	return out
}

func rankUnitsToReview(
	courseCode string,
	units []acmodel.UnitEffectiveness,
	fidelity []acrepo.UnitFidelityRow,
) []acmodel.UnitToReview {
	fidByUnit := make(map[uuid.UUID]acrepo.UnitFidelityRow, len(fidelity))
	for _, f := range fidelity {
		fidByUnit[f.UnitID] = f
	}
	var out []acmodel.UnitToReview
	seen := make(map[uuid.UUID]bool)
	workspaceBase := "/courses/" + courseCode + "/settings/adaptive-content"

	// 1) Regressing first.
	for _, u := range units {
		if u.Verdict != VerdictRegressing {
			continue
		}
		out = append(out, acmodel.UnitToReview{
			UnitID:       u.UnitID,
			Reason:       ReviewReasonRegressing,
			Verdict:      u.Verdict,
			MeanLift:     u.TreatmentMinusHoldout,
			WorkspaceURL: workspaceBase + "?unit=" + u.UnitID.String(),
		})
		seen[u.UnitID] = true
	}
	// 2) Low fidelity (mean fidelity below unit min when variants exist).
	for _, f := range fidelity {
		if seen[f.UnitID] || f.NVariants == 0 || f.MeanFidelity == nil {
			continue
		}
		if float64(*f.MeanFidelity) >= f.MinFidelity {
			continue
		}
		verdict := VerdictInsufficientData
		var lift *float32
		for _, u := range units {
			if u.UnitID == f.UnitID {
				verdict = u.Verdict
				lift = u.TreatmentMinusHoldout
				break
			}
		}
		mf := *f.MeanFidelity
		out = append(out, acmodel.UnitToReview{
			UnitID:       f.UnitID,
			Reason:       ReviewReasonLowFidelity,
			Verdict:      verdict,
			MeanFidelity: &mf,
			MeanLift:     lift,
			WorkspaceURL: workspaceBase + "?unit=" + f.UnitID.String(),
		})
		seen[f.UnitID] = true
	}
	// 3) Insufficient data (units with cache rows).
	for _, u := range units {
		if seen[u.UnitID] || u.Verdict != VerdictInsufficientData {
			continue
		}
		out = append(out, acmodel.UnitToReview{
			UnitID:       u.UnitID,
			Reason:       ReviewReasonInsufficientData,
			Verdict:      u.Verdict,
			MeanLift:     u.TreatmentMinusHoldout,
			WorkspaceURL: workspaceBase + "?unit=" + u.UnitID.String(),
		})
		seen[u.UnitID] = true
	}
	return out
}

func aggregateModes(units []acmodel.UnitEffectiveness) []acmodel.ModeBreakdown {
	type acc struct {
		n    int
		sum  float64
		has  bool
	}
	by := map[string]*acc{}
	for _, u := range units {
		for _, m := range u.ByMode {
			a := by[m.EmphasisMode]
			if a == nil {
				a = &acc{}
				by[m.EmphasisMode] = a
			}
			a.n += m.N
			if m.MeanLift != nil && m.N >= SmallCellMinN {
				a.sum += float64(*m.MeanLift) * float64(m.N)
				a.has = true
			}
		}
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]acmodel.ModeBreakdown, 0, len(keys))
	for _, k := range keys {
		a := by[k]
		row := acmodel.ModeBreakdown{EmphasisMode: k, N: a.n}
		if a.has && a.n >= SmallCellMinN {
			v := float32(a.sum / float64(a.n))
			row.MeanLift = &v
		}
		out = append(out, row)
	}
	return out
}

// BuildCourseReport assembles the instructor Adaptive Content report from caches (AC.9 FR-1/FR-7).
func BuildCourseReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, courseCode string) (acmodel.CourseReportResponse, error) {
	out := acmodel.CourseReportResponse{
		CourseID:      courseID,
		CourseCode:    courseCode,
		Empty:         true,
		UnitsToReview: []acmodel.UnitToReview{},
		Units:         []acmodel.UnitEffectiveness{},
		ByMode:        []acmodel.ModeBreakdown{},
		SmallCellMinN: SmallCellMinN,
		MinNPerArm:    MinNPerArm,
		Cost: acmodel.CourseReportCost{
			Unlimited: true,
			PeriodStart: func() string {
				now := time.Now().UTC()
				return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			}(),
		},
	}

	unitsRows, err := acrepo.ListUnits(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	out.NUnits = len(unitsRows)
	for _, u := range unitsRows {
		if u.Status == "active" {
			out.NActiveUnits++
		}
	}

	rollup, err := acrepo.GetCourseReportRollup(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	if rollup != nil {
		out.NUnits = rollup.NUnits
		out.NActiveUnits = rollup.NActiveUnits
		out.MeanLiftVsControl = rollup.MeanLiftVsControl
		out.NHelping = rollup.NHelping
		out.NRegressing = rollup.NRegressing
		out.NInsufficient = rollup.NInsufficient
		out.NNoEffect = rollup.NNoEffect
		t := rollup.RefreshedAt
		out.DataAsOf = &t
		out.Cost.TokensUsedPeriod = rollup.TokensUsedPeriod
		out.Cost.MonthlyTokenBudget = rollup.MonthlyTokenBudget
	}

	cov, err := acrepo.GetCoverage(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	if cov != nil {
		out.Coverage = acmodel.CourseReportCoverage{
			EligibleContentItems:  cov.EligibleContentItems,
			AdaptedUnits:          cov.AdaptedUnits,
			CoveragePct:           coveragePct(cov.AdaptedUnits, cov.EligibleContentItems),
			StudentsProfiled:      cov.StudentsProfiled,
			StudentsServedVariant: cov.StudentsServedVariant,
			StudentsHoldout:       cov.StudentsHoldout,
		}
		if out.DataAsOf == nil || cov.RefreshedAt.After(*out.DataAsOf) {
			t := cov.RefreshedAt
			out.DataAsOf = &t
		}
	}

	settings, err := acrepo.GetSettings(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	if settings == nil {
		def := acrepo.DefaultSettings(courseID)
		settings = &def
	}
	out.Cost.MonthlyTokenBudget = settings.MonthlyTokenBudget
	out.Cost.TokensUsedPeriod = settings.TokensUsedPeriod
	if settings.BudgetPeriodStart != nil {
		out.Cost.PeriodStart = settings.BudgetPeriodStart.UTC().Format("2006-01-02")
	} else {
		now := time.Now().UTC()
		out.Cost.PeriodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	if settings.MonthlyTokenBudget <= 0 {
		out.Cost.Unlimited = true
		out.Cost.BudgetRemaining = nil
	} else {
		out.Cost.Unlimited = false
		rem := settings.MonthlyTokenBudget - settings.TokensUsedPeriod
		if rem < 0 {
			rem = 0
		}
		out.Cost.BudgetRemaining = &rem
	}

	caches, err := acrepo.ListEffectivenessForCourse(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	units := make([]acmodel.UnitEffectiveness, 0, len(caches))
	for i := range caches {
		c := caches[i]
		modes, err := acrepo.ListModeEffectiveness(ctx, pool, c.UnitID)
		if err != nil {
			return out, err
		}
		variants, err := acrepo.ListVariantEffectiveness(ctx, pool, c.UnitID)
		if err != nil {
			return out, err
		}
		units = append(units, mapUnitEffectiveness(&c, modes, variants, c.UnitID))
	}
	out.Units = units
	out.ByMode = aggregateModes(units)

	fid, err := acrepo.ListUnitMeanFidelity(ctx, pool, courseID)
	if err != nil {
		return out, err
	}
	out.UnitsToReview = rankUnitsToReview(courseCode, units, fid)

	out.Empty = out.NUnits == 0 &&
		out.Coverage.StudentsProfiled == 0 &&
		out.Coverage.StudentsServedVariant == 0 &&
		len(out.Units) == 0

	return out, nil
}

// WriteCourseReportCSV streams the instructor report as CSV (AC.9 FR-3).
func WriteCourseReportCSV(w io.Writer, report acmodel.CourseReportResponse) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"section", "unit_id", "verdict", "reason", "n_treatment", "n_holdout",
		"treatment_minus_holdout", "mean_lift", "emphasis_mode", "n", "mean_fidelity",
		"coverage_pct", "students_served_variant", "tokens_used_period", "monthly_token_budget",
		"data_as_of",
	})
	asOf := ""
	if report.DataAsOf != nil {
		asOf = report.DataAsOf.UTC().Format(time.RFC3339)
	}
	liftStr := ""
	if report.MeanLiftVsControl != nil {
		liftStr = fmt.Sprintf("%g", *report.MeanLiftVsControl)
	}
	_ = cw.Write([]string{
		"summary", "", "", "", "", "",
		liftStr, "", "", "", "",
		fmt.Sprintf("%g", report.Coverage.CoveragePct),
		fmt.Sprintf("%d", report.Coverage.StudentsServedVariant),
		fmt.Sprintf("%d", report.Cost.TokensUsedPeriod),
		fmt.Sprintf("%d", report.Cost.MonthlyTokenBudget),
		asOf,
	})
	for _, u := range report.UnitsToReview {
		ml := ""
		if u.MeanLift != nil {
			ml = fmt.Sprintf("%g", *u.MeanLift)
		}
		mf := ""
		if u.MeanFidelity != nil {
			mf = fmt.Sprintf("%g", *u.MeanFidelity)
		}
		_ = cw.Write([]string{
			"units_to_review", u.UnitID.String(), u.Verdict, u.Reason, "", "",
			ml, "", "", "", mf, "", "", "", "", asOf,
		})
	}
	for _, u := range report.Units {
		tmh := ""
		if u.TreatmentMinusHoldout != nil {
			// Preserve small-cell suppression: omit when either arm below min.
			if u.NTreatment >= SmallCellMinN && u.NHoldout >= SmallCellMinN {
				tmh = fmt.Sprintf("%g", *u.TreatmentMinusHoldout)
			}
		}
		_ = cw.Write([]string{
			"unit", u.UnitID.String(), u.Verdict, "",
			fmt.Sprintf("%d", u.NTreatment), fmt.Sprintf("%d", u.NHoldout),
			tmh, "", "", "", "", "", "", "", "", asOf,
		})
		for _, m := range u.ByMode {
			ml := ""
			if m.MeanLift != nil && m.N >= SmallCellMinN {
				ml = fmt.Sprintf("%g", *m.MeanLift)
			}
			_ = cw.Write([]string{
				"mode", u.UnitID.String(), "", "", "", "",
				"", ml, m.EmphasisMode, fmt.Sprintf("%d", m.N), "",
				"", "", "", "", asOf,
			})
		}
	}
	cw.Flush()
	return cw.Error()
}

// BuildAdminReport assembles the org-wide Adaptive Content rollup (AC.9 FR-2).
func BuildAdminReport(ctx context.Context, pool *pgxpool.Pool) (acmodel.AdminReportResponse, error) {
	out := acmodel.AdminReportResponse{
		Courses:       []acmodel.AdminReportCourse{},
		SmallCellMinN: SmallCellMinN,
		KillSwitch:    KillSwitchEngaged(),
	}
	SyncDurableKillSwitchFromDB(ctx, pool)
	out.KillSwitch = KillSwitchEngaged()

	var err error
	out.CoursesUsingACE, err = acrepo.CountEnabledACECourses(ctx, pool)
	if err != nil {
		return out, err
	}
	out.StudentsImpacted, err = acrepo.SumCoverageStudentsImpacted(ctx, pool)
	if err != nil {
		return out, err
	}
	out.CostUSD30d, err = acrepo.SumAdaptiveContentCostUSD(ctx, pool)
	if err != nil {
		return out, err
	}
	out.BudgetHeadroomTokens, err = acrepo.SumBudgetHeadroomTokens(ctx, pool)
	if err != nil {
		return out, err
	}
	out.AggregateLift, err = acrepo.MeanAggregateLiftVsControl(ctx, pool)
	if err != nil {
		return out, err
	}
	out.DisparityFlags, err = acrepo.CountDisparityFlags(ctx, pool, nil)
	if err != nil {
		return out, err
	}
	out.OpenContests, err = acrepo.CountOpenContestsPlatform(ctx, pool)
	if err != nil {
		return out, err
	}
	out.RegressingUnits, err = acrepo.CountRegressingUnits(ctx, pool)
	if err != nil {
		return out, err
	}
	out.GenerationPaused, _ = acrepo.GetPlatformGenerationPaused(ctx, pool)
	out.QueueDepth, _ = acrepo.CountPendingJobs(ctx, pool)

	rows, err := acrepo.ListAdminCourseRollups(ctx, pool, 200)
	if err != nil {
		return out, err
	}
	var latest *time.Time
	for _, r := range rows {
		pct := coveragePct(r.AdaptedUnits, r.EligibleContentItems)
		out.Courses = append(out.Courses, acmodel.AdminReportCourse{
			CourseID:              r.CourseID,
			CourseCode:            r.CourseCode,
			Title:                 r.Title,
			NUnits:                r.NUnits,
			NActiveUnits:          r.NActiveUnits,
			MeanLiftVsControl:     r.MeanLiftVsControl,
			NRegressing:           r.NRegressing,
			NHelping:              r.NHelping,
			TokensUsedPeriod:      r.TokensUsedPeriod,
			MonthlyTokenBudget:    r.MonthlyTokenBudget,
			CoveragePct:           pct,
			StudentsServedVariant: r.StudentsServedVariant,
			DisparityFlags:        r.DisparityFlags,
			OpenContests:          r.OpenContests,
			ReportURL:             "/courses/" + r.CourseCode + "/settings/adaptive-content?tab=report",
		})
		if r.ReportRefreshedAt != nil && (latest == nil || r.ReportRefreshedAt.After(*latest)) {
			t := *r.ReportRefreshedAt
			latest = &t
		}
		if r.CoverageRefreshedAt != nil && (latest == nil || r.CoverageRefreshedAt.After(*latest)) {
			t := *r.CoverageRefreshedAt
			latest = &t
		}
	}
	out.DataAsOf = latest
	return out, nil
}

// WriteAdminReportCSV streams the admin org report as CSV (AC.9 FR-3).
func WriteAdminReportCSV(w io.Writer, report acmodel.AdminReportResponse) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"section", "course_code", "title", "n_units", "n_active_units",
		"mean_lift_vs_control", "n_regressing", "n_helping",
		"coverage_pct", "students_served_variant", "tokens_used_period",
		"monthly_token_budget", "disparity_flags", "open_contests",
		"courses_using_ace", "students_impacted", "cost_usd_30d",
		"budget_headroom_tokens", "aggregate_lift", "kill_switch", "data_as_of",
	})
	asOf := ""
	if report.DataAsOf != nil {
		asOf = report.DataAsOf.UTC().Format(time.RFC3339)
	}
	agg := ""
	if report.AggregateLift != nil {
		agg = fmt.Sprintf("%g", *report.AggregateLift)
	}
	_ = cw.Write([]string{
		"summary", "", "", "", "",
		"", fmt.Sprintf("%d", report.RegressingUnits), "",
		"", fmt.Sprintf("%d", report.StudentsImpacted), "",
		"", fmt.Sprintf("%d", report.DisparityFlags), fmt.Sprintf("%d", report.OpenContests),
		fmt.Sprintf("%d", report.CoursesUsingACE), fmt.Sprintf("%d", report.StudentsImpacted),
		fmt.Sprintf("%g", report.CostUSD30d),
		fmt.Sprintf("%d", report.BudgetHeadroomTokens), agg,
		fmt.Sprintf("%t", report.KillSwitch), asOf,
	})
	for _, c := range report.Courses {
		ml := ""
		if c.MeanLiftVsControl != nil {
			ml = fmt.Sprintf("%g", *c.MeanLiftVsControl)
		}
		_ = cw.Write([]string{
			"course", c.CourseCode, c.Title,
			fmt.Sprintf("%d", c.NUnits), fmt.Sprintf("%d", c.NActiveUnits),
			ml, fmt.Sprintf("%d", c.NRegressing), fmt.Sprintf("%d", c.NHelping),
			fmt.Sprintf("%g", c.CoveragePct), fmt.Sprintf("%d", c.StudentsServedVariant),
			fmt.Sprintf("%d", c.TokensUsedPeriod), fmt.Sprintf("%d", c.MonthlyTokenBudget),
			fmt.Sprintf("%d", c.DisparityFlags), fmt.Sprintf("%d", c.OpenContests),
			"", "", "", "", "", "", asOf,
		})
	}
	cw.Flush()
	return cw.Error()
}
