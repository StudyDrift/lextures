package peerreview

import (
	"math"
	"testing"

	"github.com/lextures/lextures/server/internal/models/peerreview"
)

func TestAggregateScores_mean(t *testing.T) {
	scores := []float64{80, 90, 100}
	got := AggregateScores(scores, peerreview.AggregationMean)
	if got == nil || math.Abs(*got-90) > 0.001 {
		t.Fatalf("mean want 90 got %v", got)
	}
}

func TestAggregateScores_medianOdd(t *testing.T) {
	scores := []float64{70, 80, 90}
	got := AggregateScores(scores, peerreview.AggregationMedian)
	if got == nil || math.Abs(*got-80) > 0.001 {
		t.Fatalf("median want 80 got %v", got)
	}
}

func TestAggregateScores_medianEven(t *testing.T) {
	scores := []float64{70, 80, 90, 100}
	got := AggregateScores(scores, peerreview.AggregationMedian)
	if got == nil || math.Abs(*got-85) > 0.001 {
		t.Fatalf("median want 85 got %v", got)
	}
}

func TestAggregateScores_trimmed(t *testing.T) {
	scores := []float64{10, 80, 85, 90, 100}
	got := AggregateScores(scores, peerreview.AggregationTrimmed)
	if got == nil || math.Abs(*got-85) > 0.001 {
		t.Fatalf("trimmed mean want 85 got %v", got)
	}
}

func TestBlendGrade(t *testing.T) {
	got := BlendGrade(70, 90, 0.3)
	want := 0.7*70 + 0.3*90
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("blend want %v got %v", want, got)
	}
}

func TestAggregateScores_empty(t *testing.T) {
	if got := AggregateScores(nil, peerreview.AggregationMean); got != nil {
		t.Fatalf("empty scores want nil got %v", got)
	}
}
