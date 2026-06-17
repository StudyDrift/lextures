package coursereviews

import (
	"expvar"
	"strconv"
	"sync/atomic"
)

var (
	reviewsTotal      atomic.Uint64
	ratingHistogram   [6]atomic.Uint64 // index 1–5 used for star counts
)

func init() {
	expvar.Publish("course_reviews_total", expvar.Func(func() any {
		return reviewsTotal.Load()
	}))
	expvar.Publish("average_rating_distribution", expvar.Func(func() any {
		out := make(map[string]uint64, 5)
		for i := 1; i <= 5; i++ {
			out[strconv.Itoa(i)] = ratingHistogram[i].Load()
		}
		return out
	}))
}

// RecordReviewSubmitted increments observability counters after a new or updated review.
func RecordReviewSubmitted(rating int) {
	reviewsTotal.Add(1)
	if rating >= 1 && rating <= 5 {
		ratingHistogram[rating].Add(1)
	}
}
