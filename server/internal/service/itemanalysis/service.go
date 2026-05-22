// Package itemanalysis computes classical test theory (CTT) statistics for quiz items.
package itemanalysis

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
)

const (
	MinResponses  = 10
	FlagHard      = "hard"
	FlagEasy      = "easy"
	FlagPoorDisc  = "poor_discriminator"
	hardThreshold = 0.20
	easyThreshold = 0.90
	discThreshold = 0.15
)

// Result holds all computed statistics for one quiz administration.
type Result struct {
	QuizID    uuid.UUID
	N         int
	ItemStats []repoitemanalysis.ItemStatRow
	TestStat  repoitemanalysis.TestStatRow
}

// Compute fetches raw response data, computes CTT stats, persists them, and returns the result.
// Returns a Result with N < MinResponses and empty stats when insufficient data exists.
func Compute(ctx context.Context, pool *pgxpool.Pool, structureItemID uuid.UUID) (*Result, error) {
	now := time.Now().UTC()

	rawRows, err := repoitemanalysis.FetchAttemptResponses(ctx, pool, structureItemID)
	if err != nil {
		return nil, fmt.Errorf("fetch responses: %w", err)
	}

	itemStats, testStat := computeStats(structureItemID, rawRows, now)
	result := &Result{
		QuizID:    structureItemID,
		N:         testStat.NResponses,
		ItemStats: itemStats,
		TestStat:  testStat,
	}

	if testStat.NResponses < MinResponses {
		return result, nil
	}

	if err := repoitemanalysis.UpsertTestStats(ctx, pool, testStat); err != nil {
		return nil, fmt.Errorf("upsert test stats: %w", err)
	}
	if len(itemStats) > 0 {
		if err := repoitemanalysis.InsertItemStats(ctx, pool, itemStats); err != nil {
			return nil, fmt.Errorf("insert item stats: %w", err)
		}
	}

	return result, nil
}

