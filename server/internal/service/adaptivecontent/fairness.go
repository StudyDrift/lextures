package adaptivecontent

import (
	"context"
	"log/slog"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notifevents"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// Fairness audit thresholds (AC.8 FR-3).
const (
	// FairnessDisparityThreshold is the absolute gap in mean fidelity or lift vs cohort mean
	// that raises a disparity flag (when group n ≥ FairnessMinN).
	FairnessDisparityThreshold = 0.10
	// FairnessMinN is the minimum cell size before a disparity can be flagged (also uses SmallCellMinN for suppression).
	FairnessMinN = 10

	FairnessDimLanguage       = "language"
	FairnessDimGradeBand      = "grade_band"
	FairnessDimSection        = "section"
	FairnessDimAccommodation  = "accommodation"
)

// FairnessNotifyDeps wires optional admin/instructor alerts on disparity flags.
type FairnessNotifyDeps struct {
	Pool   *pgxpool.Pool
	Config config.Config
	SSEHub *notifevents.Hub
}

// FairnessCellInput is one raw group aggregate before suppression/flagging.
type FairnessCellInput struct {
	Dimension    string
	GroupLabel   string
	N            int
	MeanFidelity *float64
	CoveragePct  *float64
	MeanLift     *float64
}

// FairnessCellResult is the suppressed, flagged cell written to analytics.
type FairnessCellResult struct {
	Dimension     string
	GroupLabel    string
	N             int
	MeanFidelity  *float32
	CoveragePct   *float32
	MeanLift      *float32
	DisparityFlag bool
}

// SuppressFairnessMean returns nil when n < SmallCellMinN.
func SuppressFairnessMean(n int, mean *float64) *float32 {
	if mean == nil || n < SmallCellMinN {
		return nil
	}
	v := float32(*mean)
	return &v
}

// FlagDisparity reports whether a group's metric is materially worse than the reference mean.
// Only flags when n ≥ FairnessMinN and both means are present; lower-is-worse for fidelity/lift.
func FlagDisparity(n int, groupMean, referenceMean *float64, threshold float64) bool {
	if n < FairnessMinN || groupMean == nil || referenceMean == nil {
		return false
	}
	if threshold <= 0 {
		threshold = FairnessDisparityThreshold
	}
	// Flag when the group is below the reference by more than threshold.
	return (*referenceMean - *groupMean) > threshold
}

// EvaluateFairnessCells applies suppression and disparity flags relative to dimension means.
func EvaluateFairnessCells(cells []FairnessCellInput) []FairnessCellResult {
	// Reference = unweighted mean of uns suppressed group means with n ≥ FairnessMinN.
	type dimRef struct {
		fidSum, liftSum float64
		fidN, liftN     int
	}
	refs := map[string]*dimRef{}
	for _, c := range cells {
		r := refs[c.Dimension]
		if r == nil {
			r = &dimRef{}
			refs[c.Dimension] = r
		}
		if c.N >= FairnessMinN && c.MeanFidelity != nil {
			r.fidSum += *c.MeanFidelity
			r.fidN++
		}
		if c.N >= FairnessMinN && c.MeanLift != nil {
			r.liftSum += *c.MeanLift
			r.liftN++
		}
	}

	out := make([]FairnessCellResult, 0, len(cells))
	for _, c := range cells {
		res := FairnessCellResult{
			Dimension:    c.Dimension,
			GroupLabel:   c.GroupLabel,
			N:            c.N,
			MeanFidelity: SuppressFairnessMean(c.N, c.MeanFidelity),
			CoveragePct:  SuppressFairnessMean(c.N, c.CoveragePct),
			MeanLift:     SuppressFairnessMean(c.N, c.MeanLift),
		}
		r := refs[c.Dimension]
		if r != nil {
			var refFid, refLift *float64
			if r.fidN > 0 {
				v := r.fidSum / float64(r.fidN)
				refFid = &v
			}
			if r.liftN > 0 {
				v := r.liftSum / float64(r.liftN)
				refLift = &v
			}
			// Fidelity is 0–1; lift is percentage points — use 0.10 and 10.0 thresholds.
			if FlagDisparity(c.N, c.MeanFidelity, refFid, FairnessDisparityThreshold) ||
				FlagDisparity(c.N, c.MeanLift, refLift, 10.0) {
				res.DisparityFlag = true
			}
		}
		out = append(out, res)
	}
	return out
}

// RefreshFairnessCourse recomputes fairness aggregates for one course.
func RefreshFairnessCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, notify *FairnessNotifyDeps) (int, error) {
	if pool == nil || courseID == uuid.Nil {
		return 0, nil
	}
	raw, err := acrepo.CollectFairnessRaw(ctx, pool, courseID)
	if err != nil {
		return 0, err
	}
	inputs := make([]FairnessCellInput, 0, len(raw))
	for _, r := range raw {
		inputs = append(inputs, FairnessCellInput{
			Dimension:    r.Dimension,
			GroupLabel:   r.GroupLabel,
			N:            r.N,
			MeanFidelity: r.MeanFidelity,
			CoveragePct:  r.CoveragePct,
			MeanLift:     r.MeanLift,
		})
	}
	results := EvaluateFairnessCells(inputs)
	rows := make([]acrepo.FairnessUpsert, 0, len(results))
	flagged := 0
	for _, res := range results {
		if res.DisparityFlag {
			flagged++
			IncFairnessDisparityFlag()
		}
		rows = append(rows, acrepo.FairnessUpsert{
			CourseID:      courseID,
			Dimension:     res.Dimension,
			GroupLabel:    res.GroupLabel,
			N:             res.N,
			MeanFidelity:  res.MeanFidelity,
			CoveragePct:   res.CoveragePct,
			MeanLift:      res.MeanLift,
			DisparityFlag: res.DisparityFlag,
		})
	}
	if err := acrepo.ReplaceFairnessRows(ctx, pool, courseID, rows); err != nil {
		return 0, err
	}
	if flagged > 0 {
		_ = acrepo.InsertEvent(ctx, pool, courseID, nil, nil, nil, EventFairnessFlag, map[string]any{
			"flaggedCells": flagged,
			"cells":        len(rows),
		})
		notifyFairnessDisparity(ctx, pool, courseID, flagged, notify)
	}
	return len(rows), nil
}

