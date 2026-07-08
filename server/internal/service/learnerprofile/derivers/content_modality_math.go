package derivers

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	contentModalityWindowDays        = 90
	contentModalityMinDistinctItems  = 3
	contentModalityMinModalities     = 2
	contentModalityDeriverVersion    = 1
	contentModalitySourceTable       = "analytics.engagement_events"
	contentModalityReadingSourceTable = "course.module_content_pages"
	contentModalityQuizSourceTable   = "course.quiz_attempts"
	contentModalityHeartbeatSec      = 30
	contentModalityReadingWPM        = 200
	contentModalityMinExpectedReadSec = 60.0
	contentModalityDefaultQuizSec    = 600.0
	contentModalityDefaultInteractiveSec = 900.0
	contentModalityThoroughThreshold = 0.75
	contentModalitySkimThreshold     = 0.45
	contentModalityComfortThreshold  = 0.55
)

type modalityKind string

const (
	modalityVideo       modalityKind = "video"
	modalityReading     modalityKind = "reading"
	modalityInteractive modalityKind = "interactive"
	modalityQuiz        modalityKind = "quiz"
)

// ModalitySummary is the facet-level aggregate returned in summary JSON.
type ModalitySummary struct {
	ModalityAffinity  map[string]float64    `json:"modalityAffinity"`
	ComplexityComfort *ComplexityComfortBand  `json:"complexityComfort,omitempty"`
	Pacing            string                  `json:"pacing"`
}

// ComplexityComfortBand is the reading-level band where engagement stays high.
type ComplexityComfortBand struct {
	Low        string  `json:"low"`
	High       string  `json:"high"`
	Confidence float64 `json:"confidence,omitempty"`
}

type itemEngagement struct {
	ItemKey            string
	Modality           modalityKind
	CourseKey          string
	MaxVideoPct        float64
	MaxScrollDepth     float64
	TimeOnTaskSec      int
	QuizCompleted      bool
	ReadingLevelFKGL   *float64
	ExpectedDurationSec float64
}

type modalityComputeInput struct {
	Items       []itemEngagement
	WindowStart int // unused in math but kept for symmetry with other derivers
}

func mapItemTypeToModality(itemType string) modalityKind {
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "video":
		return modalityVideo
	case "content_page":
		return modalityReading
	case "quiz":
		return modalityQuiz
	case "h5p", "vibe_activity", "scorm", "lti_link", "activity":
		return modalityInteractive
	default:
		return ""
	}
}

func estimateReadSeconds(wordCount int) float64 {
	if wordCount <= 0 {
		return contentModalityMinExpectedReadSec
	}
	seconds := float64(wordCount) / float64(contentModalityReadingWPM) * 60.0
	if seconds < contentModalityMinExpectedReadSec {
		return contentModalityMinExpectedReadSec
	}
	return seconds
}

func countWords(markdown string) int {
	fields := strings.Fields(markdown)
	return len(fields)
}

func pacingRatio(timeOnTaskSec int, expectedSec float64) float64 {
	if expectedSec <= 0 || timeOnTaskSec <= 0 {
		return 0
	}
	return round2(clampFloat(float64(timeOnTaskSec)/expectedSec, 0, 1))
}

func (item itemEngagement) engagementScore() float64 {
	switch item.Modality {
	case modalityVideo:
		return round2(clampFloat(item.MaxVideoPct/100.0, 0, 1))
	case modalityReading:
		scroll := clampFloat(item.MaxScrollDepth/100.0, 0, 1)
		timeRatio := pacingRatio(item.TimeOnTaskSec, item.ExpectedDurationSec)
		switch {
		case item.MaxScrollDepth > 0 && item.TimeOnTaskSec > 0:
			return round2((scroll + timeRatio) / 2)
		case item.MaxScrollDepth > 0:
			return round2(scroll)
		case item.TimeOnTaskSec > 0:
			return round2(timeRatio)
		default:
			return 0
		}
	case modalityQuiz:
		if item.QuizCompleted {
			return 1
		}
		return pacingRatio(item.TimeOnTaskSec, item.ExpectedDurationSec)
	case modalityInteractive:
		scroll := clampFloat(item.MaxScrollDepth/100.0, 0, 1)
		timeRatio := pacingRatio(item.TimeOnTaskSec, item.ExpectedDurationSec)
		switch {
		case scroll > 0 && timeRatio > 0:
			return round2((scroll + timeRatio) / 2)
		case scroll > 0:
			return round2(scroll)
		default:
			return round2(timeRatio)
		}
	default:
		return 0
	}
}

func (item itemEngagement) pacingScore() float64 {
	switch item.Modality {
	case modalityVideo:
		return clampFloat(item.MaxVideoPct/100.0, 0, 1)
	case modalityReading:
		timeRatio := pacingRatio(item.TimeOnTaskSec, item.ExpectedDurationSec)
		scroll := clampFloat(item.MaxScrollDepth/100.0, 0, 1)
		if scroll > 0 && timeRatio > 0 {
			return round2((scroll + timeRatio) / 2)
		}
		if scroll > 0 {
			return scroll
		}
		return timeRatio
	default:
		return pacingRatio(item.TimeOnTaskSec, item.ExpectedDurationSec)
	}
}

