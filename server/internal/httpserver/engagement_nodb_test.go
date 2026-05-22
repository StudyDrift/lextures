package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestEngagement_PostEvents_FeatureOffReturns404(t *testing.T) {
	// Feature off → 404 before any auth check.
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, Config: config.Config{EngagementTrackingEnabled: false}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/events", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when feature off, got %d", rr.Code)
	}
}

func TestEngagement_PostEvents_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/events", nil)
	r.Header.Set("Authorization", "Bearer x")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestEngagement_PostEvents_FeatureOnRequiresAuth(t *testing.T) {
	// Feature on but no JWT → 401.
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s, Config: config.Config{EngagementTrackingEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/events", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when feature on but no token, got %d", rr.Code)
	}
}

func TestEngagement_GetEnrollmentEngagement_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil, Config: config.Config{EngagementTrackingEnabled: false}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/enrollments/00000000-0000-0000-0000-000000000001/engagement", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when feature off, got %d", rr.Code)
	}
}

func TestEngagement_GetEnrollmentEngagement_FeatureOnRequiresAuth(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s, Config: config.Config{EngagementTrackingEnabled: true}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/enrollments/00000000-0000-0000-0000-000000000001/engagement", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 when feature on but no token, got %d", rr.Code)
	}
}

func TestEngagement_VideoDropoff_Unauthorized(t *testing.T) {
	// requireCourseAccess checks auth first, so unauthenticated → 401 regardless of feature state.
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/analytics/video-dropoff/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestEngagement_OverviewReport_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/analytics/engagement-overview", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestEngagement_RoutesRegistered(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s, Config: config.Config{EngagementTrackingEnabled: true}})
	routes := []string{
		"/api/v1/analytics/events",
		"/api/v1/courses/CS101/enrollments/00000000-0000-0000-0000-000000000001/engagement",
		"/api/v1/courses/CS101/analytics/video-dropoff/00000000-0000-0000-0000-000000000001",
		"/api/v1/courses/CS101/analytics/engagement-overview",
	}
	for _, path := range routes {
		method := http.MethodGet
		if path == "/api/v1/analytics/events" {
			method = http.MethodPost
		}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Errorf("route not registered: %s %s", method, path)
		}
	}
}

func TestEngagement_BuildDropoffCurve_Empty(t *testing.T) {
	pts := buildDropoffCurve(nil, 0)
	if len(pts) != 0 {
		t.Fatalf("expected empty curve for no data, got %d points", len(pts))
	}
}

func TestEngagement_BuildDropoffCurve_AllWatched(t *testing.T) {
	// 5 users all watched 100%.
	pcts := []float64{100, 100, 100, 100, 100}
	pts := buildDropoffCurve(pcts, 5)
	if len(pts) == 0 {
		t.Fatal("expected non-empty curve")
	}
	// First point (0%) should show 100% still watching.
	if pts[0].PctStillWatching != 100 {
		t.Fatalf("want 100%% at start, got %.1f", pts[0].PctStillWatching)
	}
}

func TestEngagement_EngagementScore_Zero(t *testing.T) {
	s := engagementSummaryJSON{}
	score := engagementScore(s)
	if score != 0 {
		t.Fatalf("empty summary should have 0 score, got %f", score)
	}
}

func TestEngagement_EngagementScore_Max(t *testing.T) {
	h := 100.0
	s := engagementSummaryJSON{
		LoginsLast7Days:         5,
		AvgTimeOnTaskPerSession: 1800,
		AvgVideoWatchPct:        &h,
		AvgScrollDepth:          &h,
	}
	score := engagementScore(s)
	if score != 100 {
		t.Fatalf("max inputs should give 100, got %f", score)
	}
}
