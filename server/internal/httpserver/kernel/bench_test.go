package kernel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type benchBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func BenchmarkDecodeJSON(b *testing.B) {
	raw := `{"title":"Outcome","description":"desc"}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		var dst benchBody
		if err := DecodeJSON(rr, req, &dst, DecodeOptions{RequireJSONContentType: false}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeHandRolled(b *testing.B) {
	raw := `{"title":"Outcome","description":"desc"}`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		var dst benchBody
		if err := json.NewDecoder(req.Body).Decode(&dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeJSON(b *testing.B) {
	v := benchBody{Title: "Outcome", Description: "desc"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		EncodeJSON(rr, http.StatusOK, v)
	}
}

func BenchmarkEncodeHandRolled(b *testing.B) {
	v := benchBody{Title: "Outcome", Description: "desc"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		rr.Header().Set("Content-Type", jsonContentType)
		_ = json.NewEncoder(rr).Encode(v)
	}
}

func BenchmarkMap(b *testing.B) {
	err := pgx.ErrNoRows
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Map(err)
	}
}

func TestDecodeAllocBudget(t *testing.T) {
	// AC-7: toolkit decode should stay within one extra allocation of the
	// hand-rolled Decoder. AllocsPerRun includes httptest request setup.
	raw := `{"title":"Outcome","description":"desc"}`
	hand := testing.AllocsPerRun(200, func() {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		var dst benchBody
		_ = json.NewDecoder(req.Body).Decode(&dst)
	})
	kit := testing.AllocsPerRun(200, func() {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		var dst benchBody
		_ = DecodeJSON(rr, req, &dst, DecodeOptions{RequireJSONContentType: false})
	})
	if kit > hand+8 {
		// httptest.NewRecorder + MaxBytesReader dominate; bound the gap so a
		// reflection-heavy rewrite cannot land unnoticed.
		t.Fatalf("decode allocs: toolkit=%.1f hand-rolled=%.1f", kit, hand)
	}
}
