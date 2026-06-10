package httpserver

import (
	"encoding/json"
	"net/http"

	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	svcquizanalytics "github.com/lextures/lextures/server/internal/service/quizanalytics"
)

type quizAnalyticsJSON struct {
	QuizID         string                            `json:"quizId"`
	NAttempts      int                               `json:"nAttempts"`
	MeanScore      *float64                          `json:"meanScore"`
	ScoreBuckets   []svcquizanalytics.ScoreBucket    `json:"scoreBuckets"`
	QuestionStats  []svcquizanalytics.QuestionStat `json:"questionStats"`
	FocusAttempts  []svcquizanalytics.AttemptFocusStat `json:"focusAttempts"`
}

func (d Deps) handleGetQuizAnalytics() http.HandlerFunc {
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

		_, _, itemID, ok := d.requireItemAnalysisAccess(w, r)
		if !ok {
			return
		}

		ctx := r.Context()
		rows, err := repoitemanalysis.FetchAttemptResponses(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz analytics.")
			return
		}

		report := svcquizanalytics.BuildReport(itemID, rows)
		focusRows, err := quizattempts.ListAttemptFocusSummariesForItem(ctx, d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load focus-loss stats.")
			return
		}
		focusAttempts := make([]svcquizanalytics.AttemptFocusStat, 0, len(focusRows))
		for _, f := range focusRows {
			focusAttempts = append(focusAttempts, svcquizanalytics.AttemptFocusStat{
				AttemptID:             f.AttemptID.String(),
				AttemptNumber:         f.AttemptNumber,
				EventCount:            f.EventCount,
				AcademicIntegrityFlag: f.AcademicIntegrityFlag,
			})
		}
		report.FocusAttempts = focusAttempts
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(quizAnalyticsJSON{
			QuizID:        report.QuizID.String(),
			NAttempts:     report.NAttempts,
			MeanScore:     report.MeanScore,
			ScoreBuckets:  report.ScoreBuckets,
			QuestionStats: report.QuestionStats,
			FocusAttempts: report.FocusAttempts,
		})
	}
}
