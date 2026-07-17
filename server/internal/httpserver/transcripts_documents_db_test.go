package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/transcriptissue"
)

func setupTranscriptDocsTest(t *testing.T, ctx context.Context) (*pgxpool.Pool, http.Handler, string, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}

	em := fmt.Sprintf("transcript-docs-%d@test.com", time.Now().UnixNano())
	ph, _ := auth.HashPassword("longpassword0longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		pool.Close()
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	orgID := organization.SeedDefaultOrgID
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgID, uid); err != nil {
		pool.Close()
		t.Fatalf("set org: %v", err)
	}

	// Seed three terms + graded enrollments.
	var termIDs [3]uuid.UUID
	for i, name := range []string{"Fall 2024", "Spring 2025", "Summer 2025"} {
		start := time.Date(2024, time.Month(9+i*4), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 3, 0)
		if err := pool.QueryRow(ctx, `
INSERT INTO tenant.terms (org_id, name, term_type, start_date, end_date, status)
VALUES ($1, $2, 'semester', $3::date, $4::date, 'active')
RETURNING id
`, orgID, name, start, end).Scan(&termIDs[i]); err != nil {
			pool.Close()
			t.Fatalf("term %d: %v", i, err)
		}
	}

	grades := []string{"A", "B", "A-"}
	credits := []float64{3, 3, 4}
	for i := 0; i < 3; i++ {
		// Format: C-[A-Z0-9]{6}
		cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])
		var courseID, enrollID uuid.UUID
		if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id, term_id)
VALUES ($1, $2, $3, $4) RETURNING id
`, cc, fmt.Sprintf("Course %d", i+1), uid, termIDs[i]).Scan(&courseID); err != nil {
			pool.Close()
			t.Fatalf("course: %v", err)
		}
		if err := pool.QueryRow(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active)
VALUES ($1, $2, 'student', true) RETURNING id
`, courseID, uid).Scan(&enrollID); err != nil {
			pool.Close()
			t.Fatalf("enroll: %v", err)
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO course.final_grade_submissions
  (course_id, enrollment_id, submitted_by, computed_grade, final_grade, submission_method)
VALUES ($1, $2, $3, $4, $4, 'csv')
`, courseID, enrollID, uid, grades[i]); err != nil {
			pool.Close()
			t.Fatalf("final grade: %v", err)
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO catalog.catalog_sections (
  org_id, term_id, sis_course_id, sis_section_id, subject, course_number, title, credits, lms_course_id
)
VALUES ($1, $2, $3, $4, 'TEST', $5, $6, $7, $8)
`, orgID, termIDs[i], cc, cc+"-1", fmt.Sprintf("%d", 100+i), fmt.Sprintf("Course %d", i+1), credits[i], courseID); err != nil {
			pool.Close()
			t.Fatalf("catalog section: %v", err)
		}
	}

	if _, err := pool.Exec(ctx, `
UPDATE settings.transcripts_config SET official_enabled = true WHERE id = 1
`); err != nil {
		pool.Close()
		t.Fatalf("enable official: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, _ := signer.Sign(ctx, row.ID, em, "", "", nil)
	// Official issuance signs a VC (T08); PublicWebOrigin + JWTSecret required for did:web key material.
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{
		FFTranscripts:   true,
		PublicWebOrigin: "http://localhost:5173",
		JWTSecret:       "01234567890123456789012345678901",
	}})
	return pool, h, tok, uid
}

func TestTranscriptDocuments_PreviewGenerateHash_Pg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pool, h, tok, uid := setupTranscriptDocsTest(t, ctx)
	defer pool.Close()

	// Preview (no persistence).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transcripts/preview", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preview: want 200 got %d body=%s", w.Code, w.Body.String())
	}
	var preview struct {
		Persisted bool `json:"persisted"`
		Record    struct {
			Terms []struct {
				Courses []struct {
					Code string `json:"code"`
				} `json:"courses"`
			} `json:"terms"`
			Cumulative struct {
				GPA *float64 `json:"gpa"`
			} `json:"cumulative"`
		} `json:"record"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Persisted {
		t.Fatal("preview must not persist")
	}
	docs, err := transcriptsrepo.ListDocumentsByUser(ctx, pool, uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Fatalf("preview should not create documents, got %d", len(docs))
	}

	// Issue official twice → versions 1 and 2.
	for wantVersion := 1; wantVersion <= 2; wantVersion++ {
		body := bytes.NewBufferString(`{"variant":"official","format":["pdf","xml"]}`)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/transcripts/documents", body)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("generate v%d: want 201 got %d body=%s", wantVersion, w.Code, w.Body.String())
		}
		var resp struct {
			Document struct {
				ID      string `json:"id"`
				Version int    `json:"version"`
			} `json:"document"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if resp.Document.Version != wantVersion {
			t.Fatalf("version=%d want %d", resp.Document.Version, wantVersion)
		}
		docID, _ := uuid.Parse(resp.Document.ID)
		doc, err := transcriptsrepo.GetDocumentByID(ctx, pool, uid, docID)
		if err != nil || doc == nil {
			t.Fatalf("load doc: %v", err)
		}
		if !transcriptsrepo.VerifyDocumentHash(doc) {
			t.Fatal("hash verify failed on fresh document")
		}
		// Tamper → fail closed.
		doc.Canonical = append([]byte{}, doc.Canonical...)
		if len(doc.Canonical) > 10 {
			doc.Canonical[5] ^= 0xff
		}
		if transcriptsrepo.VerifyDocumentHash(doc) {
			t.Fatal("tampered canonical should fail hash verify")
		}
	}

	docs, err = transcriptsrepo.ListDocumentsByUser(ctx, pool, uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 issued docs, got %d", len(docs))
	}

	// Hash stability of generate service with fixed timestamp.
	fixed := time.Date(2026, 7, 16, 15, 0, 0, 0, time.UTC)
	r1, err := transcriptissue.Generate(ctx, pool, transcriptissue.GenerateParams{
		UserID: uid, GeneratedBy: uid, Variant: "in_progress", Persist: false, GeneratedAt: fixed,
		Formats: transcriptissue.GenerateFormats{PDF: true, XML: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := transcriptissue.Generate(ctx, pool, transcriptissue.GenerateParams{
		UserID: uid, GeneratedBy: uid, Variant: "in_progress", Persist: false, GeneratedAt: fixed,
		Formats: transcriptissue.GenerateFormats{PDF: true, XML: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r1.Hash != r2.Hash {
		t.Fatalf("canonical hash not stable: %s vs %s", r1.Hash, r2.Hash)
	}
}
