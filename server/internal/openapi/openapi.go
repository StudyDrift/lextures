// Package openapi serves the API description and Swagger UI until a fuller generated spec exists.
package openapi

import (
	"net/http"
)

// spec is OpenAPI 3.0, aligned with the legacy Rust Utoipa bootstrap (title, one health path).
// Extend this document as route handlers are ported; clients may generate TS types from
// /api/openapi.json per migration notes.
const spec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "StudyDrift API",
    "description": "Lextures LMS HTTP API. Generate TypeScript types: npx openapi-typescript http://localhost:8080/api/openapi.json -o src/lib/api-types.generated.ts (with the API running).",
    "version": "0.1.0"
  },
  "tags": [
    { "name": "meta", "description": "Health and API metadata" },
    { "name": "auth", "description": "Sign-in and password reset (ported from server/src/routes/auth.rs)" },
    { "name": "me", "description": "Current user (ported from server/src/routes/me.rs)" },
    { "name": "accommodations", "description": "Student accommodations (server/src/routes/accommodations.rs)" },
    { "name": "search", "description": "Global search index (server/src/routes/search.rs)" },
    { "name": "reports", "description": "Admin reports (server/src/routes/reports.rs); requires global:app:reports:view" },
    { "name": "communication", "description": "Inbox / messaging (server/src/routes/communication.rs)" },
    { "name": "courses", "description": "Course APIs (server/src/routes/courses.rs; partial in Go)" },
    { "name": "admin", "description": "Global Admin maintenance (server/src/routes/admin.rs; requires global:app:rbac:manage)" },
    { "name": "settings", "description": "Roles and permissions (server/src/routes/rbac.rs; requires global:app:rbac:manage)" }
  ],
  "paths": {
    "/health": {
      "get": {
        "tags": ["meta"],
        "summary": "Liveness (alias for /health/live)",
        "responses": {
          "200": {
            "description": "JSON liveness payload {\"status\":\"ok\"}"
          }
        }
      }
    },
    "/health/live": {
      "get": {
        "tags": ["meta"],
        "summary": "Liveness probe",
        "responses": {
          "200": {
            "description": "Process is alive; no dependency checks"
          }
        }
      }
    },
    "/health/ready": {
      "get": {
        "tags": ["meta"],
        "summary": "Readiness probe",
        "responses": {
          "200": {
            "description": "All critical dependencies reachable"
          },
          "503": {
            "description": "One or more dependencies unavailable (safe JSON, no internal errors)"
          }
        }
      }
    },
    "/health/detailed": {
      "get": {
        "tags": ["meta"],
        "summary": "Detailed health (Global Admin JWT required)",
        "responses": {
          "200": {
            "description": "Per-component latency and safe error summaries"
          },
          "401": {
            "description": "Authentication required"
          }
        }
      }
    },
    "/api/v1/auth/login": {
      "post": {
        "tags": ["auth"],
        "summary": "Email and password sign-in (short-lived access_token + refresh_token)",
        "responses": { "200": { "description": "Access token, refresh token, expires_in, user" }, "401": { "description": "Invalid credentials" } }
      }
    },
    "/api/v1/auth/signup": {
      "post": {
        "tags": ["auth"],
        "summary": "Create account (teacher role + welcome message; returns access + refresh tokens)",
        "responses": { "200": { "description": "Access token, refresh token, user" }, "409": { "description": "Email taken" } }
      }
    },
    "/api/v1/auth/forgot-password": {
      "post": {
        "tags": ["auth"],
        "summary": "Request password reset email",
        "responses": { "200": { "description": "Generic success message" } }
      }
    },
    "/api/v1/auth/reset-password": {
      "post": {
        "tags": ["auth"],
        "summary": "Complete password reset with one-time token",
        "responses": { "200": { "description": "Password updated" }, "400": { "description": "Invalid or expired token" } }
      }
    },
    "/api/v1/auth/refresh": {
      "post": {
        "tags": ["auth"],
        "summary": "Exchange refresh token for new access token (rotates refresh token)",
        "responses": { "200": { "description": "access_token, refresh_token, expires_in" }, "401": { "description": "Invalid or expired refresh token" } }
      }
    },
    "/api/v1/auth/logout": {
      "post": {
        "tags": ["auth"],
        "summary": "Revoke refresh token (JSON body: refresh_token)",
        "responses": { "200": { "description": "ok" }, "401": { "description": "Invalid refresh token" } }
      }
    },
    "/api/v1/auth/logout-all": {
      "post": {
        "tags": ["auth"],
        "summary": "Revoke all refresh tokens for the current user (Bearer access token)",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "ok" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/auth/magic-link/request": {
      "post": {
        "tags": ["auth"],
        "summary": "Request a one-time email sign-in link (disabled when MAGIC_LINK_ENABLED=0)",
        "responses": { "200": { "description": "Generic message (enumeration-safe)" }, "404": { "description": "Feature disabled" }, "429": { "description": "Rate limited" } }
      }
    },
    "/api/v1/auth/magic-link/consume": {
      "get": {
        "tags": ["auth"],
        "summary": "Consume magic link token from email (query: token, optional redirect_to for SPA)",
        "responses": { "200": { "description": "Access token or MFA pending" }, "410": { "description": "Used or expired token" } }
      },
      "post": {
        "tags": ["auth"],
        "summary": "Consume magic link token (JSON body: token)",
        "responses": { "200": { "description": "Access token or MFA pending" }, "410": { "description": "Used or expired token" } }
      }
    },
    "/api/v1/auth/saml/status": {
      "get": {
        "tags": ["auth"],
        "summary": "SAML IdP status for the login page (default IdP when enabled)",
        "responses": { "200": { "description": "enabled, optional idp" } }
      }
    },
    "/api/v1/auth/oidc/status": {
      "get": {
        "tags": ["auth"],
        "summary": "Which OIDC IdPs are configured (env + custom DB providers)",
        "responses": { "200": { "description": "enabled, apiBase, provider flags, custom" } }
      }
    },
    "/api/v1/auth/oidc/link": {
      "post": {
        "tags": ["auth"],
        "summary": "Start OIDC account linking (returns loginUrl with linkId); browser completes at /auth/oidc/...",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "ok, linkId, loginUrl" }, "400": { "description": "Invalid request" }, "401": { "description": "Not signed in" } }
      }
    },
    "/auth/oidc/{provider}/login": {
      "get": {
        "tags": ["auth"],
        "summary": "Begin OIDC authorization (redirect to IdP; query: next, linkId, configId for custom)",
        "parameters": [
          { "name": "provider", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "next", "in": "query", "schema": { "type": "string" } },
          { "name": "linkId", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "configId", "in": "query", "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "307": { "description": "Redirect to IdP" }, "400": { "description": "Invalid input" }, "503": { "description": "No database" } }
      }
    },
    "/auth/oidc/{provider}/callback": {
      "get": {
        "tags": ["auth"],
        "summary": "OIDC callback; returns HTML with fragment access_token (same as SAML browser flow)",
        "parameters": [
          { "name": "provider", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "text/html" }, "400": { "description": "Invalid code/state" } }
      }
    },
    "/api/v1/search": {
      "get": {
        "tags": ["search", "me"],
        "summary": "Courses the user is enrolled in and people visible with enrollments:read on each course",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "courses, people" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/search/query": {
      "get": {
        "tags": ["search", "me"],
        "summary": "Query-driven command palette search (courses, people, module content)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "q", "in": "query", "required": true, "schema": { "type": "string", "minLength": 2 } },
          { "name": "scope", "in": "query", "schema": { "type": "string" }, "description": "Optional course code scope" },
          { "name": "types", "in": "query", "schema": { "type": "string" }, "description": "Comma-separated: course,person,content" }
        ],
        "responses": { "200": { "description": "QueryResponse grouped results" }, "400": { "description": "Query too short" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/reports/learning-activity": {
      "get": {
        "tags": ["reports"],
        "summary": "Learning activity (user.user_audit) aggregates for a date range",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "from", "in": "query", "description": "RFC 3339 start (default: 30 days before to)" },
          { "name": "to", "in": "query", "description": "RFC 3339 end exclusive upper bound in SQL (default: now)" }
        ],
        "responses": { "200": { "description": "LearningActivityReport" }, "400": { "description": "Invalid range" }, "401": { "description": "Not signed in" }, "403": { "description": "Missing global:app:reports:view" } }
      }
    },
    "/api/v1/communication/messages": {
      "get": {
        "tags": ["communication", "me"],
        "summary": "List mailbox (folder, optional q); folders: inbox, starred, sent, drafts, trash",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "folder", "in": "query", "required": true, "schema": { "type": "string" } },
          { "name": "q", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "messages" }, "400": { "description": "Invalid folder" }, "401": { "description": "Not signed in" } }
      },
      "post": {
        "tags": ["communication", "me"],
        "summary": "Send a message to a user by email, or save a draft",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "message id" }, "400": { "description": "Invalid request" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/communication/messages/{id}": {
      "get": {
        "tags": ["communication", "me"],
        "summary": "Get a single message",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "responses": { "200": { "description": "message" }, "404": { "description": "Not found" } }
      },
      "patch": {
        "tags": ["communication", "me"],
        "summary": "Mark read, star, or move to folder",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "responses": { "200": { "description": "ok" }, "400": { "description": "Invalid" }, "404": { "description": "Not found" } }
      }
    },
    "/api/v1/communication/unread-count": {
      "get": {
        "tags": ["communication", "me"],
        "summary": "Unread count for inbox",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "unreadInbox" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/communication/ws": {
      "get": {
        "tags": ["communication", "me"],
        "summary": "WebSocket; first text frame: JSON with authToken (login JWT); server pushes mailbox events",
        "responses": { "200": { "description": "WebSocket upgrade" }, "503": { "description": "realtime not configured" } }
      }
    },
    "/api/v1/marketplace/courses": {
      "get": {
        "tags": ["courses"],
        "summary": "Authenticated marketplace storefront course list (plan MKT3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "q", "in": "query", "schema": { "type": "string" } },
          { "name": "category", "in": "query", "schema": { "type": "string" } },
          { "name": "level", "in": "query", "schema": { "type": "string", "enum": ["beginner", "intermediate", "advanced"] } },
          { "name": "language", "in": "query", "schema": { "type": "string" } },
          { "name": "price_max", "in": "query", "schema": { "type": "integer", "minimum": 0 } },
          { "name": "free_only", "in": "query", "schema": { "type": "boolean" } },
          { "name": "sort", "in": "query", "schema": { "type": "string", "enum": ["popular", "rating", "newest", "relevance", "price"] } },
          { "name": "cursor", "in": "query", "schema": { "type": "string" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "minimum": 1, "maximum": 50 } }
        ],
        "responses": {
          "200": { "description": "{ courses: MarketplaceCard[], total, nextCursor }" },
          "400": { "description": "Invalid filter" },
          "401": { "description": "Sign in required" },
          "404": { "description": "Marketplace feature disabled" }
        }
      }
    },
    "/api/v1/public/institution-inquiries": {
      "post": {
        "tags": ["public"],
        "summary": "Submit institution request-information lead",
        "description": "Stores a marketing lead from lextures.com/request-information. Unauthenticated; IP rate-limited. Email notification may be added later.",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["organization_type", "organization_name", "contact_name", "email", "enrollment_size", "hosting_preference", "message"],
                "properties": {
                  "organization_type": { "type": "string" },
                  "organization_name": { "type": "string" },
                  "contact_name": { "type": "string" },
                  "email": { "type": "string", "format": "email" },
                  "role": { "type": "string" },
                  "enrollment_size": { "type": "string" },
                  "hosting_preference": { "type": "string" },
                  "message": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "{ id: UUID }" },
          "400": { "description": "Invalid input" },
          "429": { "description": "Rate limited" },
          "503": { "description": "Database unavailable" }
        }
      }
    },
    "/api/v1/public/marketplace/courses": {
      "get": {
        "tags": ["public-marketplace"],
        "summary": "Unauthenticated public marketplace course list (plan MKT7)",
        "parameters": [
          { "name": "q", "in": "query", "schema": { "type": "string" } },
          { "name": "category", "in": "query", "schema": { "type": "string" } },
          { "name": "level", "in": "query", "schema": { "type": "string", "enum": ["beginner", "intermediate", "advanced"] } },
          { "name": "language", "in": "query", "schema": { "type": "string" } },
          { "name": "price_max", "in": "query", "schema": { "type": "integer", "minimum": 0 } },
          { "name": "free_only", "in": "query", "schema": { "type": "boolean" } },
          { "name": "sort", "in": "query", "schema": { "type": "string", "enum": ["popular", "rating", "newest", "relevance", "price"] } },
          { "name": "cursor", "in": "query", "schema": { "type": "string" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "minimum": 1, "maximum": 50 } }
        ],
        "responses": {
          "200": { "description": "{ courses: PublicMarketplaceCourse[], total, nextCursor } — no owned field" },
          "400": { "description": "Invalid filter" },
          "404": { "description": "Marketplace feature disabled" }
        }
      }
    },
    "/api/v1/public/marketplace/categories": {
      "get": {
        "tags": ["public-marketplace"],
        "summary": "Public marketplace category facets (plan MKT7)",
        "responses": {
          "200": { "description": "{ categories: [{ category, count }] }" },
          "404": { "description": "Marketplace feature disabled" }
        }
      }
    },
    "/api/v1/public/marketplace/courses/{slug}": {
      "get": {
        "tags": ["public-marketplace"],
        "summary": "Public marketplace course detail with JSON-LD (plan MKT7)",
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ course, whatsIncluded, jsonLd }" },
          "404": { "description": "Not listed, unpublished, or feature disabled" }
        }
      }
    },
    "/api/v1/public/marketplace/courses/{slug}/reviews": {
      "get": {
        "tags": ["public-marketplace"],
        "summary": "Public marketplace course reviews by slug (plan MKT7)",
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "cursor", "in": "query", "schema": { "type": "string" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer" } }
        ],
        "responses": {
          "200": { "description": "{ summary, reviews, nextCursor? }" },
          "404": { "description": "Not listed or feature disabled" }
        }
      }
    },
    "/api/v1/marketplace/categories": {
      "get": {
        "tags": ["courses"],
        "summary": "Marketplace category facets for listed courses (plan MKT3)",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "{ categories: [{ category, count }] }" },
          "401": { "description": "Sign in required" },
          "404": { "description": "Marketplace feature disabled" }
        }
      }
    },
    "/api/v1/marketplace/courses/{slug}": {
      "get": {
        "tags": ["courses"],
        "summary": "Marketplace course detail by catalog slug or course code (plan MKT3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ course, owned, priceCents, priceCurrency, listPriceCents, whatsIncluded, rating }" },
          "401": { "description": "Sign in required" },
          "404": { "description": "Not listed, unpublished, or feature disabled" }
        }
      }
    },
    "/api/v1/marketplace/courses/{slug}/claim": {
      "post": {
        "tags": ["courses"],
        "summary": "Claim a free marketplace course (entitlement + enrollment) (plan MKT4)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ enrolled, entitlementId, alreadyOwned?, firstItemId?, courseCode }" },
          "401": { "description": "Sign in required" },
          "402": { "description": "Paid course — use checkout; body includes checkoutHint" },
          "404": { "description": "Not listed or feature disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/marketplace/courses/{slug}/checkout": {
      "post": {
        "tags": ["courses"],
        "summary": "Start Stripe Checkout for a paid marketplace course (plan MKT4)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ checkoutUrl, sessionId } or { alreadyOwned, courseCode, courseId }" },
          "400": { "description": "Free course — use claim instead" },
          "401": { "description": "Sign in required" },
          "404": { "description": "Not listed, feature disabled, or payments disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/me/purchases": {
      "get": {
        "tags": ["billing"],
        "summary": "List the caller's active marketplace course purchases (plan MKT5)",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": {
            "description": "{ purchases: [{ courseCode, courseId, title, priceCents, currency, source, acquiredAt, receiptUrl?, entitlementId }] }"
          },
          "401": { "description": "Sign in required" },
          "404": { "description": "Marketplace feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/catalog-listing": {
      "get": {
        "tags": ["courses"],
        "summary": "Catalog and marketplace listing settings for a course",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": {
            "description": "listing",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "listing": { "$ref": "#/components/schemas/CatalogListing" }
                  }
                }
              }
            }
          },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "put": {
        "tags": ["courses"],
        "summary": "Update public catalog and marketplace listing settings",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/CatalogListingBody" }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Updated listing",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "listing": { "$ref": "#/components/schemas/CatalogListing" }
                  }
                }
              }
            }
          },
          "400": { "description": "Invalid input" },
          "403": { "description": "Forbidden" },
          "422": { "description": "Cannot list unpublished course in marketplace" }
        }
      }
    },
    "/api/v1/courses/{course_code}/course-context": {
      "post": {
        "tags": ["courses", "me"],
        "summary": "Record course visit or content open/leave (LMS state sync; user.user_audit)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["kind"],
                "properties": {
                  "kind": { "type": "string", "description": "course_visit | content_open | content_leave" },
                  "structureItemId": { "type": "string", "format": "uuid", "description": "Required for content_open / content_leave" }
                }
              }
            }
          }
        },
        "responses": {
          "204": { "description": "Recorded" },
          "400": { "description": "Invalid input" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Not enrolled, unknown course, or content item" }
        }
      }
    },
    "/api/v1/me/learner-profile": {
      "get": {
        "tags": ["me"],
        "summary": "Caller learner profile (facets, insights, evidence summary)",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "profile object with facets" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" }
        }
      }
    },
    "/api/v1/me/learner-profile/facets/{facetKey}": {
      "get": {
        "tags": ["me"],
        "summary": "One learner profile facet with insights",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "facetKey", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "facet and insights" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Unknown facet or not derived" }
        }
      }
    },
    "/api/v1/me/learner-profile/facets/{facetKey}/evidence": {
      "get": {
        "tags": ["me"],
        "summary": "Provenance evidence drill-down for a facet",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "facetKey", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "insightKey to evidence map" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Unknown facet" }
        }
      }
    },
    "/api/v1/me/learner-profile/pause": {
      "post": {
        "tags": ["me"],
        "summary": "Pause learner profile derivation",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "status paused" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/me/learner-profile/resume": {
      "post": {
        "tags": ["me"],
        "summary": "Resume learner profile derivation",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "status active" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/me/learner-profile/reset": {
      "post": {
        "tags": ["me"],
        "summary": "Reset (erase) learner profile data",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "status reset" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/me/learner-profile/export": {
      "get": {
        "tags": ["me"],
        "summary": "Portable learner profile export with provenance",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "JSON export with facets, insights, evidence" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/me/permissions": {
      "get": {
        "tags": ["me"],
        "summary": "Effective permission strings (optional courseCode, viewAs query)",
        "parameters": [
          { "name": "courseCode", "in": "query", "schema": { "type": "string" } },
          { "name": "viewAs", "in": "query", "schema": { "type": "string", "enum": ["teacher", "student"] } }
        ],
        "responses": { "200": { "description": "permissionStrings" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/me/oidc-identities": {
      "get": {
        "tags": ["me"],
        "summary": "Linked OIDC identities (id, provider, email)",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "identities array" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/me/oidc-identities/{id}": {
      "delete": {
        "tags": ["me"],
        "summary": "Unlink an OIDC identity by id",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "ok" }, "401": { "description": "Not signed in" }, "404": { "description": "Not found" } }
      }
    },
    "/api/v1/me/notebooks/query": {
      "post": {
        "tags": ["me"],
        "summary": "RAG over client-supplied course notebook Markdown (OpenRouter)",
        "security": [ { "bearerAuth": [] } ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "question": { "type": "string" },
                  "notebooks": {
                    "type": "array",
                    "items": {
                      "type": "object",
                      "properties": {
                        "courseCode": { "type": "string" },
                        "courseTitle": { "type": "string" },
                        "markdown": { "type": "string" }
                      }
                    }
                  }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "answerMarkdown, sources" },
          "400": { "description": "Invalid input" },
          "401": { "description": "Not signed in" },
          "502": { "description": "Model / OpenRouter error" },
          "503": { "description": "AI not configured" }
        }
      }
    },
    "/api/v1/accommodations/users": {
      "get": {
        "tags": ["accommodations"],
        "summary": "Search learners for accommodation management (q= email, name, sid, or uuid)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "q", "in": "query", "required": true, "schema": { "type": "string" } } ],
        "responses": { "200": { "description": "users" }, "400": { "description": "Invalid input" }, "401": { "description": "Not signed in" }, "403": { "description": "Missing global:user:accommodations:manage" } }
      }
    },
    "/api/v1/enrollments/{enrollmentID}/accommodation-summary": {
      "get": {
        "tags": ["accommodations"],
        "summary": "Instructor summary of effective accommodation flags for the enrollment (requires course enrollments read on that course)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "enrollmentID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "responses": { "200": { "description": "hasAccommodation, flags" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" }, "404": { "description": "Not found" } }
      }
    },
    "/api/v1/users/{userID}/accommodations": {
      "get": {
        "tags": ["accommodations"],
        "summary": "List a learner’s accommodation records (coordinator perm)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "userID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "responses": { "200": { "description": "Array of accommodation records" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      },
      "post": {
        "tags": ["accommodations"],
        "summary": "Create a learner accommodation row",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "userID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "responses": { "200": { "description": "Record" }, "400": { "description": "Validation or unknown course" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/users/{userID}/accommodations/{accommodationID}": {
      "put": {
        "tags": ["accommodations"],
        "summary": "Update an accommodation row",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "userID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "accommodationID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "Record" }, "400": { "description": "Validation" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" }, "404": { "description": "Not found" } }
      },
      "delete": {
        "tags": ["accommodations"],
        "summary": "Delete an accommodation row",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "userID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "accommodationID", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "204": { "description": "No content" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" }, "404": { "description": "Not found" } }
      }
    },
    "/api/v1/me/accommodations": {
      "get": {
        "tags": ["accommodations", "me"],
        "summary": "List this learner’s active (by date range) accommodation summary entries",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "accommodations" }, "401": { "description": "Not signed in" } }
      }
    },
    "/api/v1/admin/jobs/irt-calibrate": {
      "post": {
        "tags": ["admin"],
        "summary": "Start IRT 2PL calibration job (202 + jobId); fits a/b for active uncalibrated or pilot items with ≥200 responses; optional conceptId scopes to one concept",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "202": { "description": "Accepted" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/admin/originality-config": {
      "put": {
        "tags": ["admin"],
        "summary": "Upsert platform originality provider settings",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "ok" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/admin/users/{userId}/dsar-export": {
      "get": {
        "tags": ["admin"],
        "summary": "DSAR FERPA slice: originality report metadata for a user",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "userId", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "Export" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/admin/users/{userId}/sessions": {
      "delete": {
        "tags": ["admin"],
        "summary": "Revoke all refresh tokens and bump session version for a user (plan 4.8)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "userId", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "ok" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/admin/saml/config": {
      "get": {
        "tags": ["admin"],
        "summary": "Default SAML IdP config (or { config: null })",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "IdP or null config" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      },
      "put": {
        "tags": ["admin"],
        "summary": "Create or update SAML IdP",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "id, entityId" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/admin/oidc/providers": {
      "get": {
        "tags": ["admin"],
        "summary": "List custom OIDC provider configurations",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "providers" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      },
      "put": {
        "tags": ["admin"],
        "summary": "Create or update a custom OIDC provider",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "id" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      }
    },
    "/api/v1/admin/provisioning/oneroster/upload": {
      "post": {
        "tags": ["admin"],
        "summary": "Upload OneRoster CSV bundle (multipart; ONEROSTER_ENABLED=1)",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "201": { "description": "syncRunId" }, "400": {}, "401": {}, "403": {}, "404": { "description": "Feature off" } }
      }
    },
    "/api/v1/admin/provisioning/oneroster/sync-runs": {
      "get": {
        "tags": ["admin"],
        "summary": "List OneRoster sync runs for an institution",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "syncRuns" }, "401": {}, "403": {}, "404": {} }
      }
    },
    "/api/v1/admin/provisioning/oneroster/sync-runs/{id}": {
      "get": {
        "tags": ["admin"],
        "summary": "OneRoster sync run event log",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "events" }, "401": {}, "403": {}, "404": {} }
      }
    },
    "/api/v1/admin/provisioning/oneroster/bearer-credentials": {
      "post": {
        "tags": ["admin"],
        "summary": "Register hashed bearer token for GET /oneroster/v1p2/*",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "ok" }, "401": {}, "403": {}, "404": {} }
      }
    },
    "/oneroster/v1p2/users": {
      "get": {
        "tags": ["admin"],
        "summary": "OneRoster-style users collection (Bearer token from admin credential)",
        "responses": { "200": {}, "401": {} }
      }
    },
      "get": {
        "tags": ["settings"],
        "summary": "List all permission rows",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "permissions" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      },
      "post": {
        "tags": ["settings"],
        "summary": "Create a permission",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "permission" }, "400": { "description": "Invalid input" }, "401": {}, "403": {} }
      }
    },
    "/api/v1/admin/intro-course": {
      "get": {
        "tags": ["admin"],
        "summary": "Intro course admin status (IC08)",
        "description": "Operational snapshot: flag state, content version, sync/backfill health, locale coverage. Requires global:app:rbac:manage.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "{ enabled, coursePresent, contentVersion, moduleCount, lastSyncedAt, backfill, localeCoverage }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/admin/intro-course/resync": {
      "post": {
        "tags": ["admin"],
        "summary": "Re-sync the canonical intro course (IC01)",
        "description": "Idempotently provisions or reconciles the platform-owned Welcome to Lextures course. Requires global:app:rbac:manage.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "{ courseId, status: created|reconciled }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "409": { "description": "Intro course disabled and not yet provisioned" }
        }
      }
    },
    "/api/v1/me/intro-course": {
      "get": {
        "tags": ["me"],
        "summary": "Intro course progress and completion (IC05)",
        "description": "Returns enrolled state, module progress, running grade, completion timestamp, credential id, next item deep link, and IC06 onboarding UI flags.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "{ enrolled, courseCode?, modulesComplete, modulesTotal, percent, runningGrade?, completedAt?, credentialId?, nextItem?, modules?, welcomeBannerDismissed, celebrationSeen }" },
          "401": { "description": "Not signed in" }
        }
      }
    },
    "/api/v1/me/intro-course/welcome-banner-dismissed": {
      "put": {
        "tags": ["me"],
        "summary": "Dismiss intro course welcome banner (IC06)",
        "description": "Persists first-login welcome banner dismissal per user across devices.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "204": { "description": "Dismissed" },
          "401": { "description": "Not signed in" }
        }
      }
    },
    "/api/v1/me/intro-course/celebration-seen": {
      "put": {
        "tags": ["me"],
        "summary": "Mark intro course completion celebration seen (IC06)",
        "description": "Persists that the one-time completion celebration was shown and dismissed.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "204": { "description": "Recorded" },
          "401": { "description": "Not signed in" }
        }
      }
    },
    "/api/v1/admin/intro-course/analytics": {
      "get": {
        "tags": ["admin"],
        "summary": "Intro course completion analytics (IC05/IC08)",
        "description": "Completion rate, per-module funnel, drop-off module, and average time-to-complete. Requires global:app:rbac:manage.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "{ enrolled, completed, completionRate, perModuleFunnel, dropOffModuleSlug, avgTimeToCompleteHours }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/admin/intro-course/backfill": {
      "post": {
        "tags": ["admin"],
        "summary": "Start or resume intro course enrollment backfill (IC02)",
        "description": "Queues a durable job to enroll eligible existing users as students in the intro course. Requires global:app:rbac:manage.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "202": { "description": "{ startedAt, remaining }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "409": { "description": "Intro course disabled" }
        }
      },
      "get": {
        "tags": ["admin"],
        "summary": "Intro course enrollment backfill status (IC02)",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "{ startedAt, completedAt, enrolledCount, remaining }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/feedback": {
      "post": {
        "tags": ["me"],
        "summary": "Submit in-app product feedback (plan FB0)",
        "description": "Creates one feedback.submissions row for the authenticated user. Requires ffFeedback enabled.",
        "security": [ { "bearerAuth": [] } ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["message", "source"],
                "properties": {
                  "message": { "type": "string", "maxLength": 5000 },
                  "category": { "type": "string", "enum": ["bug", "idea", "question", "praise", "other"] },
                  "source": { "type": "string", "enum": ["web", "ios", "android"] },
                  "app_version": { "type": "string" },
                  "context": {
                    "type": "object",
                    "properties": {
                      "route": { "type": "string" },
                      "locale": { "type": "string" },
                      "viewport": { "type": "string" }
                    }
                  },
                  "idempotency_key": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "{ id, created_at }" },
          "400": { "description": "Validation error" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/admin/feedback": {
      "get": {
        "tags": ["admin"],
        "summary": "List product feedback submissions (plan FB0)",
        "description": "Paginated, filterable feedback queue. Requires global:app:rbac:manage.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "status", "in": "query", "schema": { "type": "string" } },
          { "name": "category", "in": "query", "schema": { "type": "string" } },
          { "name": "source", "in": "query", "schema": { "type": "string" } },
          { "name": "q", "in": "query", "schema": { "type": "string" } },
          { "name": "from", "in": "query", "schema": { "type": "string", "format": "date-time" } },
          { "name": "to", "in": "query", "schema": { "type": "string", "format": "date-time" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 25, "maximum": 100 } },
          { "name": "cursor", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ items, next_cursor?, total? }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/admin/feedback/{id}": {
      "get": {
        "tags": ["admin"],
        "summary": "Get product feedback detail (plan FB0)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "responses": {
          "200": { "description": "Full feedback record with submitter context" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found" }
        }
      },
      "patch": {
        "tags": ["admin"],
        "summary": "Update product feedback status or admin note (plan FB0)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [ { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } } ],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "status": { "type": "string", "enum": ["new", "triaged", "in_progress", "resolved", "wont_fix", "archived"] },
                  "admin_note": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Updated feedback detail" },
          "400": { "description": "Invalid input" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/settings/roles": {
      "get": {
        "tags": ["settings"],
        "summary": "List app roles and attached permissions",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "roles" }, "401": { "description": "Not signed in" }, "403": { "description": "Forbidden" } }
      },
      "post": {
        "tags": ["settings"],
        "summary": "Create an app role (empty permissions)",
        "security": [ { "bearerAuth": [] } ],
        "responses": { "200": { "description": "role" }, "400": {}, "401": {}, "403": {} }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": { "type": "http", "scheme": "bearer" }
    },
    "schemas": {
      "CatalogListing": {
        "type": "object",
        "properties": {
          "isPublic": { "type": "boolean" },
          "category": { "type": "string", "nullable": true },
          "difficultyLevel": { "type": "string", "nullable": true },
          "language": { "type": "string" },
          "priceCents": { "type": "integer" },
          "priceCurrency": { "type": "string" },
          "slug": { "type": "string" },
          "marketplaceListed": { "type": "boolean" },
          "publishState": { "type": "string", "enum": ["draft", "published"] },
          "activePurchaseCount": { "type": "integer" }
        }
      },
      "CatalogListingBody": {
        "type": "object",
        "properties": {
          "isPublic": { "type": "boolean" },
          "category": { "type": "string", "nullable": true },
          "difficultyLevel": { "type": "string", "nullable": true },
          "language": { "type": "string" },
          "priceCents": { "type": "integer" },
          "priceCurrency": { "type": "string" },
          "slug": { "type": "string" },
          "marketplaceListed": { "type": "boolean" }
        }
      }
    }
  }
}`

const docHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>StudyDrift API</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin="anonymous"></script>
<script>
  window.onload = function () {
    window.ui = SwaggerUIBundle({ url: '/api/openapi.json', dom_id: '#swagger-ui' });
  };
</script>
</body>
</html>
`

// ServeOpenAPI returns the OpenAPI JSON document.
func ServeOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(spec))
}

// ServeDocs returns HTML that loads Swagger UI against /api/openapi.json.
func ServeDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docHTML))
}
