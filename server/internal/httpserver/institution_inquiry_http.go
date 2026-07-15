package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/institutioninquiry"
)

// ---------------------------------------------------------------------------
// IP rate limit (public lead form)
// ---------------------------------------------------------------------------

const (
	inquiryRateLimit  = 5
	inquiryRateWindow = 10 * time.Minute
	inquiryCleanEvery = time.Hour
	inquiryMaxBody    = 16 << 10 // 16 KiB
)

type inquiryIPEntry struct {
	count int
	reset time.Time
}

var (
	inquiryMu        sync.Mutex
	inquiryLimiters  = map[string]*inquiryIPEntry{}
	inquiryLastClean = time.Now()
)

func inquiryCheckRate(ip string) bool {
	inquiryMu.Lock()
	defer inquiryMu.Unlock()

	now := time.Now()
	if now.Sub(inquiryLastClean) > inquiryCleanEvery {
		for k, v := range inquiryLimiters {
			if now.After(v.reset) {
				delete(inquiryLimiters, k)
			}
		}
		inquiryLastClean = now
	}

	e, ok := inquiryLimiters[ip]
	if !ok {
		inquiryLimiters[ip] = &inquiryIPEntry{count: 1, reset: now.Add(inquiryRateWindow)}
		return true
	}
	if now.After(e.reset) {
		e.count = 1
		e.reset = now.Add(inquiryRateWindow)
		return true
	}
	e.count++
	return e.count <= inquiryRateLimit
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

type institutionInquiryRequest struct {
	OrganizationType  string `json:"organization_type"`
	OrganizationName  string `json:"organization_name"`
	ContactName       string `json:"contact_name"`
	Email             string `json:"email"`
	Role              string `json:"role"`
	EnrollmentSize    string `json:"enrollment_size"`
	HostingPreference string `json:"hosting_preference"`
	Message           string `json:"message"`
}

// POST /api/v1/public/institution-inquiries
func (d Deps) handlePublicInstitutionInquiry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		ip := onboardingRealIP(r)
		if !inquiryCheckRate(ip) {
			w.Header().Set("Retry-After", "600")
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Please try again later.")
			return
		}

		var req institutionInquiryRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, inquiryMaxBody)).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}

		orgType := inquiryTrim(req.OrganizationType, 80)
		orgName := inquiryTrim(req.OrganizationName, 200)
		contact := inquiryTrim(req.ContactName, 200)
		email := inquiryTrim(req.Email, 320)
		role := inquiryTrim(req.Role, 200)
		enrollment := inquiryTrim(req.EnrollmentSize, 80)
		hosting := inquiryTrim(req.HostingPreference, 120)
		message := inquiryTrim(req.Message, 5000)

		switch {
		case orgType == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "organization_type is required.")
			return
		case orgName == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "organization_name is required.")
			return
		case contact == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "contact_name is required.")
			return
		case email == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "email is required.")
			return
		case !inquiryLooksLikeEmail(email):
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid email address.")
			return
		case enrollment == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "enrollment_size is required.")
			return
		case hosting == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "hosting_preference is required.")
			return
		case message == "":
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "message is required.")
			return
		}

		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Service unavailable.")
			return
		}

		in := institutioninquiry.Inquiry{
			OrganizationType:  orgType,
			OrganizationName:  orgName,
			ContactName:       contact,
			Email:             email,
			EnrollmentSize:    enrollment,
			HostingPreference: hosting,
			Message:           message,
			IPAddress:         onboardingStrPtr(ip),
			UserAgent:         onboardingStrPtr(r.Header.Get("User-Agent")),
		}
		if role != "" {
			in.Role = &role
		}

		id, err := institutioninquiry.Insert(r.Context(), d.Pool, in)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save inquiry.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

func inquiryTrim(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}

func inquiryLooksLikeEmail(email string) bool {
	if len(email) < 3 || len(email) > 320 {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".")
}
