package adaptivecontent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"

	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

func TestRankUnitsToReview_RegressingFirst(t *testing.T) {
	u1 := uuid.New()
	u2 := uuid.New()
	u3 := uuid.New()
	liftNeg := float32(-8)
	liftPos := float32(6)
	lowFid := float32(0.5)
	units := []acmodel.UnitEffectiveness{
		{UnitID: u1, Verdict: VerdictInsufficientData, TreatmentMinusHoldout: &liftPos},
		{UnitID: u2, Verdict: VerdictRegressing, TreatmentMinusHoldout: &liftNeg},
		{UnitID: u3, Verdict: VerdictHelping, TreatmentMinusHoldout: &liftPos},
	}
	fid := []acrepo.UnitFidelityRow{
		{UnitID: u3, MeanFidelity: &lowFid, MinFidelity: 0.85, NVariants: 2},
	}
	out := rankUnitsToReview("DEMO", units, fid)
	if len(out) < 2 {
		t.Fatalf("expected at least 2 review rows, got %d", len(out))
	}
	if out[0].UnitID != u2 || out[0].Reason != ReviewReasonRegressing {
		t.Fatalf("expected regressing unit first, got %+v", out[0])
	}
	if out[1].UnitID != u3 || out[1].Reason != ReviewReasonLowFidelity {
		t.Fatalf("expected low_fidelity second, got %+v", out[1])
	}
	if !strings.Contains(out[0].WorkspaceURL, "unit="+u2.String()) {
		t.Fatalf("workspace url missing unit id: %s", out[0].WorkspaceURL)
	}
}

func TestWriteCourseReportCSV_PreservesSuppression(t *testing.T) {
	unitID := uuid.New()
	lift := float32(12.5)
	report := acmodel.CourseReportResponse{
		CourseCode: "DEMO",
		Coverage:   acmodel.CourseReportCoverage{CoveragePct: 50, StudentsServedVariant: 3},
		Cost:       acmodel.CourseReportCost{TokensUsedPeriod: 100, MonthlyTokenBudget: 1000},
		Units: []acmodel.UnitEffectiveness{
			{
				UnitID:                unitID,
				Verdict:               VerdictHelping,
				NTreatment:            2, // below SmallCellMinN
				NHoldout:              2,
				TreatmentMinusHoldout: &lift,
				ByMode: []acmodel.ModeEffectiveness{
					{EmphasisMode: "introduce", N: 2, MeanLift: &lift},
				},
			},
		},
		SmallCellMinN: SmallCellMinN,
	}
	var buf bytes.Buffer
	if err := WriteCourseReportCSV(&buf, report); err != nil {
		t.Fatal(err)
	}
	csv := buf.String()
	if strings.Contains(csv, "12.5") {
		t.Fatalf("expected small-cell lift to be suppressed, got:\n%s", csv)
	}
	if !strings.Contains(csv, unitID.String()) {
		t.Fatalf("expected unit id in csv, got:\n%s", csv)
	}
}

func TestCoveragePct(t *testing.T) {
	if got := coveragePct(0, 0); got != 0 {
		t.Fatalf("empty eligible => 0, got %v", got)
	}
	if got := coveragePct(1, 4); got != 25 {
		t.Fatalf("want 25, got %v", got)
	}
}

func TestWriteAdminReportCSV_SummaryRow(t *testing.T) {
	agg := float32(3.5)
	report := acmodel.AdminReportResponse{
		CoursesUsingACE:  2,
		StudentsImpacted: 10,
		CostUSD30d:       1.25,
		AggregateLift:    &agg,
		KillSwitch:       false,
		Courses: []acmodel.AdminReportCourse{
			{CourseCode: "C1", Title: "Course 1", CoveragePct: 40},
		},
	}
	var buf bytes.Buffer
	if err := WriteAdminReportCSV(&buf, report); err != nil {
		t.Fatal(err)
	}
	csv := buf.String()
	if !strings.Contains(csv, "summary") || !strings.Contains(csv, "C1") {
		t.Fatalf("unexpected csv:\n%s", csv)
	}
}
