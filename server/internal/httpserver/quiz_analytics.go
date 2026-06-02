package httpserver

import (
	"encoding/json"
	"net/http"

	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
	"github.com/lextures/lextures/server/internal/apierr"
	svcquizanalytics "github.com/lextures/lextures/server/internal/service/quizanalytics"
)

type quizAnalyticsJSON struct {
	QuizID         string                      `json:"quizId"`
	NAttempts      int                         `json:"nAttempts"`
	MeanScore      *float64                    `json:"meanScore"`
	ScoreBuckets   []svcquizanalytics.ScoreBucket  `json:"scoreBuckets"`
	QuestionStats  []svcquizanalytics.QuestionStat `json:"questionStats"`
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(quizAnalyticsJSON{
			QuizID:        report.QuizID.String(),
			NAttempts:     report.NAttempts,
			MeanScore:     report.MeanScore,
			ScoreBuckets:  report.ScoreBuckets,
			QuestionStats: report.QuestionStats,
		})
	}
}
