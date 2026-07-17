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
    "description": "Lextures LMS HTTP API. Generate TypeScript types: npx openapi-typescript http://localhost:8080/api/openapi.json -o src/lib/api-types.generated.ts (with the API running). AP.9 (0.2.0): Multi-provider AI GA. Deprecated: openRouterApiKey / clearOpenRouterApiKey on /api/v1/settings/ai and openRouterConfigured on /api/v1/platform/features — use /api/v1/settings/ai/providers and aiConfigured. Dual-read of platform_app_settings.openrouter_api_key continues for ≥1 minor release; see docs/api-changelog-ai-providers.md.",
    "version": "0.2.0"
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
    { "name": "settings", "description": "Roles and permissions (server/src/routes/rbac.rs; requires global:app:rbac:manage)" },
    { "name": "transcripts", "description": "Academic transcript preview, issuance, recipient directory, and multi-destination orders (T01–T02)" }
  ],
  "paths": {
    "/api/v1/transcripts/recipients": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Search transcript recipient directory (typeahead)",
        "parameters": [
          { "name": "q", "in": "query", "schema": { "type": "string" } },
          { "name": "type", "in": "query", "schema": { "type": "string", "enum": ["institution", "application_service", "employer", "self", "other"] } }
        ],
        "responses": {
          "200": { "description": "recipients array with delivery capabilities" },
          "401": { "description": "Authentication required" },
          "404": { "description": "Transcripts feature disabled" }
        }
      }
    },
    "/api/v1/transcripts/orders": {
      "get": {
        "tags": ["transcripts"],
        "summary": "List transcript orders for the current user",
        "responses": {
          "200": { "description": "orders array" },
          "401": { "description": "Authentication required" }
        }
      },
      "post": {
        "tags": ["transcripts"],
        "summary": "Create a draft multi-recipient transcript order",
        "responses": {
          "201": { "description": "order with items" },
          "400": { "description": "Validation error (e.g. delivery method not in recipient capabilities)" },
          "401": { "description": "Authentication required" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Get a transcript order owned by the current user",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "order detail" },
          "404": { "description": "Not found or not owned by caller" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/items": {
      "post": {
        "tags": ["transcripts"],
        "summary": "Add an item to a draft transcript order",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "updated order" },
          "400": { "description": "Not draft or invalid item" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/items/{itemId}": {
      "delete": {
        "tags": ["transcripts"],
        "summary": "Remove an item from a draft transcript order",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "itemId", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "updated order" },
          "400": { "description": "Not draft or would leave order empty" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/submit": {
      "post": {
        "tags": ["transcripts"],
        "summary": "Submit a draft transcript order into the T03 lifecycle (holds/consent/payment gates)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "order in in_review, on_hold, processing, or pending_* state" },
          "400": { "description": "Validation failed" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/consent/preview": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Preview FERPA release authorization text and recipient/scope summary (T04)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "locale", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "preview object with authorizationText, recipients, requiresConsent" },
          "404": { "description": "Order not found" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/consent": {
      "post": {
        "tags": ["transcripts"],
        "summary": "Sign FERPA release authorization for a transcript order (T04)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["method", "signatureData", "agree"],
                "properties": {
                  "method": { "type": "string", "enum": ["typed", "drawn"] },
                  "signatureData": { "type": "string" },
                  "agree": { "type": "boolean" },
                  "locale": { "type": "string" },
                  "purpose": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "consent + advanced order" },
          "400": { "description": "Invalid signature or agreement" },
          "403": { "description": "Guardian required for minors" },
          "409": { "description": "Already signed or wrong state" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/consent/revoke": {
      "post": {
        "tags": ["transcripts"],
        "summary": "Revoke FERPA authorization before delivery (T04)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "revoked consent + order returned to pending_consent" },
          "409": { "description": "Already delivered or not revocable" }
        }
      }
    },
    "/api/v1/transcripts/orders/{id}/consent/export": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Export FERPA consent audit record as JSON or PDF (T04)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "format", "in": "query", "schema": { "type": "string", "enum": ["json", "pdf"] } }
        ],
        "responses": {
          "200": { "description": "export object or PDF bytes" },
          "404": { "description": "Consent not found" }
        }
      }
    },
    "/api/v1/parent/transcripts/orders/{id}/consent": {
      "post": {
        "tags": ["transcripts", "parent"],
        "summary": "Guardian e-signature for a minor student's transcript order (T04)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["method", "signatureData", "agree"],
                "properties": {
                  "method": { "type": "string", "enum": ["typed", "drawn"] },
                  "signatureData": { "type": "string" },
                  "agree": { "type": "boolean" },
                  "locale": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "guardian consent + advanced order" },
          "403": { "description": "Not a linked parent/guardian" }
        }
      }
    },
    "/api/v1/admin/transcripts/orders": {
      "get": {
        "tags": ["transcripts", "admin"],
        "summary": "Registrar fulfillment queue (filter by status, hold, q)",
        "parameters": [
          { "name": "status", "in": "query", "schema": { "type": "string" } },
          { "name": "hold", "in": "query", "schema": { "type": "string", "enum": ["true", "false"] } },
          { "name": "q", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "orders array with hold summary and events" },
          "403": { "description": "Missing RBAC" }
        }
      }
    },
    "/api/v1/admin/transcripts/orders/{id}": {
      "get": {
        "tags": ["transcripts", "admin"],
        "summary": "Get order detail for registrar fulfillment",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "order with holds and audit events" } }
      }
    },
    "/api/v1/admin/transcripts/orders/{id}/transition": {
      "post": {
        "tags": ["transcripts", "admin"],
        "summary": "Transition order (approve|reject|cancel|complete|hold|release)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["action"],
                "properties": {
                  "action": { "type": "string", "enum": ["approve", "reject", "cancel", "complete", "hold", "release"] },
                  "reason": { "type": "string", "description": "Required for reject" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "updated order" },
          "400": { "description": "Illegal transition or missing reason" }
        }
      }
    },
    "/api/v1/admin/transcripts/holds": {
      "get": {
        "tags": ["transcripts", "admin"],
        "summary": "List transcript holds",
        "parameters": [
          { "name": "userId", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "active", "in": "query", "schema": { "type": "boolean", "default": true } }
        ],
        "responses": { "200": { "description": "holds array" } }
      },
      "post": {
        "tags": ["transcripts", "admin"],
        "summary": "Place a hold on a student (blocks issuance)",
        "responses": { "201": { "description": "hold created; open orders re-evaluated" } }
      }
    },
    "/api/v1/admin/transcripts/holds/{id}/release": {
      "post": {
        "tags": ["transcripts", "admin"],
        "summary": "Release a hold and resume on-hold orders",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "released hold" } }
      }
    },
    "/api/v1/integrations/transcripts/holds": {
      "post": {
        "tags": ["transcripts", "integrations"],
        "summary": "SIS/bursar hold upsert (HMAC X-Lextures-Signature, idempotent by externalId)",
        "responses": {
          "200": { "description": "hold upserted or released" },
          "401": { "description": "Invalid signature" }
        }
      }
    },
    "/api/v1/admin/transcripts/recipients": {
      "get": {
        "tags": ["transcripts", "admin"],
        "summary": "List global + org recipient directory entries",
        "responses": { "200": { "description": "recipients array" } }
      },
      "post": {
        "tags": ["transcripts", "admin"],
        "summary": "Create an org-scoped directory recipient",
        "responses": { "201": { "description": "recipient" } }
      }
    },
    "/api/v1/admin/transcripts/recipients/{id}": {
      "put": {
        "tags": ["transcripts", "admin"],
        "summary": "Update/verify/deactivate a directory recipient",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "recipient" } }
      }
    },
    "/api/v1/transcripts/preview": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Unofficial watermarked academic-record preview (not persisted)",
        "parameters": [
          { "name": "format", "in": "query", "schema": { "type": "string", "enum": ["json", "pdf", "xml"] } }
        ],
        "responses": {
          "200": { "description": "Canonical record JSON, PDF, or PESC XML" },
          "401": { "description": "Authentication required" },
          "404": { "description": "Transcripts feature disabled" }
        }
      }
    },
    "/api/v1/transcripts/documents": {
      "get": {
        "tags": ["transcripts"],
        "summary": "List issued transcript documents for the current user",
        "responses": {
          "200": { "description": "documents array" },
          "401": { "description": "Authentication required" }
        }
      },
      "post": {
        "tags": ["transcripts"],
        "summary": "Generate and persist an official/partial/in_progress transcript",
        "responses": {
          "201": { "description": "document + record" },
          "403": { "description": "Official generation not enabled" },
          "401": { "description": "Authentication required" }
        }
      }
    },
    "/api/v1/transcripts/documents/{id}": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Get issued transcript metadata and canonical record (hash-verified)",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "document + record" },
          "409": { "description": "Integrity check failed" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/transcripts/documents/{id}/download": {
      "get": {
        "tags": ["transcripts"],
        "summary": "Download issued transcript PDF or PESC XML",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "format", "in": "query", "schema": { "type": "string", "enum": ["pdf", "xml"] } }
        ],
        "responses": {
          "200": { "description": "Binary PDF or XML" },
          "409": { "description": "Integrity check failed" }
        }
      }
    },
    "/api/v1/admin/transcripts/students/{uid}/documents": {
      "get": {
        "tags": ["transcripts", "admin"],
        "summary": "Registrar list of a student's issued transcripts",
        "parameters": [
          { "name": "uid", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "200": { "description": "documents array" } }
      },
      "post": {
        "tags": ["transcripts", "admin"],
        "summary": "Registrar generate/reissue transcript for a student",
        "parameters": [
          { "name": "uid", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": { "201": { "description": "document + record" } }
      }
    },
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
    "/api/v1/courses/{course_code}/boards": {
      "get": {
        "tags": ["courses"],
        "summary": "List collaboration boards (plan VC.1)",
        "description": "Requires per-course visualBoardsEnabled. Returns 404 when the course flag is off.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "includeArchived", "in": "query", "schema": { "type": "boolean", "default": false } }
        ],
        "responses": {
          "200": { "description": "{ boards: Board[] }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "No course access" },
          "404": { "description": "Feature disabled" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Create a collaboration board (plan VC.1 / VC.8)",
        "description": "Requires course:{code}:item:create. Optional from=template:{id} or from=board:{id}&mode=structure|full. Full copy may return 202 with a job when large.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "from", "in": "query", "schema": { "type": "string" }, "description": "template:{templateId} or board:{boardId}" },
          { "name": "mode", "in": "query", "schema": { "type": "string", "enum": ["structure", "full"] }, "description": "Copy mode when from=board:{id}" },
          { "name": "locale", "in": "query", "schema": { "type": "string" }, "description": "Locale for built-in template copy" }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "title": { "type": "string", "maxLength": 200 },
                  "description": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Board", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Board" } } } },
          "202": { "description": "{ job: BoardCopyJob } for large full copies" },
          "400": { "description": "Validation error" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Feature disabled" }
        }
      }
    },
    "/api/v1/board-templates": {
      "get": {
        "tags": ["courses"],
        "summary": "List board templates (plan VC.8)",
        "description": "Gallery of built-in, course, and org templates. Requires per-course visualBoardsEnabled.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "scope", "in": "query", "schema": { "type": "string", "enum": ["builtin", "course", "org"] } },
          { "name": "courseCode", "in": "query", "schema": { "type": "string" } },
          { "name": "q", "in": "query", "schema": { "type": "string" } },
          { "name": "locale", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ templates: BoardTemplate[] }" },
          "401": { "description": "Not signed in" },
          "404": { "description": "Feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/save-as-template": {
      "post": {
        "tags": ["courses"],
        "summary": "Save a board as a template (plan VC.8)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["scope"],
                "properties": {
                  "scope": { "type": "string", "enum": ["course", "org"] },
                  "title": { "type": "string" },
                  "description": { "type": "string" },
                  "tags": { "type": "array", "items": { "type": "string" } },
                  "includePosts": { "type": "boolean", "default": false }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "BoardTemplate" },
          "400": { "description": "Validation error" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/board-copy-jobs/{job_id}": {
      "get": {
        "tags": ["courses"],
        "summary": "Get board copy job progress (plan VC.8)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "job_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "BoardCopyJob" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/admin/boards/policies": {
      "get": {
        "tags": ["admin"],
        "summary": "Get org collaboration board policies (plan VC.10)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "orgId", "in": "query", "required": false, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "BoardOrgPolicies" },
          "403": { "description": "Forbidden" }
        }
      },
      "patch": {
        "tags": ["admin"],
        "summary": "Update org collaboration board policies (plan VC.10)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "orgId", "in": "query", "required": false, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "externalSharing": { "type": "boolean" },
                  "minorModerationFloor": { "type": "boolean" },
                  "defaultAttribution": { "type": "string", "enum": ["named", "anon_to_peers", "anonymous"] },
                  "boardCapPerCourse": { "type": "integer", "minimum": 0, "nullable": true },
                  "clearBoardCap": { "type": "boolean" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "BoardOrgPolicies" },
          "400": { "description": "Validation error" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/admin/boards/overview": {
      "get": {
        "tags": ["admin"],
        "summary": "Org collaboration boards adoption overview (plan VC.10)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "orgId", "in": "query", "required": false, "schema": { "type": "string", "format": "uuid" } },
          { "name": "activeDays", "in": "query", "required": false, "schema": { "type": "integer", "default": 30 } }
        ],
        "responses": {
          "200": { "description": "BoardAdminOverview" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/analytics": {
      "get": {
        "tags": ["courses"],
        "summary": "Board engagement analytics for managers (plan VC.10)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "days", "in": "query", "required": false, "schema": { "type": "integer", "default": 14 } }
        ],
        "responses": {
          "200": { "description": "BoardAnalyticsSummary" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/export": {
      "post": {
        "tags": ["courses"],
        "summary": "Start a board export job (plan VC.9)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["format"],
                "properties": {
                  "format": { "type": "string", "enum": ["pdf", "csv", "image"] },
                  "includeModeration": { "type": "boolean", "default": false }
                }
              }
            }
          }
        },
        "responses": {
          "202": { "description": "{ job: BoardExportJob }" },
          "400": { "description": "Validation error" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/export/{job_id}": {
      "get": {
        "tags": ["courses"],
        "summary": "Get board export job status (plan VC.9)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "job_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "BoardExportJob" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/export/{job_id}/content": {
      "get": {
        "tags": ["courses"],
        "summary": "Download a completed board export file (plan VC.9)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "job_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Export file bytes" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not ready or not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/qr": {
      "get": {
        "tags": ["courses"],
        "summary": "QR code for board quick-join (plan VC.9)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "format", "in": "query", "schema": { "type": "string", "enum": ["png", "svg"], "default": "png" } },
          { "name": "size", "in": "query", "schema": { "type": "integer", "minimum": 64, "maximum": 1024 } },
          { "name": "url", "in": "query", "schema": { "type": "string" }, "description": "Optional share/access URL on PublicWebOrigin" }
        ],
        "responses": {
          "200": { "description": "PNG or SVG QR image; X-Board-Access-Url header has the encoded URL" },
          "400": { "description": "Invalid url" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/embed": {
      "get": {
        "tags": ["courses"],
        "summary": "Embed render context for a board (plan VC.9)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ mode, board, posts, sections, capabilities }" },
          "404": { "description": "Feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits": {
      "get": {
        "tags": ["courses"],
        "summary": "List live quiz kits (plan IQ.1)",
        "description": "Requires platform ffInteractiveQuizzes and per-course interactiveQuizzesEnabled. Returns 404 when either flag is off.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "q", "in": "query", "schema": { "type": "string" }, "description": "Title search" },
          { "name": "tag", "in": "query", "schema": { "type": "string" } },
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "pageSize", "in": "query", "schema": { "type": "integer", "default": 50 } },
          { "name": "includeArchived", "in": "query", "schema": { "type": "boolean", "default": false } }
        ],
        "responses": {
          "200": { "description": "{ kits: QuizKit[], total, page, pageSize, totalPages }" },
          "401": { "description": "Not signed in" },
          "403": { "description": "No course access" },
          "404": { "description": "Feature disabled" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Create a live quiz kit (plan IQ.1)",
        "description": "Requires course:{code}:item:create. Requires platform ffInteractiveQuizzes and per-course interactiveQuizzesEnabled.",
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
                "required": ["title"],
                "properties": {
                  "title": { "type": "string", "maxLength": 200 },
                  "description": { "type": "string" },
                  "tags": { "type": "array", "items": { "type": "string" } }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "QuizKit", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/QuizKit" } } } },
          "400": { "description": "Validation error" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}": {
      "get": {
        "tags": ["courses"],
        "summary": "Get a live quiz kit (plan IQ.1)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "QuizKit", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/QuizKit" } } } },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "patch": {
        "tags": ["courses"],
        "summary": "Update a live quiz kit (plan IQ.1)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "title": { "type": "string", "maxLength": 200 },
                  "description": { "type": "string" },
                  "coverImageRef": { "type": "string", "nullable": true },
                  "status": { "type": "string", "enum": ["draft", "ready", "archived"] },
                  "visibility": { "type": "string", "enum": ["private", "course", "org", "public"] },
                  "tags": { "type": "array", "items": { "type": "string" } },
                  "archived": { "type": "boolean" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "QuizKit", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/QuizKit" } } } },
          "400": { "description": "Validation error" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/duplicate": {
      "post": {
        "tags": ["courses"],
        "summary": "Duplicate a live quiz kit (metadata-only stub, plan IQ.1)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "QuizKit", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/QuizKit" } } } },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/archive": {
      "post": {
        "tags": ["courses"],
        "summary": "Soft-archive a live quiz kit (plan IQ.1)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "QuizKit", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/QuizKit" } } } },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/restore": {
      "post": {
        "tags": ["courses"],
        "summary": "Restore an archived live quiz kit (plan IQ.1)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "QuizKit", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/QuizKit" } } } },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/validate": {
      "get": {
        "tags": ["courses"],
        "summary": "Validate a live quiz kit for hosting readiness (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ isReady, issues[] }" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/games": {
      "post": {
        "tags": ["courses"],
        "summary": "Start a live quiz game from a ready kit (plan IQ.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "{ gameId, joinCode, game }" },
          "400": { "description": "Kit not ready" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or hosting disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/games/{game_id}": {
      "get": {
        "tags": ["courses"],
        "summary": "Get a live quiz game session (plan IQ.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "game_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Live game session" },
          "404": { "description": "Not found or hosting disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/games/{game_id}/end": {
      "post": {
        "tags": ["courses"],
        "summary": "End a live quiz game (plan IQ.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "game_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Ended session" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or hosting disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/games/{game_id}/players": {
      "post": {
        "tags": ["courses"],
        "summary": "Join a live quiz game as an enrolled player (plan IQ.4); rejoin rotates playerToken",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "game_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "{ playerId, nickname, playerToken, totalScore, rejoined:false }" },
          "200": { "description": "Rejoin: { playerId, nickname, playerToken, totalScore, rejoined:true }" },
          "409": { "description": "Nickname taken" },
          "404": { "description": "Not found or hosting disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/games/{game_id}/ws": {
      "get": {
        "tags": ["courses"],
        "summary": "WebSocket hub for live quiz host/projector/player (plan IQ.3). First text frame: {authToken, role, playerToken?}",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "game_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "101": { "description": "Switching Protocols" },
          "404": { "description": "Not found or hosting disabled" }
        }
      }
    },
    "/api/v1/live-quizzes/join/{code}": {
      "get": {
        "tags": ["courses"],
        "summary": "Public rate-limited join-code lookup (plan IQ.4)",
        "parameters": [
          { "name": "code", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ gameId, courseCode, kitTitle, requiresAuth, allowsGuests, phase, status }" },
          "404": { "description": "Unknown or expired code" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/questions": {
      "get": {
        "tags": ["courses"],
        "summary": "List questions in a live quiz kit (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ questions: LiveQuizQuestion[] }" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Add a question to a live quiz kit (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "LiveQuizQuestion", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/LiveQuizQuestion" } } } },
          "400": { "description": "Validation error" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/questions/reorder": {
      "post": {
        "tags": ["courses"],
        "summary": "Bulk-reorder kit questions (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ questions: LiveQuizQuestion[] }" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/questions/import-bank": {
      "post": {
        "tags": ["courses"],
        "summary": "Import question-bank items into a kit (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "{ questions: LiveQuizQuestion[] }" },
          "400": { "description": "Validation error" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/questions/{qid}": {
      "patch": {
        "tags": ["courses"],
        "summary": "Update a kit question with If-Match version (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "qid", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "If-Match", "in": "header", "required": true, "schema": { "type": "string" }, "description": "Current question version" }
        ],
        "responses": {
          "200": { "description": "LiveQuizQuestion", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/LiveQuizQuestion" } } } },
          "409": { "description": "Version conflict" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "delete": {
        "tags": ["courses"],
        "summary": "Delete a kit question (plan IQ.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "kit_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "qid", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Deleted" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}": {
      "get": {
        "tags": ["courses"],
        "summary": "Get a collaboration board (plan VC.1)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Board", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Board" } } } },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "patch": {
        "tags": ["courses"],
        "summary": "Update a collaboration board (plans VC.1 / VC.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "title": { "type": "string", "maxLength": 200 },
                  "description": { "type": "string" },
                  "archived": { "type": "boolean" },
                  "layout": { "type": "string", "enum": ["wall", "stream", "grid", "columns", "canvas", "timeline", "map"] },
                  "layoutLocked": { "type": "boolean" },
                  "settings": { "type": "object" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Board", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Board" } } } },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "delete": {
        "tags": ["courses"],
        "summary": "Archive or hard-delete a collaboration board (plan VC.1)",
        "description": "Soft-archives by default. Pass hard=true to permanently delete (requires enrollments:update).",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "hard", "in": "query", "schema": { "type": "boolean", "default": false } }
        ],
        "responses": {
          "204": { "description": "Archived or deleted" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/sections": {
      "get": {
        "tags": ["courses"],
        "summary": "List board sections (plan VC.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ sections: BoardSection[] }" },
          "404": { "description": "Board not found or feature disabled" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Create a board section (plan VC.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["title"],
                "properties": {
                  "title": { "type": "string", "maxLength": 200 },
                  "sortIndex": { "type": "number" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "BoardSection" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Board not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/sections/{section_id}": {
      "patch": {
        "tags": ["courses"],
        "summary": "Rename or reorder a board section (plan VC.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "section_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "title": { "type": "string", "maxLength": 200 },
                  "sortIndex": { "type": "number" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "BoardSection" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Section not found" }
        }
      },
      "delete": {
        "tags": ["courses"],
        "summary": "Delete a board section; cards move to Unsorted (plan VC.3)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "section_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Deleted" },
          "400": { "description": "Cannot delete Unsorted" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Section not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/ws": {
      "get": {
        "tags": ["courses"],
        "summary": "Board realtime WebSocket (plan VC.4)",
        "description": "Upgrades to a Y.js sync WebSocket. First text frame must be JSON {\"authToken\":\"…\"}. Binary framing matches collab-docs: byte 0 = sync (persist+relay), byte 1 = awareness (relay only). Requires ffBoardsRealtime, per-course visualBoardsEnabled, enrollment, and a non-archived board. Oversized or flooding clients are disconnected.",
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "101": { "description": "Switching Protocols (WebSocket)" },
          "404": { "description": "Feature disabled, board archived, or not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts/{post_id}/arrange": {
      "patch": {
        "tags": ["courses"],
        "summary": "Arrange a board post (section, sort, position, date, geo) (plan VC.3)",
        "description": "Author or item:create. Blocked with 403 when layoutLocked for non-managers.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "sectionId": { "type": "string", "format": "uuid" },
                  "sortIndex": { "type": "number" },
                  "position": {
                    "type": "object",
                    "properties": {
                      "x": { "type": "number" },
                      "y": { "type": "number" },
                      "w": { "type": "number" },
                      "h": { "type": "number" }
                    }
                  },
                  "eventDate": { "type": "string", "format": "date-time" },
                  "lat": { "type": "number" },
                  "lng": { "type": "number" },
                  "clearGeo": { "type": "boolean" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "BoardPost" },
          "400": { "description": "Invalid arrangement fields" },
          "403": { "description": "Forbidden (ownership or layout lock)" },
          "404": { "description": "Post not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts": {
      "get": {
        "tags": ["courses"],
        "summary": "List board posts (plan VC.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ posts: BoardPost[] }" },
          "404": { "description": "Board not found or feature disabled" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Create a board post (plan VC.2)",
        "description": "Any course member may create. contentType requires matching payload (e.g. link needs linkUrl).",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["contentType"],
                "properties": {
                  "contentType": { "type": "string", "enum": ["text", "image", "file", "link", "video", "audio", "drawing"] },
                  "title": { "type": "string" },
                  "body": { "type": "object" },
                  "linkUrl": { "type": "string", "format": "uri" },
                  "drawingData": {},
                  "attachmentId": { "type": "string", "format": "uuid" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "BoardPost" },
          "400": { "description": "Validation error" },
          "404": { "description": "Board not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts/{post_id}": {
      "get": {
        "tags": ["courses"],
        "summary": "Get a board post (plan VC.2)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "BoardPost" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "patch": {
        "tags": ["courses"],
        "summary": "Update a board post (plan VC.2)",
        "description": "Author or course:{code}:item:create may edit.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "BoardPost" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "delete": {
        "tags": ["courses"],
        "summary": "Delete a board post (plan VC.2)",
        "description": "Author or course:{code}:item:create may delete.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Deleted" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts/{post_id}/reaction": {
      "put": {
        "tags": ["courses"],
        "summary": "Set or toggle a board post reaction (plan VC.5)",
        "description": "Idempotent toggle for like/vote; set/update for star/grade. Grade requires item:create.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "kind": { "type": "string", "enum": ["like", "vote", "star", "grade"] },
                  "value": { "type": "number", "description": "Required for star (1-5) and grade" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Updated aggregates" },
          "400": { "description": "Invalid kind/value or reactions disabled" },
          "403": { "description": "Forbidden (grade without permission)" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "delete": {
        "tags": ["courses"],
        "summary": "Clear the viewer's reaction on a board post (plan VC.5)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Cleared" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts/{post_id}/comments": {
      "get": {
        "tags": ["courses"],
        "summary": "List comments on a board post (plan VC.5)",
        "description": "Managers see hidden comments; students do not.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ comments: BoardComment[] }" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Create a comment on a board post (plan VC.5)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["body"],
                "properties": {
                  "body": { "type": "object" },
                  "parentId": { "type": "string", "format": "uuid" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "BoardComment" },
          "400": { "description": "Validation error" },
          "429": { "description": "Rate limited" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts/{post_id}/comments/{comment_id}": {
      "patch": {
        "tags": ["courses"],
        "summary": "Update or hide a board comment (plan VC.5)",
        "description": "Author may edit body; item:create may hide any comment.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "comment_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "BoardComment" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      },
      "delete": {
        "tags": ["courses"],
        "summary": "Soft-hide a board comment (plan VC.5)",
        "description": "Author or item:create. Soft-hide preserves the row for audit/FERPA.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "comment_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Hidden" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/posts/{post_id}/grade-sync": {
      "post": {
        "tags": ["courses"],
        "summary": "Sync a card grade to the gradebook (plan VC.5)",
        "description": "Requires grade reaction mode, linked assignmentId, and item:create.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "post_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Synced grade summary" },
          "400": { "description": "Missing grade, mode, or assignment link" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/attachments": {
      "post": {
        "tags": ["courses"],
        "summary": "Upload or initiate a board post attachment (plan VC.2)",
        "description": "multipart/form-data with file, or JSON for presigned PUT init.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "BoardAttachment" },
          "400": { "description": "Validation error" },
          "404": { "description": "Board not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/attachments/{attachment_id}/content": {
      "get": {
        "tags": ["courses"],
        "summary": "Download board attachment content (plan VC.2)",
        "description": "Returns 403 when scan_status is pending/blocked and AV scanning is enabled.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "attachment_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "File bytes" },
          "302": { "description": "Presigned redirect" },
          "403": { "description": "Scanning or blocked" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/link-preview": {
      "post": {
        "tags": ["courses"],
        "summary": "Unfurl a URL for a board link/video post (plan VC.2)",
        "description": "SSRF-safe server-side fetch. Private/loopback URLs return 400.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["url"],
                "properties": { "url": { "type": "string", "format": "uri" } }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Link preview payload" },
          "400": { "description": "Invalid or blocked URL" },
          "404": { "description": "Board not found or feature disabled" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/members": {
      "get": {
        "tags": ["courses"],
        "summary": "List board members (plan VC.6)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ members: BoardMember[] }" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Board not found" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Add or update a board member (plan VC.6)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["userId"],
                "properties": {
                  "userId": { "type": "string", "format": "uuid" },
                  "role": { "type": "string", "enum": ["owner", "editor", "contributor", "viewer"] }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "BoardMember" },
          "400": { "description": "Invalid input" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/members/{user_id}": {
      "delete": {
        "tags": ["courses"],
        "summary": "Remove a board member (plan VC.6)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "user_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Removed" },
          "404": { "description": "Member not found" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/shares": {
      "get": {
        "tags": ["courses"],
        "summary": "List board share links (plan VC.6)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "{ shares: BoardShare[] }" },
          "403": { "description": "Forbidden" }
        }
      },
      "post": {
        "tags": ["courses"],
        "summary": "Create a board share link (plan VC.6)",
        "description": "Requires ffBoardsExternalSharing. Returns the raw token once.",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "capability": { "type": "string", "enum": ["view", "contribute"] },
                  "password": { "type": "string" },
                  "expiresAt": { "type": "string", "format": "date-time", "nullable": true }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "BoardShare including token and url" },
          "403": { "description": "External sharing disabled or minors policy" }
        }
      }
    },
    "/api/v1/courses/{course_code}/boards/{board_id}/shares/{share_id}": {
      "delete": {
        "tags": ["courses"],
        "summary": "Revoke a board share link (plan VC.6)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "course_code", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "board_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "share_id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "204": { "description": "Revoked" },
          "404": { "description": "Share not found" }
        }
      }
    },
    "/api/v1/board-links/{token}": {
      "get": {
        "tags": ["public"],
        "summary": "Resolve a board share link (plan VC.6)",
        "description": "Unauthenticated. Optional X-Board-Share-Password header for password-protected links.",
        "parameters": [
          { "name": "token", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "{ board, capability, posts, requiresPassword }" },
          "401": { "description": "Incorrect password" },
          "403": { "description": "External sharing disabled" },
          "404": { "description": "Invalid, expired, or revoked" }
        }
      }
    },
    "/api/v1/board-links/{token}/posts": {
      "post": {
        "tags": ["public"],
        "summary": "Create a post via contribute share link (plan VC.6)",
        "parameters": [
          { "name": "token", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["displayName", "contentType"],
                "properties": {
                  "displayName": { "type": "string" },
                  "contentType": { "type": "string" },
                  "title": { "type": "string" },
                  "body": { "type": "object" },
                  "linkUrl": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Created BoardPost" },
          "403": { "description": "Link is view-only" }
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
    "/api/v1/public/ai-disclosure": {
      "get": {
        "tags": ["meta"],
        "summary": "Public AI usage disclosure document (configured providers + models)",
        "responses": {
          "200": {
            "description": "PublicDisclosure JSON: version, provider, providers[], models[], features[]"
          }
        }
      }
    },
    "/api/v1/settings/ai": {
      "get": {
        "tags": ["settings"],
        "summary": "Get Intelligence AI model settings",
        "description": "Returns feature model ids and activeProvider. Response field openRouterApiKey is deprecated (AP.9); use GET /api/v1/settings/ai/providers.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": {
            "description": "AI settings JSON",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "imageModelId": { "type": "string" },
                    "courseSetupModelId": { "type": "string" },
                    "notebookFlashcardsModelId": { "type": "string" },
                    "vibeActivityModelId": { "type": "string" },
                    "graderAgentModelId": { "type": "string" },
                    "activeProvider": { "type": "string" },
                    "openRouterApiKey": { "type": "string", "deprecated": true, "description": "Masked legacy OpenRouter key; prefer /settings/ai/providers" }
                  }
                }
              }
            }
          },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
        }
      },
      "put": {
        "tags": ["settings"],
        "summary": "Update Intelligence AI model settings",
        "description": "Request fields openRouterApiKey and clearOpenRouterApiKey are deprecated (AP.9); prefer PUT/DELETE /api/v1/settings/ai/providers/{provider}.",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "Updated AI settings JSON" },
          "400": { "description": "Invalid input" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
        }
      }
    },
    "/api/v1/settings/ai/providers": {
      "get": {
        "tags": ["settings"],
        "summary": "List platform AI provider credentials (masked)",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": { "description": "credentials[], providers[], tenant BYOK policy" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" },
          "404": { "description": "Abstraction disabled (rollback)" }
        }
      }
    },
    "/api/v1/platform/features": {
      "get": {
        "tags": ["meta"],
        "summary": "Runtime platform feature flags for the signed-in user",
        "description": "aiConfigured and aiProvidersConfigured[] are authoritative for AI availability. openRouterConfigured is a deprecated alias of aiConfigured (AP.9).",
        "security": [ { "bearerAuth": [] } ],
        "responses": {
          "200": {
            "description": "Feature flags JSON",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "aiConfigured": { "type": "boolean" },
                    "aiProvidersConfigured": { "type": "array", "items": { "type": "string" } },
                    "aiProviderAbstractionEnabled": { "type": "boolean" },
                    "openRouterConfigured": { "type": "boolean", "deprecated": true, "description": "Deprecated alias of aiConfigured" }
                  }
                }
              }
            }
          },
          "401": { "description": "Not signed in" }
        }
      }
    },
    "/api/v1/settings/ai/reports": {
      "get": {
        "tags": ["settings"],
        "summary": "Intelligence AI usage and cost reports (multi-provider)",
        "security": [ { "bearerAuth": [] } ],
        "parameters": [
          { "name": "from", "in": "query", "schema": { "type": "string", "format": "date-time" } },
          { "name": "to", "in": "query", "schema": { "type": "string", "format": "date-time" } },
          { "name": "feature", "in": "query", "schema": { "type": "string" } },
          { "name": "provider", "in": "query", "schema": { "type": "string" }, "description": "Filter by AI provider (openrouter, anthropic, openai, azure_openai, bedrock, vertex)" },
          { "name": "userQuery", "in": "query", "schema": { "type": "string" } },
          { "name": "courseCode", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "ReportsPayload with cost.byProvider and providers[]" },
          "401": { "description": "Not signed in" },
          "403": { "description": "Forbidden" }
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
      },
      "QuizKit": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "courseId": { "type": "string", "format": "uuid" },
          "title": { "type": "string" },
          "description": { "type": "string" },
          "slug": { "type": "string" },
          "coverImageRef": { "type": "string", "nullable": true },
          "status": { "type": "string", "enum": ["draft", "ready", "archived"] },
          "visibility": { "type": "string", "enum": ["private", "course", "org", "public"] },
          "tags": { "type": "array", "items": { "type": "string" } },
          "questionCount": { "type": "integer" },
          "archived": { "type": "boolean" },
          "createdBy": { "type": "string", "format": "uuid", "nullable": true },
          "createdAt": { "type": "string", "format": "date-time" },
          "updatedAt": { "type": "string", "format": "date-time" }
        }
      },
      "LiveQuizQuestion": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "kitId": { "type": "string", "format": "uuid" },
          "position": { "type": "integer" },
          "questionType": {
            "type": "string",
            "enum": ["mc_single", "mc_multiple", "true_false", "type_answer", "numeric", "poll", "ordering", "word_cloud"]
          },
          "prompt": { "type": "string" },
          "promptMediaRef": { "type": "string", "nullable": true },
          "promptMediaAlt": { "type": "string", "nullable": true },
          "options": { "type": "array", "items": { "type": "object" } },
          "correctAnswer": { "type": "object", "nullable": true },
          "timeLimitSeconds": { "type": "integer", "minimum": 5, "maximum": 240 },
          "pointsStyle": { "type": "string", "enum": ["standard", "double", "no_points"] },
          "answerShuffle": { "type": "boolean" },
          "explanation": { "type": "string", "nullable": true },
          "sourceQuestionId": { "type": "string", "format": "uuid", "nullable": true },
          "version": { "type": "integer" },
          "createdAt": { "type": "string", "format": "date-time" },
          "updatedAt": { "type": "string", "format": "date-time" }
        }
      },
      "Board": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "courseId": { "type": "string", "format": "uuid" },
          "title": { "type": "string" },
          "description": { "type": "string" },
          "slug": { "type": "string" },
          "archived": { "type": "boolean" },
          "layout": { "type": "string", "enum": ["wall", "stream", "grid", "columns", "canvas", "timeline", "map"] },
          "layoutLocked": { "type": "boolean" },
          "settings": { "type": "object" },
          "reactionMode": { "type": "string", "enum": ["none", "like", "vote", "star", "grade"] },
          "assignmentId": { "type": "string", "format": "uuid", "nullable": true },
          "visibility": { "type": "string", "enum": ["course", "section", "group", "invite", "link", "public"] },
          "visibilityTarget": { "type": "string", "format": "uuid", "nullable": true },
          "attribution": { "type": "string", "enum": ["named", "anon_to_peers", "anonymous"] },
          "canPost": { "type": "boolean" },
          "canInteract": { "type": "boolean" },
          "canArrange": { "type": "boolean" },
          "capabilities": {
            "type": "object",
            "properties": {
              "canView": { "type": "boolean" },
              "canPost": { "type": "boolean" },
              "canInteract": { "type": "boolean" },
              "canArrange": { "type": "boolean" },
              "canManage": { "type": "boolean" }
            }
          },
          "createdBy": { "type": "string", "format": "uuid", "nullable": true },
          "createdAt": { "type": "string", "format": "date-time" },
          "updatedAt": { "type": "string", "format": "date-time" }
        }
      },
      "BoardMember": {
        "type": "object",
        "properties": {
          "boardId": { "type": "string", "format": "uuid" },
          "userId": { "type": "string", "format": "uuid" },
          "role": { "type": "string", "enum": ["owner", "editor", "contributor", "viewer"] },
          "createdAt": { "type": "string", "format": "date-time" }
        }
      },
      "BoardShare": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "boardId": { "type": "string", "format": "uuid" },
          "capability": { "type": "string", "enum": ["view", "contribute"] },
          "hasPassword": { "type": "boolean" },
          "expiresAt": { "type": "string", "format": "date-time", "nullable": true },
          "revokedAt": { "type": "string", "format": "date-time", "nullable": true },
          "createdBy": { "type": "string", "format": "uuid" },
          "createdAt": { "type": "string", "format": "date-time" },
          "token": { "type": "string", "description": "Raw token; only returned on create" },
          "url": { "type": "string" }
        }
      },
      "BoardSection": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "boardId": { "type": "string", "format": "uuid" },
          "title": { "type": "string" },
          "sortIndex": { "type": "number" },
          "createdAt": { "type": "string", "format": "date-time" }
        }
      },
      "BoardPost": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "boardId": { "type": "string", "format": "uuid" },
          "authorId": { "type": "string", "format": "uuid", "nullable": true },
          "guestDisplayName": { "type": "string" },
          "contentType": { "type": "string", "enum": ["text", "image", "file", "link", "video", "audio", "drawing"] },
          "title": { "type": "string" },
          "body": { "type": "object" },
          "linkUrl": { "type": "string" },
          "linkPreview": { "type": "object" },
          "drawingData": {},
          "attachment": { "$ref": "#/components/schemas/BoardAttachment" },
          "sectionId": { "type": "string", "format": "uuid" },
          "sortIndex": { "type": "number" },
          "position": {
            "type": "object",
            "properties": {
              "x": { "type": "number" },
              "y": { "type": "number" },
              "w": { "type": "number" },
              "h": { "type": "number" }
            }
          },
          "eventDate": { "type": "string", "format": "date-time" },
          "lat": { "type": "number" },
          "lng": { "type": "number" },
          "reactionCount": { "type": "integer" },
          "myReaction": {
            "type": "object",
            "properties": {
              "kind": { "type": "string" },
              "value": { "type": "number", "nullable": true }
            }
          },
          "avgStars": { "type": "number" },
          "commentCount": { "type": "integer" },
          "grade": { "type": "number", "description": "Visible only to card author and graders (plan VC.5)" },
          "createdAt": { "type": "string", "format": "date-time" },
          "updatedAt": { "type": "string", "format": "date-time" }
        }
      },
      "BoardComment": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "postId": { "type": "string", "format": "uuid" },
          "parentId": { "type": "string", "format": "uuid", "nullable": true },
          "authorId": { "type": "string", "format": "uuid", "nullable": true },
          "body": { "type": "object" },
          "hidden": { "type": "boolean" },
          "createdAt": { "type": "string", "format": "date-time" },
          "updatedAt": { "type": "string", "format": "date-time" }
        }
      },
      "BoardAttachment": {
        "type": "object",
        "properties": {
          "id": { "type": "string", "format": "uuid" },
          "url": { "type": "string", "nullable": true },
          "fileName": { "type": "string" },
          "mimeType": { "type": "string" },
          "sizeBytes": { "type": "integer" },
          "altText": { "type": "string" },
          "scanStatus": { "type": "string", "enum": ["pending", "clean", "blocked"] }
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