func computeContentModality(items []itemEngagement) (ModalitySummary, bool, map[string]int) {
	counts := modalityItemCounts(items)
	if !modalityDataSufficient(counts) {
		return ModalitySummary{}, false, counts
	}
	affinity := computeModalityAffinity(items)
	comfort, _ := computeComplexityComfort(items)
	pacing := computePacingLabel(items)
	return ModalitySummary{
		ModalityAffinity:  affinity,
		ComplexityComfort: comfort,
		Pacing:            pacing,
	}, true, counts
}

func modalityItemCounts(items []itemEngagement) map[string]int {
	out := make(map[string]int)
	for _, item := range items {
		if item.Modality == "" {
			continue
		}
		out[string(item.Modality)]++
	}
	return out
}

func modalityDataSufficient(counts map[string]int) bool {
	total := 0
	modalities := 0
	for _, n := range counts {
		if n > 0 {
			modalities++
			total += n
		}
	}
	return total >= contentModalityMinDistinctItems && modalities >= contentModalityMinModalities
}

func computeModalityAffinity(items []itemEngagement) map[string]float64 {
	byModality := make(map[string][]float64)
	for _, item := range items {
		if item.Modality == "" {
			continue
		}
		byModality[string(item.Modality)] = append(byModality[string(item.Modality)], item.engagementScore())
	}
	out := map[string]float64{
		string(modalityVideo):       0,
		string(modalityReading):     0,
		string(modalityInteractive): 0,
		string(modalityQuiz):        0,
	}
	for mod, scores := range byModality {
		if len(scores) == 0 {
			continue
		}
		sum := 0.0
		for _, s := range scores {
			sum += s
		}
		out[mod] = round2(sum / float64(len(scores)))
	}
	return out
}

func computeComplexityComfort(items []itemEngagement) (*ComplexityComfortBand, float64) {
	byGrade := make(map[int][]float64)
	for _, item := range items {
		if item.Modality != modalityReading || item.ReadingLevelFKGL == nil {
			continue
		}
		grade := int(math.Round(*item.ReadingLevelFKGL))
		grade = clampInt(grade, 0, 12)
		byGrade[grade] = append(byGrade[grade], item.engagementScore())
	}
	if len(byGrade) < 2 {
		return nil, 0
	}
	grades := sortedIntKeys(byGrade)
	avgByGrade := make(map[int]float64, len(byGrade))
	for grade, scores := range byGrade {
		sum := 0.0
		for _, s := range scores {
			sum += s
		}
		avgByGrade[grade] = sum / float64(len(scores))
	}

	bestLow, bestHigh := -1, -1
	runLow := -1
	for i, grade := range grades {
		if avgByGrade[grade] >= contentModalityComfortThreshold {
			if runLow < 0 {
				runLow = grade
			}
			runHigh := grade
			if i+1 < len(grades) && grades[i+1] == grade+1 {
				continue
			}
			if bestLow < 0 || runHigh-runLow > bestHigh-bestLow {
				bestLow, bestHigh = runLow, runHigh
			}
			runLow = -1
			continue
		}
		runLow = -1
	}
	if bestLow < 0 {
		return nil, 0
	}
	confidence := round2(math.Min(1, float64(len(byGrade))/5.0))
	return &ComplexityComfortBand{
		Low:        fmt.Sprintf("grade%d", bestLow),
		High:       fmt.Sprintf("grade%d", bestHigh),
		Confidence: confidence,
	}, confidence
}

func computePacingLabel(items []itemEngagement) string {
	avgByMod := make(map[string]float64)
	counts := make(map[string]int)
	for _, item := range items {
		if item.Modality == "" {
			continue
		}
		mod := string(item.Modality)
		avgByMod[mod] += item.pacingScore()
		counts[mod]++
	}
	for mod, total := range avgByMod {
		if counts[mod] > 0 {
			avgByMod[mod] = round2(total / float64(counts[mod]))
		}
	}

	var thoroughMod, skimMod string
	thoroughScore := -1.0
	skimScore := 2.0
	for mod, avg := range avgByMod {
		if avg >= contentModalityThoroughThreshold && avg > thoroughScore {
			thoroughMod, thoroughScore = mod, avg
		}
		if avg <= contentModalitySkimThreshold && avg < skimScore {
			skimMod, skimScore = mod, avg
		}
	}
	if thoroughMod != "" && skimMod != "" && thoroughMod != skimMod {
		return fmt.Sprintf("thorough-on-%s-skim-on-%s", pacingModalityLabel(thoroughMod), pacingModalityLabel(skimMod))
	}
	if thoroughMod != "" {
		return "thorough-on-" + pacingModalityLabel(thoroughMod)
	}
	if skimMod != "" {
		return "skim-on-" + pacingModalityLabel(skimMod)
	}
	return "balanced"
}

func pacingModalityLabel(mod string) string {
	if mod == string(modalityReading) {
		return "text"
	}
	return mod
}

func modalityConfidence(itemCount int, modalityCount int) float64 {
	if itemCount < contentModalityMinDistinctItems || modalityCount < contentModalityMinModalities {
		return 0
	}
	c := math.Min(1, float64(itemCount)/12.0) * math.Min(1, float64(modalityCount)/4.0)
	return round2(math.Max(0.25, c))
}

func sortedIntKeys(m map[int][]float64) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}