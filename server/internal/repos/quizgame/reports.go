package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/itemanalysis"
)

var (
	ErrReportNotFound = errors.New("quizgame: report not found")
	ErrGameNotEnded   = errors.New("quizgame: game not ended")
)

// QuestionAggregate is one per-question row in a game report.
type QuestionAggregate struct {
	Index            int                `json:"index"`
	Prompt           string             `json:"prompt"`
	CorrectPct       float64            `json:"correctPct"`
	AvgMs            float64            `json:"avgMs"`
	AnswerCount      int                `json:"answerCount"`
	Distribution     map[string]int     `json:"distribution"`
	SourceQuestionID *string            `json:"sourceQuestionId,omitempty"`
	HardestRank      int                `json:"hardestRank,omitempty"` // 1 = hardest
}

// GameReport is the cached post-game summary (FR-1).
type GameReport struct {
	SessionID     string              `json:"sessionId"`
	PlayerCount   int                 `json:"playerCount"`
	AnsweredCount int                 `json:"answeredCount"`
	ScoreAvg      *float64            `json:"scoreAvg"`
	ScoreMedian   *float64            `json:"scoreMedian"`
	ScoreMax      *int                `json:"scoreMax"`
	PerQuestion   []QuestionAggregate `json:"perQuestion"`
	GeneratedAt   time.Time           `json:"generatedAt"`
}

// PlayerResultRow is one student in the instructor report table.
type PlayerResultRow struct {
	PlayerID   string  `json:"playerId"`
	Nickname   string  `json:"nickname"`
	UserID     *string `json:"userId,omitempty"`
	IsGuest    bool    `json:"isGuest"`
	TotalScore int     `json:"totalScore"`
	Rank       int     `json:"rank"`
	Answered   int     `json:"answered"`
	Correct    int     `json:"correct"`
}

// ReviewItem is one question a learner should review (incorrect or slow).
type ReviewItem struct {
	Index            int             `json:"index"`
	Prompt           string          `json:"prompt"`
	Explanation      *string         `json:"explanation,omitempty"`
	IsCorrect        bool            `json:"isCorrect"`
	Points           int             `json:"points"`
	ResponseMs       int             `json:"responseMs"`
	Answer           json.RawMessage `json:"answer"`
	CorrectOptionIDs []string        `json:"correctOptionIds,omitempty"`
	CorrectAnswer    map[string]any  `json:"correctAnswer,omitempty"`
	Reason           string          `json:"reason"` // incorrect | slow
}

// MyResults is the self-scoped student results payload (FR-3).
type MyResults struct {
	SessionID   string       `json:"sessionId"`
	Nickname    string       `json:"nickname"`
	TotalScore  int          `json:"totalScore"`
	Rank        int          `json:"rank"`
	PlayerCount int          `json:"playerCount"`
	Answered    int          `json:"answered"`
	Correct     int          `json:"correct"`
	ReviewThese []ReviewItem `json:"reviewThese"`
}

