package derivers

import "testing"

func TestComputeInterests_SelfDirectedRanksHigher(t *testing.T) {
	assigned := make([]rawInterestSignal, 4)
	for i := range assigned {
		assigned[i] = rawInterestSignal{
			Topic:        "Statistics",
			Kind:         signalCourses,
			Weight:       interestsAssignedWeight,
			SelfDirected: false,
		}
	}
	selfDirected := make([]rawInterestSignal, 4)
	for i := range selfDirected {
		selfDirected[i] = rawInterestSignal{
			Topic:        "Ecology",
			Kind:         signalNotebooks,
			Weight:       interestsNotebookGlobalWeight,
			SelfDirected: true,
		}
	}
	signals := append(assigned, selfDirected...)
	summary, sufficient := computeInterests(signals)
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if len(summary.Topics) < 2 {
		t.Fatalf("topics=%+v", summary.Topics)
	}
	if summary.Topics[0].Topic != "Ecology" {
		t.Fatalf("top topic=%q want Ecology", summary.Topics[0].Topic)
	}
	if !summary.Topics[0].SelfDirected {
		t.Fatal("expected ecology to be self-directed")
	}
}

func TestComputeInterests_NoFabricatedTopic(t *testing.T) {
	signals := []rawInterestSignal{
		{Topic: "", Kind: signalReading, Weight: interestsReadingWeight, SelfDirected: true, Ref: "orphan-read"},
		{Topic: "", Kind: signalNotebooks, Weight: interestsNotebookGlobalWeight, SelfDirected: true, Ref: "orphan-note"},
	}
	_, sufficient := computeInterests(signals)
	if sufficient {
		t.Fatal("expected insufficient_data when topics are unlabeled")
	}
}

func TestComputeInterests_SingleTopicInsufficient(t *testing.T) {
	signals := []rawInterestSignal{
		{Topic: "Ecology", Kind: signalNotebooks, Weight: interestsNotebookGlobalWeight, SelfDirected: true},
		{Topic: "Ecology", Kind: signalReading, Weight: interestsReadingWeight, SelfDirected: true},
		{Topic: "Ecology", Kind: signalFeed, Weight: interestsFeedWeight, SelfDirected: true},
	}
	_, sufficient := computeInterests(signals)
	if sufficient {
		t.Fatal("expected insufficient_data with only one qualifying topic")
	}
}

func TestComputeInterests_TwoTopicsSufficient(t *testing.T) {
	signals := []rawInterestSignal{
		{Topic: "Ecology", Kind: signalNotebooks, Weight: interestsNotebookGlobalWeight, SelfDirected: true},
		{Topic: "Ecology", Kind: signalReading, Weight: interestsReadingWeight, SelfDirected: true},
		{Topic: "Statistics", Kind: signalCourses, Weight: interestsAssignedWeight},
		{Topic: "Statistics", Kind: signalCourses, Weight: interestsAssignedWeight},
	}
	summary, sufficient := computeInterests(signals)
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if len(summary.Topics) != 2 {
		t.Fatalf("topics=%+v", summary.Topics)
	}
	if summary.Topics[0].Affinity < summary.Topics[1].Affinity {
		t.Fatalf("affinities not ranked: %+v", summary.Topics)
	}
}

func TestComputeInterests_CapsTopics(t *testing.T) {
	signals := make([]rawInterestSignal, 0, 30)
	for i := 0; i < 12; i++ {
		name := "Topic" + string(rune('A'+i))
		signals = append(signals,
			rawInterestSignal{Topic: name, Kind: signalNotebooks, Weight: interestsNotebookGlobalWeight, SelfDirected: true},
			rawInterestSignal{Topic: name, Kind: signalFeed, Weight: interestsFeedWeight, SelfDirected: true},
		)
	}
	summary, sufficient := computeInterests(signals)
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if len(summary.Topics) != interestsMaxTopics {
		t.Fatalf("topics=%d want cap %d", len(summary.Topics), interestsMaxTopics)
	}
}

func TestTopicForNotebookPage_InheritsGroup(t *testing.T) {
	pages := []notebookPage{
		{ID: "g1", Title: "Ecology", Kind: "group"},
		{ID: "p1", ParentID: strPtr("g1"), Kind: "page", ContentMd: "notes"},
	}
	groupTopics := map[string]string{"g1": "Ecology"}
	got := topicForNotebookPage(pages[1], pages, groupTopics, "")
	if got != "Ecology" {
		t.Fatalf("topic=%q want Ecology", got)
	}
}

func strPtr(s string) *string { return &s }