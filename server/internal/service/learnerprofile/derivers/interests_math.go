package derivers

import (
	"math"
	"sort"
	"strings"
)

const (
	interestsDeriverVersion       = 1
	interestsWindowDays           = 90
	interestsMinTopics            = 2
	interestsMinSignalsPerTopic   = 2
	interestsMaxTopics            = 10
	interestsAssignedWeight       = 1.0
	interestsSelfDirectedWeight   = 2.5
	interestsNotebookGlobalWeight = 2.5
	interestsNotebookCourseWeight = 1.8
	interestsReadingWeight        = 2.0
	interestsFeedWeight           = 2.5
	interestsTaskWeight           = 2.0
	interestsSourceTable          = "learner.interests_signals"
)

// TopicSources counts contributing signal kinds per topic.
type TopicSources struct {
	Courses   int `json:"courses,omitempty"`
	Notebooks int `json:"notebooks,omitempty"`
	Reading   int `json:"reading,omitempty"`
	Feed      int `json:"feed,omitempty"`
	Tasks     int `json:"tasks,omitempty"`
}

// TopicItem is one ranked topic affinity in the facet summary.
type TopicItem struct {
	Topic          string       `json:"topic"`
	Affinity       float64      `json:"affinity"`
	Sources        TopicSources `json:"sources"`
	SelfDirected   bool         `json:"selfDirected"`
}

// InterestsSummary is the facet-level aggregate returned in summary JSON.
type InterestsSummary struct {
	Topics []TopicItem `json:"topics"`
}

type interestSignalKind string

const (
	signalCourses   interestSignalKind = "courses"
	signalNotebooks interestSignalKind = "notebooks"
	signalReading   interestSignalKind = "reading"
	signalFeed      interestSignalKind = "feed"
	signalTasks     interestSignalKind = "tasks"
)

type rawInterestSignal struct {
	Topic          string
	Kind           interestSignalKind
	Weight         float64
	SelfDirected   bool
	CourseID       string
	Ref            string
}

type topicAccumulator struct {
	Label            string
	WeightedScore    float64
	SelfDirectedW    float64
	AssignedW        float64
	Sources          TopicSources
	EvidenceByKind   map[interestSignalKind]topicEvidenceAcc
}

type topicEvidenceAcc struct {
	Count      int
	CourseIDs  map[string]struct{}
	Refs       []string
}

func topicKey(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func normalizeTopicLabel(label string) string {
	return strings.TrimSpace(label)
}

func computeInterests(signals []rawInterestSignal) (InterestsSummary, bool) {
	if len(signals) == 0 {
		return InterestsSummary{}, false
	}

	byTopic := make(map[string]*topicAccumulator)
	for _, sig := range signals {
		label := normalizeTopicLabel(sig.Topic)
		if label == "" {
			continue
		}
		key := topicKey(label)
		acc, ok := byTopic[key]
		if !ok {
			acc = &topicAccumulator{
				Label:          label,
				EvidenceByKind: make(map[interestSignalKind]topicEvidenceAcc),
			}
			byTopic[key] = acc
		}
		acc.WeightedScore += sig.Weight
		if sig.SelfDirected {
			acc.SelfDirectedW += sig.Weight
		} else {
			acc.AssignedW += sig.Weight
		}
		incrementTopicSource(&acc.Sources, sig.Kind)
		ev := acc.EvidenceByKind[sig.Kind]
		ev.Count++
		if ev.CourseIDs == nil {
			ev.CourseIDs = make(map[string]struct{})
		}
		if sig.CourseID != "" {
			ev.CourseIDs[sig.CourseID] = struct{}{}
		}
		if sig.Ref != "" && len(ev.Refs) < 5 {
			ev.Refs = append(ev.Refs, sig.Ref)
		}
		acc.EvidenceByKind[sig.Kind] = ev
	}

	qualified := make([]*topicAccumulator, 0, len(byTopic))
	for _, acc := range byTopic {
		rawCount := acc.Sources.Courses + acc.Sources.Notebooks + acc.Sources.Reading +
			acc.Sources.Feed + acc.Sources.Tasks
		if rawCount >= interestsMinSignalsPerTopic {
			qualified = append(qualified, acc)
		}
	}
	if len(qualified) < interestsMinTopics {
		return InterestsSummary{}, false
	}

	sort.Slice(qualified, func(i, j int) bool {
		if qualified[i].WeightedScore == qualified[j].WeightedScore {
			return qualified[i].Label < qualified[j].Label
		}
		return qualified[i].WeightedScore > qualified[j].WeightedScore
	})
	if len(qualified) > interestsMaxTopics {
		qualified = qualified[:interestsMaxTopics]
	}

	maxScore := qualified[0].WeightedScore
	if maxScore <= 0 {
		maxScore = 1
	}

	topics := make([]TopicItem, len(qualified))
	for i, acc := range qualified {
		topics[i] = TopicItem{
			Topic:        acc.Label,
			Affinity:     round2(acc.WeightedScore / maxScore),
			Sources:      acc.Sources,
			SelfDirected: acc.SelfDirectedW >= acc.AssignedW,
		}
	}
	return InterestsSummary{Topics: topics}, true
}

func incrementTopicSource(s *TopicSources, kind interestSignalKind) {
	switch kind {
	case signalCourses:
		s.Courses++
	case signalNotebooks:
		s.Notebooks++
	case signalReading:
		s.Reading++
	case signalFeed:
		s.Feed++
	case signalTasks:
		s.Tasks++
	}
}

func interestsConfidence(summary InterestsSummary, qualifiedTopicCount int) float64 {
	if qualifiedTopicCount < interestsMinTopics {
		return 0
	}
	topicFactor := math.Min(1, float64(len(summary.Topics))/float64(interestsMaxTopics))
	signalFactor := math.Min(1, float64(qualifiedTopicCount)/5.0)
	return round2(math.Max(0.25, topicFactor*signalFactor))
}