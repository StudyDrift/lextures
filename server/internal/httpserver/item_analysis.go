package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	svcitemanalysis "github.com/lextures/lextures/server/internal/service/itemanalysis"
)

// itemAnalysisJSON is the API response shape for GET item-analysis.
type itemAnalysisJSON struct {
	QuizID           string              `json:"quizId"`
	InsufficientData bool                `json:"insufficientData,omitempty"`
	NResponses       int                 `json:"nResponses"`
	MinimumRequired  int                 `json:"minimumRequired,omitempty"`
	StatsPending     bool                `json:"statsPending,omitempty"`
	TestStats        *testStatsJSON      `json:"testStats,omitempty"`
	ItemStats        []itemStatJSON      `json:"itemStats,omitempty"`
}

type testStatsJSON struct {
	NResponses    int      `json:"nResponses"`
	KR20          *float64 `json:"kr20"`
	CronbachAlpha *float64 `json:"cronbachAlpha"`
	MeanScore     *float64 `json:"meanScore"`
	StdDev        *float64 `json:"stdDev"`
	ComputedAt    string   `json:"computedAt"`
}

type itemStatJSON struct {
	QuestionIndex   int                `json:"questionIndex"`
	QuestionText    string             `json:"questionText"`
	NResponses      int                `json:"nResponses"`
	PValue          *float64           `json:"pValue"`
	RPb             *float64           `json:"rPb"`
	DistractorFreqs map[string]float64 `json:"distractorFreqs,omitempty"`
	Flag            *string            `json:"flag"`
}

