# Transcripts & Credentials Platform (T01–T12)

Turn Lextures' thin transcript-*request* feature into a full transcript, credential, and
delivery network in the spirit of [Parchment](https://www.parchment.com/) (order → fulfill →
deliver → verify), built on top of the academic record, gradebook, enrollments, billing, and
verifiable-credential infrastructure that already ships.

## What exists today (baseline)

| Area | State | Where |
|---|---|---|
| Transcript **generation** (T01) | DONE — canonical academic record → immutable PDF + PESC XML; unofficial preview; official sealing behind `official_enabled` | `service/academicrecord`, `service/transcriptissue`, `service/transcriptpdf`, `service/transcriptpesc`, migration `386`, [completed T01](../../completed/transcripts/T01-official-transcript-generation.md) |
| Recipient directory & **orders** (T02) | DONE — searchable directory; multi-item orders; legacy request backfill + deprecated proxy; order UI behind `orders_ui_enabled` | migration `388`, `repos/transcripts/{recipients,orders}.go`, `transcripts_orders_http.go`, [completed T02](../../completed/transcripts/T02-recipient-directory-and-orders.md) |
| Order **lifecycle / holds** (T03) | DONE — state machine; financial/registrar holds; registrar queue; auto-approval; SIS hold webhook; `order_events` audit | migration `393`, `models/transcriptorder`, `repos/transcripts/{holds,lifecycle}.go`, `transcripts_lifecycle_http.go`, `/admin/transcripts`, [completed T03](../../completed/transcripts/T03-order-lifecycle-fulfillment-holds.md) |
| FERPA **consent / e-signature** (T04) | DONE — signed release per order; hard gate at `pending_consent`; guardian path for minors; revoke + JSON/PDF export | migration `395`, `service/transcriptconsent`, `repos/transcripts/consents.go`, `transcripts_consent_http.go`, [completed T04](../../completed/transcripts/T04-ferpa-consent-esignature.md) |
| Transcript **request** (legacy) | DEPRECATED — still accepted; proxies to a one-item order and fires the institution webhook | `transcripts_http.go`, migrations `263`–`265` |
| Delivery options | Order methods: electronic_pesc / electronic_pdf / secure_link_email / postal_mail / api_peer; legacy email/mail/pickup still mapped | migration `387`, `264`, `265` |
| Admin config | webhook URL + secret + pickup + `official_enabled`; feature flag `ff_transcripts` | `settings.transcripts_config`, `platformconfig` |
| Co-Curricular Record / CLR | Achievement aggregation, W3C Verifiable Credential signing, public verify + DID | `ccr` schema, `service/ccr`, `service/vc_signing`, migration `259`, plan [14.13](../../completed/14-higher-ed-specific/14.13-co-curricular-transcript.md) |
| Mastery / SBG transcript PDF | PDF renderer scaffold (academic PDFs live in `transcriptpdf`) | `server/internal/service/masterytranscriptpdf` |
| Seat-time / CE transcript | seat-time records + CE transcript page | `seattime` repo, `pages/lms/CeTranscript.tsx` |

The remaining gap: Lextures can **produce** sealed transcripts, **take multi-recipient orders**, **gate fulfillment with holds/review**, and **capture FERPA e-signature consent** (T01–T04) but cannot yet fully **price, deliver,
receive, or verify** them through the fulfillment network (T05–T12).

## Stories