// computeStats is the pure computation layer — no I/O, easily unit-tested.
//
// Two total-score scales are maintained:
//   - totalScorePct  (0–100): from a.score_percent; used for r_pb, mean_score, std_dev display.
//   - totalScoreRaw  (0–k):  sum of per-item proportions (PointsAwarded/MaxPoints);
//     used for KR-20 / Cronbach α variance so the formula is dimensionally consistent
//     with the per-item p·q and item-variance terms (both on 0–1 item scale).
func computeStats(quizID uuid.UUID, rows []repoitemanalysis.AttemptResponseRow, now time.Time) ([]repoitemanalysis.ItemStatRow, repoitemanalysis.TestStatRow) {
	type studentAttempt struct {
		totalScorePct float64
		totalScoreRaw float64
		responses     map[int]repoitemanalysis.AttemptResponseRow
	}

	attempts := map[uuid.UUID]*studentAttempt{}
	for _, r := range rows {
		sa, ok := attempts[r.AttemptID]
		if !ok {
			sa = &studentAttempt{
				totalScorePct: attemptScorePct(r),
				responses:     map[int]repoitemanalysis.AttemptResponseRow{},
			}
			attempts[r.AttemptID] = sa
		}
		sa.responses[r.QuestionIndex] = r
	}

	// Compute raw total scores (item-unit scale) after all responses are grouped.
	for _, sa := range attempts {
		var raw float64
		for _, r := range sa.responses {
			if r.MaxPoints > 0 {
				raw += r.PointsAwarded / r.MaxPoints
			} else if r.IsCorrect != nil && *r.IsCorrect {
				raw += 1
			}
		}
		sa.totalScoreRaw = raw
	}

	n := len(attempts)
	empty := repoitemanalysis.TestStatRow{QuizID: quizID, NResponses: n, ComputedAt: now}
	if n < MinResponses {
		return nil, empty
	}

	// Collect total-score slices and unique question indices.
	totalScoresPct := make([]float64, 0, n)
	totalScoresRaw := make([]float64, 0, n)
	questionTypes := map[int]string{}
	questionTexts := map[int]string{}
	for _, sa := range attempts {
		totalScoresPct = append(totalScoresPct, sa.totalScorePct)
		totalScoresRaw = append(totalScoresRaw, sa.totalScoreRaw)
		for qi, r := range sa.responses {
			questionTypes[qi] = r.QuestionType
			if r.PromptText != nil && questionTexts[qi] == "" {
				questionTexts[qi] = *r.PromptText
			}
		}
	}

	meanPct := mean(totalScoresPct)
	stdPct := math.Sqrt(populationVariance(totalScoresPct, meanPct))

	meanRaw := mean(totalScoresRaw)
	varRaw := populationVariance(totalScoresRaw, meanRaw)

	// Per-item statistics.
	var itemStats []repoitemanalysis.ItemStatRow
	for qi, qtype := range questionTypes {
		type obs struct {
			isCorrect   bool
			choiceIndex *int
			itemFrac    float64 // PointsAwarded/MaxPoints (0–1)
			totalPct    float64 // for r_pb
		}

		var observations []obs
		for _, sa := range attempts {
			r, ok := sa.responses[qi]
			if !ok {
				continue
			}
			correct := r.IsCorrect != nil && *r.IsCorrect
			var frac float64
			if r.MaxPoints > 0 {
				frac = r.PointsAwarded / r.MaxPoints
			} else if correct {
				frac = 1
			}
			observations = append(observations, obs{
				isCorrect:   correct,
				choiceIndex: r.ChoiceIndex,
				itemFrac:    frac,
				totalPct:    attempts[r.AttemptID].totalScorePct,
			})
		}
		if len(observations) == 0 {
			continue
		}

		// p-value
		correctCount := 0
		for _, o := range observations {
			if o.isCorrect {
				correctCount++
			}
		}
		pval := float64(correctCount) / float64(len(observations))

		// point-biserial r_pb
		var rpbPtr *float64
		if stdPct > 0 && pval > 0 && pval < 1 {
			var sumC, sumW float64
			var nC, nW int
			for _, o := range observations {
				if o.isCorrect {
					sumC += o.totalPct
					nC++
				} else {
					sumW += o.totalPct
					nW++
				}
			}
			if nC > 0 && nW > 0 {
				meanC := sumC / float64(nC)
				meanW := sumW / float64(nW)
				rpb := ((meanC - meanW) / stdPct) * math.Sqrt(pval*(1-pval))
				rpbPtr = &rpb
			}
		}

		// Distractor frequencies (MC / true_false only).
		var distractorFreqs map[string]float64
		if qtype == "multiple_choice" || qtype == "true_false" {
			counts := map[int]int{}
			total := 0
			for _, o := range observations {
				if o.choiceIndex != nil {
					counts[*o.choiceIndex]++
					total++
				}
			}
			if total > 0 {
				labels := []string{"A", "B", "C", "D", "E", "F"}
				distractorFreqs = make(map[string]float64, len(counts))
				for idx, cnt := range counts {
					label := fmt.Sprintf("%d", idx)
					if idx >= 0 && idx < len(labels) {
						label = labels[idx]
					}
					distractorFreqs[label] = float64(cnt) / float64(total)
				}
			}
		}

		// Flag
		var flagPtr *string
		switch {
		case pval < hardThreshold:
			f := FlagHard
			flagPtr = &f
		case pval > easyThreshold:
			f := FlagEasy
			flagPtr = &f
		case rpbPtr != nil && *rpbPtr < discThreshold:
			f := FlagPoorDisc
			flagPtr = &f
		}

		pvalCopy := pval
		itemStats = append(itemStats, repoitemanalysis.ItemStatRow{
			QuizID:          quizID,
			QuestionIndex:   qi,
			QuestionText:    questionTexts[qi],
			NResponses:      len(observations),
			PValue:          &pvalCopy,
			RPB:             rpbPtr,
			DistractorFreqs: distractorFreqs,
			Flag:            flagPtr,
			ComputedAt:      now,
		})
	}

	// Test-level statistics.
	testStat := repoitemanalysis.TestStatRow{
		QuizID:     quizID,
		NResponses: n,
		ComputedAt: now,
		MeanScore:  &meanPct,
	}
	stdDev := stdPct
	testStat.StdDev = &stdDev

	k := len(questionTypes)
	if k > 1 && varRaw > 0 {
		allDichotomous := true
		for _, qt := range questionTypes {
			if qt != "multiple_choice" && qt != "true_false" {
				allDichotomous = false
				break
			}
		}

		if allDichotomous {
			// KR-20: Var_T is variance of raw item-unit total (0..k scale),
			// matching the p·q item scale.
			var sumPQ float64
			for _, stat := range itemStats {
				if stat.PValue != nil {
					p := *stat.PValue
					sumPQ += p * (1 - p)
				}
			}
			kr20 := float64(k) / float64(k-1) * (1 - sumPQ/varRaw)
			testStat.KR20 = &kr20
		} else {
			// Cronbach's α: sum of item variances (each on 0–1 scale).
			var sumItemVar float64
			for qi := range questionTypes {
				var scores []float64
				for _, sa := range attempts {
					r, ok := sa.responses[qi]
					if !ok {
						continue
					}
					var s float64
					if r.MaxPoints > 0 {
						s = r.PointsAwarded / r.MaxPoints
					} else if r.IsCorrect != nil && *r.IsCorrect {
						s = 1
					}
					scores = append(scores, s)
				}
				sumItemVar += populationVariance(scores, mean(scores))
			}
			alpha := float64(k) / float64(k-1) * (1 - sumItemVar/varRaw)
			testStat.CronbachAlpha = &alpha
		}
	}

	return itemStats, testStat
}

// attemptScorePct returns score_percent for an attempt (from the first response row
// where a.score_percent is the same for all responses sharing an attempt).
func attemptScorePct(r repoitemanalysis.AttemptResponseRow) float64 {
	if r.ScorePercent != nil {
		return *r.ScorePercent
	}
	if r.MaxPoints > 0 {
		return r.PointsAwarded / r.MaxPoints * 100
	}
	return 0
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func populationVariance(xs []float64, m float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		d := x - m
		s += d * d
	}
	return s / float64(len(xs))
}