// RefreshFairnessAll runs fairness audit for every ACE-enabled course.
func RefreshFairnessAll(ctx context.Context, pool *pgxpool.Pool, notify *FairnessNotifyDeps) (int, error) {
	if KillSwitchEngaged() {
		return 0, nil
	}
	ids, err := acrepo.ListCourseIDsWithAdaptiveContentEnabled(ctx, pool)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, id := range ids {
		n, err := RefreshFairnessCourse(ctx, pool, id, notify)
		if err != nil {
			slog.Error("adaptivecontent: fairness refresh failed", "course_id", id, "err", err)
			continue
		}
		total += n
	}
	return total, nil
}

func notifyFairnessDisparity(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, flagged int, notify *FairnessNotifyDeps) {
	if notify == nil || notify.Pool == nil || flagged <= 0 {
		return
	}
	instructors, err := atrisk.ListInstructorUserIDs(ctx, pool, courseID)
	if err != nil {
		slog.Error("adaptivecontent: list instructors for fairness alert failed", "err", err)
		return
	}
	code, _ := acrepo.CourseCodeForID(ctx, pool, courseID)
	actionURL := "/settings?tab=ai"
	if code != "" {
		actionURL = "/courses/" + code + "/settings?tab=adaptive-content"
	}
	title := "Adaptive content fairness disparity detected"
	body := "A fairness audit found material differences in adaptation quality across learner groups. Review the oversight console."
	push := &notifications.PushService{Pool: notify.Pool, Config: notify.Config, SSEHub: notify.SSEHub}
	for _, uid := range instructors {
		if err := push.Enqueue(ctx, uid, notifications.EventAdaptiveContentFairness, title, body, actionURL); err != nil {
			slog.Warn("adaptivecontent: fairness notify failed", "user_id", uid, "err", err)
		}
	}
}

// MaxAbsGap is a helper for tests.
func MaxAbsGap(a, b float64) float64 {
	return math.Abs(a - b)
}
