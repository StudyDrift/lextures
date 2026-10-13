package notificationevents

import "testing"

func TestCommentAndWeeklyProgressEventsAreListed(t *testing.T) {
	want := map[string]bool{SubmissionComment: false, ParentWeeklyProgress: false}
	for _, et := range All {
		if _, ok := want[et]; ok {
			want[et] = true
		}
	}
	for et, seen := range want {
		if !seen {
			t.Errorf("%s missing from All", et)
		}
	}
}