| ID | Plan | Severity | Markets | One-line |
|---|---|---|---|---|
| T01 | [Official academic transcript generation](../../completed/transcripts/T01-official-transcript-generation.md) ✅ | BLOCKER | HE · K12 | Render the canonical academic record → immutable PDF + PESC XML from enrollments/grades |
| T02 | [Recipient directory & multi-destination orders](../../completed/transcripts/T02-recipient-directory-and-orders.md) ✅ | BLOCKER | HE · K12 · SL | Order model + searchable receiver network (schools, employers, app services, self) |
| T03 | [Order lifecycle, registrar fulfillment & holds](../../completed/transcripts/T03-order-lifecycle-fulfillment-holds.md) ✅ | BLOCKER | HE · K12 | Review/approve/hold/fulfill workflow + financial/registrar holds that block issuance |
| T04 | [FERPA consent & e-signature authorization](../../completed/transcripts/T04-ferpa-consent-esignature.md) ✅ | BLOCKER | HE · K12 | Signed, scoped, auditable release authorization before records leave the institution |
| T05 | [Transcript fees, payments & waivers](T05-fees-payments-waivers.md) | MAJOR | HE · SL | Per-order/per-recipient/rush fees, fee waivers, refunds via existing Stripe billing |
| T06 | [Electronic delivery & interoperability standards](T06-electronic-delivery-standards.md) | BLOCKER | HE · K12 | PESC XML / EDI TS130 / SPEEDE / signed-PDF adapters; secure links; delivery receipts |
| T07 | [Inbound receiving & transfer-credit intake](T07-inbound-receiving-transfer-credit.md) | MAJOR | HE | Receive transcripts *from* other schools, parse PESC, match to applicant |
| T08 | [Credential verification & tamper-evidence](T08-credential-verification.md) | MAJOR | HE · K12 · SL | Digitally signed, QR-verifiable documents + third-party verifier portal |
| T09 | [Learner credential wallet](T09-learner-credential-wallet.md) | MAJOR | SL · HE · K12 | One learner-owned home for transcripts, diplomas, certificates, badges, CLR — portable |
| T10 | [Order tracking & notifications](T10-order-tracking-notifications.md) | MINOR | HE · K12 · SL | Real-time status, delivery/open receipts, resend/cancel, email + push at each step |
| T11 | [Diploma & digital certificate issuance](T11-diploma-certificate-issuance.md) | MINOR | HE · K12 | Institution-issued verifiable diplomas/certificates into the wallet |
| T12 | [Registrar console & transcript analytics](T12-registrar-console-analytics.md) | MAJOR | HE · K12 | Registrar queue/holds/fees UI + destination, volume, revenue, and SLA analytics |

## Shared data model

All stories extend the existing `transcripts` schema and add a small `credentials` schema.
The **owning** story defines each table; others reference it.

```
transcripts.transcript_documents   -- T01  immutable issued academic-record artifacts (PDF + PESC XML + canonical JSON + hash)
transcripts.recipients             -- T02  receiver directory (institution/employer/app-service/self/other)
transcripts.orders                 -- T02  an order (replaces the single-request row)
transcripts.order_items            -- T02  one recipient × one document per row
transcripts.holds                  -- T03  financial/registrar/disciplinary holds
transcripts.consents               -- T04  signed FERPA release authorizations
transcripts.fee_schedule           -- T05  per-org fee config + waiver rules
transcripts.delivery_attempts      -- T06  per-adapter delivery attempts + receipts
transcripts.share_links            -- T06  secure, expiring recipient download links
transcripts.inbound_documents      -- T07  transcripts received from other institutions
transcripts.verifications          -- T08  third-party verification lookups
credentials.wallet_items           -- T09  unified learner credential index (view + overrides)
credentials.diplomas               -- T11  issued diplomas/certificates
```

The legacy `transcripts.transcript_requests` table (migrations 263–265) is migrated into
`orders` + `order_items` in T02 and then deprecated (kept read-only for one release).

## Sequencing

```
T01 ─┬─> T02 ─┬─> T03 ─┬─> T05 ──┐
     │        │        └─> T04 ──┤
     │        └─> T06 ───────────┼─> T10
     │                           │
     └─> T08 ─> T09 <─ T11       │
                  T07 ───────────┘   (T07 independent; needs T06 parser)
                  T12  (needs T02/T03/T05 data)
```

- **Phase 1 (foundation):** T01 (generation), T02 (order + recipients), T04 (consent).
- **Phase 2 (fulfillment):** T03 (workflow + holds), T06 (delivery/standards), T05 (payments).
- **Phase 3 (network & trust):** T08 (verification), T09 (wallet), T10 (tracking), T07 (receiving).
- **Phase 4 (breadth):** T11 (diplomas), T12 (console + analytics).

## Feature flags

- Reuse `ff_transcripts` as the master gate for the ordering/delivery surface (T01–T06, T10, T12).
- Reuse `ff_co_curricular_transcript` for the CLR/wallet overlap (T09).
- Add `ff_transcript_inbound` (T07) and `ff_diplomas` (T11) so receiving and issuance ship dark.

## Conventions

- File naming here uses the `T{NN}-{slug}.md` prefix, paralleling `standards/S{NN}` and `ai-providers/AP.N`.
- Migration numbers below are **indicative** (`378`+ was the next free number when this plan was written);
  use the next available `NNN` at implementation time.
- Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md); a plan is "ready" when no `…` placeholders remain.