// ListAllResponses returns every response for a session (report rebuild feed).
func ListAllResponses(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]Response, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT session_id, question_index, player_id, answer, is_correct, response_ms, points, points_breakdown, answered_at
		FROM quizgame.session_responses
		WHERE session_id = $1
		ORDER BY question_index ASC, answered_at ASC`, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Response
	for rows.Next() {
		var r Response
		var sidU, pid uuid.UUID
		var answer, breakdown []byte
		if err := rows.Scan(&sidU, &r.QuestionIndex, &pid, &answer, &r.IsCorrect, &r.ResponseMs, &r.Points, &breakdown, &r.AnsweredAt); err != nil {
			return nil, err
		}
		r.SessionID = sidU.String()
		r.PlayerID = pid.String()
		r.Answer = answer
		r.PointsBreakdown = breakdown
		out = append(out, r)
	}
	return out, rows.Err()
}

// ComputeReportAggregates builds a report from raw players + responses (FR-11, pure).
func ComputeReportAggregates(sess *Session, players []Player, responses []Response) GameReport {
	scores := make([]int, 0, len(players))
	answeredPlayers := map[string]struct{}{}
	for _, p := range players {
		scores = append(scores, p.TotalScore)
	}
	for _, r := range responses {
		answeredPlayers[r.PlayerID] = struct{}{}
	}

	avg, med, mx := scoreStats(scores)
	qCount := 0
	if sess != nil {
		qCount = len(sess.KitSnapshot.Questions)
	}
	perQ := make([]QuestionAggregate, qCount)
	for i := 0; i < qCount; i++ {
		q := sess.KitSnapshot.Questions[i]
		agg := QuestionAggregate{
			Index:        i,
			Prompt:       q.Prompt,
			Distribution: map[string]int{},
		}
		if q.SourceQuestionID != nil && *q.SourceQuestionID != "" {
			sid := *q.SourceQuestionID
			agg.SourceQuestionID = &sid
		}
		perQ[i] = agg
	}

	type qAcc struct {
		n, correct, msSum int
	}
	acc := make([]qAcc, qCount)
	for _, r := range responses {
		if r.QuestionIndex < 0 || r.QuestionIndex >= qCount {
			continue
		}
		a := &acc[r.QuestionIndex]
		a.n++
		a.msSum += r.ResponseMs
		if r.IsCorrect {
			a.correct++
		}
		key := answerDistributionKey(r.Answer)
		perQ[r.QuestionIndex].Distribution[key]++
	}
	for i := range perQ {
		a := acc[i]
		perQ[i].AnswerCount = a.n
		if a.n > 0 {
			perQ[i].CorrectPct = math.Round(float64(a.correct)/float64(a.n)*10000) / 100
			perQ[i].AvgMs = math.Round(float64(a.msSum)/float64(a.n)*100) / 100
		}
	}
	rankHardest(perQ)

	rep := GameReport{
		SessionID:     sess.ID,
		PlayerCount:   len(players),
		AnsweredCount: len(answeredPlayers),
		PerQuestion:   perQ,
		GeneratedAt:   time.Now().UTC(),
	}
	if len(scores) > 0 {
		rep.ScoreAvg = &avg
		rep.ScoreMedian = &med
		rep.ScoreMax = &mx
	}
	return rep
}

func scoreStats(scores []int) (avg, median float64, max int) {
	if len(scores) == 0 {
		return 0, 0, 0
	}
	sum := 0
	max = scores[0]
	sorted := append([]int(nil), scores...)
	sort.Ints(sorted)
	for _, s := range scores {
		sum += s
		if s > max {
			max = s
		}
	}
	avg = math.Round(float64(sum)/float64(len(scores))*100) / 100
	n := len(sorted)
	if n%2 == 1 {
		median = float64(sorted[n/2])
	} else {
		median = math.Round((float64(sorted[n/2-1])+float64(sorted[n/2]))/2*100) / 100
	}
	return avg, median, max
}

func rankHardest(perQ []QuestionAggregate) {
	type pair struct {
		i   int
		pct float64
		n   int
	}
	var ranked []pair
	for i, q := range perQ {
		if q.AnswerCount == 0 {
			continue
		}
		ranked = append(ranked, pair{i: i, pct: q.CorrectPct, n: q.AnswerCount})
	}
	sort.SliceStable(ranked, func(a, b int) bool {
		if ranked[a].pct != ranked[b].pct {
			return ranked[a].pct < ranked[b].pct
		}
		return ranked[a].i < ranked[b].i
	})
	for rank, p := range ranked {
		perQ[p.i].HardestRank = rank + 1
	}
}

func answerDistributionKey(answer json.RawMessage) string {
	if len(answer) == 0 {
		return "(empty)"
	}
	var m map[string]any
	if err := json.Unmarshal(answer, &m); err != nil {
		return string(answer)
	}
	if ids, ok := m["selectedOptionIds"].([]any); ok {
		parts := make([]string, 0, len(ids))
		for _, id := range ids {
			parts = append(parts, fmt.Sprint(id))
		}
		sort.Strings(parts)
		b, _ := json.Marshal(parts)
		return string(b)
	}
	if id, ok := m["optionId"]; ok {
		return fmt.Sprint(id)
	}
	if ids, ok := m["optionIds"].([]any); ok {
		parts := make([]string, 0, len(ids))
		for _, id := range ids {
			parts = append(parts, fmt.Sprint(id))
		}
		sort.Strings(parts)
		b, _ := json.Marshal(parts)
		return string(b)
	}
	if v, ok := m["value"]; ok {
		return fmt.Sprint(v)
	}
	if v, ok := m["text"]; ok {
		return fmt.Sprint(v)
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// BuildAndStoreReport computes and upserts the game report (FR-1 / FR-11).
func BuildAndStoreReport(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*GameReport, error) {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status != "ended" && sess.Status != "abandoned" {
		return nil, ErrGameNotEnded
	}
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	responses, err := ListAllResponses(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	rep := ComputeReportAggregates(sess, players, responses)
	if err := upsertGameReport(ctx, pool, &rep); err != nil {
		return nil, err
	}
	_ = contributeItemAnalysis(ctx, pool, sess, &rep, responses)
	return &rep, nil
}

func upsertGameReport(ctx context.Context, pool *pgxpool.Pool, rep *GameReport) error {
	sid, err := uuid.Parse(rep.SessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	pq, err := json.Marshal(rep.PerQuestion)
	if err != nil {
		return err
	}
	var generated time.Time
	err = pool.QueryRow(ctx, `
		INSERT INTO quizgame.game_reports (
			session_id, player_count, answered_count, score_avg, score_median, score_max, per_question, generated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,NOW())
		ON CONFLICT (session_id) DO UPDATE SET
			player_count = EXCLUDED.player_count,
			answered_count = EXCLUDED.answered_count,
			score_avg = EXCLUDED.score_avg,
			score_median = EXCLUDED.score_median,
			score_max = EXCLUDED.score_max,
			per_question = EXCLUDED.per_question,
			generated_at = NOW()
		RETURNING generated_at`,
		sid, rep.PlayerCount, rep.AnsweredCount, rep.ScoreAvg, rep.ScoreMedian, rep.ScoreMax, pq,
	).Scan(&generated)
	if err != nil {
		return err
	}
	rep.GeneratedAt = generated
	return nil
}

// GetGameReport loads the cached report, or nil if missing.
func GetGameReport(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*GameReport, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	var rep GameReport
	var avg, med *float64
	var mx *int
	var pq []byte
	err = pool.QueryRow(ctx, `
		SELECT session_id, player_count, answered_count, score_avg, score_median, score_max, per_question, generated_at
		FROM quizgame.game_reports WHERE session_id = $1`, sid,
	).Scan(&sid, &rep.PlayerCount, &rep.AnsweredCount, &avg, &med, &mx, &pq, &rep.GeneratedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rep.SessionID = sid.String()
	rep.ScoreAvg = avg
	rep.ScoreMedian = med
	rep.ScoreMax = mx
	if len(pq) > 0 {
		_ = json.Unmarshal(pq, &rep.PerQuestion)
	}
	return &rep, nil
}

// EnsureGameReport returns the cached report or builds it.
func EnsureGameReport(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*GameReport, error) {
	rep, err := GetGameReport(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	if rep != nil {
		return rep, nil
	}
	return BuildAndStoreReport(ctx, pool, sessionID)
}

// ReportsMatch checks recomputation equality (AC-8).
func ReportsMatch(a, b *GameReport) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.PlayerCount != b.PlayerCount || a.AnsweredCount != b.AnsweredCount {
		return false
	}
	if !floatPtrEq(a.ScoreAvg, b.ScoreAvg) || !floatPtrEq(a.ScoreMedian, b.ScoreMedian) {
		return false
	}
	if !intPtrEq(a.ScoreMax, b.ScoreMax) {
		return false
	}
	if len(a.PerQuestion) != len(b.PerQuestion) {
		return false
	}
	for i := range a.PerQuestion {
		qa, qb := a.PerQuestion[i], b.PerQuestion[i]
		if qa.Index != qb.Index || qa.AnswerCount != qb.AnswerCount {
			return false
		}
		if qa.CorrectPct != qb.CorrectPct || qa.AvgMs != qb.AvgMs {
			return false
		}
	}
	return true
}

func floatPtrEq(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func intPtrEq(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// BuildPlayerResults builds the instructor per-student table + ranks.
func BuildPlayerResults(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]PlayerResultRow, error) {
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	lb, err := ComputeLeaderboard(ctx, pool, sessionID, len(players)+1)
	if err != nil {
		return nil, err
	}
	rankBy := map[string]int{}
	for _, e := range lb {
		rankBy[e.PlayerID] = e.Rank
	}
	responses, err := ListAllResponses(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	ansBy := map[string]int{}
	corBy := map[string]int{}
	for _, r := range responses {
		ansBy[r.PlayerID]++
		if r.IsCorrect {
			corBy[r.PlayerID]++
		}
	}
	out := make([]PlayerResultRow, 0, len(players))
	for _, p := range players {
		row := PlayerResultRow{
			PlayerID:   p.ID,
			Nickname:   p.Nickname,
			UserID:     p.UserID,
			IsGuest:    p.UserID == nil,
			TotalScore: p.TotalScore,
			Rank:       rankBy[p.ID],
			Answered:   ansBy[p.ID],
			Correct:    corBy[p.ID],
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			if out[i].Rank == 0 {
				return false
			}
			if out[j].Rank == 0 {
				return true
			}
			return out[i].Rank < out[j].Rank
		}
		return out[i].Nickname < out[j].Nickname
	})
	return out, nil
}

// BuildMyResults returns self-scoped results for an enrolled player (FR-3).
func BuildMyResults(ctx context.Context, pool *pgxpool.Pool, sessionID string, userID uuid.UUID) (*MyResults, error) {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	player, err := GetPlayerByUser(ctx, pool, sessionID, userID)
	if err != nil {
		return nil, err
	}
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	rank, err := PlayerRank(ctx, pool, sessionID, player.ID)
	if err != nil {
		return nil, err
	}
	responses, err := ListPlayerResponses(ctx, pool, sessionID, player.ID)
	if err != nil {
		return nil, err
	}
	correct := 0
	for _, r := range responses {
		if r.IsCorrect {
			correct++
		}
	}
	review := buildReviewItems(sess, responses)
	return &MyResults{
		SessionID:   sessionID,
		Nickname:    player.Nickname,
		TotalScore:  player.TotalScore,
		Rank:        rank,
		PlayerCount: len(players),
		Answered:    len(responses),
		Correct:     correct,
		ReviewThese: review,
	}, nil
}

func buildReviewItems(sess *Session, responses []Response) []ReviewItem {
	if sess == nil {
		return nil
	}
	// Slow threshold: slower than 75th percentile of this player's times, or > 70% of time limit.
	msVals := make([]int, 0, len(responses))
	for _, r := range responses {
		msVals = append(msVals, r.ResponseMs)
	}
	sort.Ints(msVals)
	slowCut := 0
	if n := len(msVals); n > 0 {
		slowCut = msVals[(n*3)/4]
	}

	var out []ReviewItem
	for _, r := range responses {
		if r.QuestionIndex < 0 || r.QuestionIndex >= len(sess.KitSnapshot.Questions) {
			continue
		}
		q := sess.KitSnapshot.Questions[r.QuestionIndex]
		limitMs := q.TimeLimitSeconds * 1000
		slow := false
		if limitMs > 0 && r.ResponseMs >= (limitMs*70)/100 {
			slow = true
		}
		if slowCut > 0 && r.ResponseMs >= slowCut && r.ResponseMs > 0 {
			slow = true
		}
		if r.IsCorrect && !slow {
			continue
		}
		reason := "incorrect"
		if r.IsCorrect {
			reason = "slow"
		}
		item := ReviewItem{
			Index:         r.QuestionIndex,
			Prompt:        q.Prompt,
			Explanation:   q.Explanation,
			IsCorrect:     r.IsCorrect,
			Points:        r.Points,
			ResponseMs:    r.ResponseMs,
			Answer:        r.Answer,
			CorrectAnswer: q.CorrectAnswer,
			Reason:        reason,
		}
		for _, o := range q.Options {
			if o.IsCorrect {
				item.CorrectOptionIDs = append(item.CorrectOptionIDs, o.ID)
			}
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// contributeItemAnalysis writes tagged CTT-ish stats for bank-linked questions (FR-2).
func contributeItemAnalysis(ctx context.Context, pool *pgxpool.Pool, sess *Session, rep *GameReport, responses []Response) error {
	if sess == nil || rep == nil {
		return nil
	}
	now := time.Now().UTC()
	var items []itemanalysis.ItemStatRow
	// Game-local stats keyed by session id.
	for _, q := range rep.PerQuestion {
		if q.AnswerCount == 0 {
			continue
		}
		pv := q.CorrectPct / 100
		flag := difficultyFlag(pv)
		prompt := q.Prompt
		if q.SourceQuestionID != nil {
			prompt = "[interactive_quiz] " + prompt
		}
		sid, err := uuid.Parse(sess.ID)
		if err != nil {
			continue
		}
		items = append(items, itemanalysis.ItemStatRow{
			QuizID:          sid,
			QuestionIndex:   q.Index,
			QuestionText:    prompt,
			NResponses:      q.AnswerCount,
			PValue:          &pv,
			DistractorFreqs: distToFreq(q.Distribution, q.AnswerCount),
			Flag:            flag,
			ComputedAt:      now,
		})
		// Also contribute under bank question id when linked (tagged, additive).
		if q.SourceQuestionID != nil {
			bq, err := uuid.Parse(*q.SourceQuestionID)
			if err != nil {
				continue
			}
			items = append(items, itemanalysis.ItemStatRow{
				QuizID:          bq,
				QuestionIndex:   0,
				QuestionText:    "[interactive_quiz] " + q.Prompt,
				NResponses:      q.AnswerCount,
				PValue:          &pv,
				DistractorFreqs: distToFreq(q.Distribution, q.AnswerCount),
				Flag:            flag,
				ComputedAt:      now,
			})
		}
	}
	_ = responses
	if len(items) == 0 {
		return nil
	}
	return itemanalysis.InsertItemStats(ctx, pool, items)
}

func difficultyFlag(pValue float64) *string {
	var f string
	switch {
	case pValue >= 0.9:
		f = "easy"
	case pValue <= 0.3:
		f = "hard"
	default:
		return nil
	}
	return &f
}

func distToFreq(dist map[string]int, n int) map[string]float64 {
	if n <= 0 || len(dist) == 0 {
		return nil
	}
	out := make(map[string]float64, len(dist))
	for k, v := range dist {
		out[k] = float64(v) / float64(n)
	}
	return out
}