// handleGetItemAnalysis returns pre-computed item stats for a quiz.
// Requires instructor access (course:CODE:item:create permission).
func (d Deps) handleGetItemAnalysis() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		courseCode, viewer, itemID, ok := d.requireItemAnalysisAccess(w, r)
		if !ok {
			return
		}
		_ = courseCode

		ctx := r.Context()

		// Count attempts to check for insufficient data early
		n, err := repoitemanalysis.CountSubmittedAttempts(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to count attempts.")
			return
		}
		_ = viewer

		if n < svcitemanalysis.MinResponses {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(itemAnalysisJSON{
				QuizID:           itemID.String(),
				InsufficientData: true,
				NResponses:       n,
				MinimumRequired:  svcitemanalysis.MinResponses,
			})
			return
		}

		testStat, err := repoitemanalysis.GetTestStats(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load item analysis.")
			return
		}
		if testStat == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(itemAnalysisJSON{
				QuizID:       itemID.String(),
				NResponses:   n,
				StatsPending: true,
			})
			return
		}

		itemRows, err := repoitemanalysis.GetItemStats(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load item stats.")
			return
		}

		out := itemAnalysisJSON{
			QuizID:     itemID.String(),
			NResponses: testStat.NResponses,
			TestStats:  toTestStatsJSON(testStat),
			ItemStats:  toItemStatsJSON(itemRows),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleComputeItemAnalysis manually triggers CTT computation for a quiz.
func (d Deps) handleComputeItemAnalysis() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		courseCode, viewer, itemID, ok := d.requireItemAnalysisAccess(w, r)
		if !ok {
			return
		}
		_ = courseCode
		_ = viewer

		ctx := r.Context()
		result, err := svcitemanalysis.Compute(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute item analysis.")
			return
		}

		if result.N < svcitemanalysis.MinResponses {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(itemAnalysisJSON{
				QuizID:           itemID.String(),
				InsufficientData: true,
				NResponses:       result.N,
				MinimumRequired:  svcitemanalysis.MinResponses,
			})
			return
		}

		out := itemAnalysisJSON{
			QuizID:     itemID.String(),
			NResponses: result.TestStat.NResponses,
			TestStats:  toTestStatsJSON(&result.TestStat),
			ItemStats:  toItemStatsJSON(result.ItemStats),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleExportItemAnalysisCSV downloads item stats as a CSV file.
func (d Deps) handleExportItemAnalysisCSV() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		courseCode, viewer, itemID, ok := d.requireItemAnalysisAccess(w, r)
		if !ok {
			return
		}
		_ = courseCode
		_ = viewer

		ctx := r.Context()
		testStat, err := repoitemanalysis.GetTestStats(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load item analysis.")
			return
		}

		itemRows, err := repoitemanalysis.GetItemStats(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load item stats.")
			return
		}

		if testStat == nil || len(itemRows) == 0 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No item analysis available yet.")
			return
		}

		filename := fmt.Sprintf("item-analysis-%s-%s.csv", itemID.String()[:8], time.Now().UTC().Format("20060102"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

		cw := csv.NewWriter(w)
		_ = cw.Write([]string{
			"question_index", "question_text", "n",
			"p_value", "r_pb",
			"distractor_a_pct", "distractor_b_pct", "distractor_c_pct", "distractor_d_pct",
			"flag",
		})

		for _, row := range itemRows {
			pval := ""
			if row.PValue != nil {
				pval = strconv.FormatFloat(*row.PValue, 'f', 4, 64)
			}
			rpb := ""
			if row.RPB != nil {
				rpb = strconv.FormatFloat(*row.RPB, 'f', 4, 64)
			}
			flag := ""
			if row.Flag != nil {
				flag = *row.Flag
			}
			dA := fmtDistractor(row.DistractorFreqs, "A")
			dB := fmtDistractor(row.DistractorFreqs, "B")
			dC := fmtDistractor(row.DistractorFreqs, "C")
			dD := fmtDistractor(row.DistractorFreqs, "D")
			_ = cw.Write([]string{
				strconv.Itoa(row.QuestionIndex),
				row.QuestionText,
				strconv.Itoa(row.NResponses),
				pval, rpb,
				dA, dB, dC, dD,
				flag,
			})
		}
		cw.Flush()
	}
}

// requireItemAnalysisAccess checks authentication and instructor permission, returning
// (courseCode, viewerID, itemID, ok). Writes an error and returns ok=false on failure.
func (d Deps) requireItemAnalysisAccess(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, uuid.UUID, bool) {
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, uuid.UUID{}, false
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}

	ctx := r.Context()
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	if cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}

	perm := "course:" + courseCode + ":item:create"
	canEdit, err := rbac.UserHasPermission(ctx, d.Pool, viewer, perm)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return "", uuid.UUID{}, uuid.UUID{}, false
	}

	return courseCode, viewer, itemID, true
}

func toTestStatsJSON(r *repoitemanalysis.TestStatRow) *testStatsJSON {
	if r == nil {
		return nil
	}
	return &testStatsJSON{
		NResponses:    r.NResponses,
		KR20:          r.KR20,
		CronbachAlpha: r.CronbachAlpha,
		MeanScore:     r.MeanScore,
		StdDev:        r.StdDev,
		ComputedAt:    r.ComputedAt.Format(time.RFC3339),
	}
}

func toItemStatsJSON(rows []repoitemanalysis.ItemStatRow) []itemStatJSON {
	sort.Slice(rows, func(i, j int) bool { return rows[i].QuestionIndex < rows[j].QuestionIndex })
	out := make([]itemStatJSON, len(rows))
	for i, r := range rows {
		out[i] = itemStatJSON{
			QuestionIndex:   r.QuestionIndex,
			QuestionText:    r.QuestionText,
			NResponses:      r.NResponses,
			PValue:          r.PValue,
			RPb:             r.RPB,
			DistractorFreqs: r.DistractorFreqs,
			Flag:            r.Flag,
		}
	}
	return out
}

func fmtDistractor(freqs map[string]float64, key string) string {
	if freqs == nil {
		return ""
	}
	if v, ok := freqs[key]; ok {
		return strconv.FormatFloat(v*100, 'f', 1, 64) + "%"
	}
	return ""
}
