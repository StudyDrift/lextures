# Standards & Legal Hardening Plans

> Goal: make Lextures **bullet-proof** against the education-and-privacy legal regimes of every market we
> sell into, and keep it that way. These plans **harden and extend** the already-shipped compliance layer
> (`docs/completed/10-compliance-privacy-security/`) — they are not a rewrite of it.

## Why this folder exists

The `10.x` compliance plans (FERPA, COPPA, GDPR/UK-GDPR, CCPA/CPRA, SDPC DPA, US state laws, WCAG, VPAT,
SOC 2, ISO 27001/27701, audit log, data residency, encryption, PII redaction, backups, bug bounty, AI
disclosure) shipped a working baseline. This folder closes the **residual gaps** that a hostile auditor,
a regulator, or an adversarial procurement/RFP process would find, and adds the **jurisdictions and
statutes not yet covered at all** (EU AI Act, PPRA, Canada/Quebec, Australia/NZ, Brazil, India, China,
APAC, Africa, and the accessibility *laws* — not just WCAG the guideline).

Two documented deferrals in the shipped baseline are explicitly picked up here:

- GDPR plan `10.3` deferred **cross-border transfer mechanisms** (SCCs/BCRs) as "legal only, not
  engineering" → hardened by **[S07](S07-cross-border-transfer-subprocessor-governance.md)**.
- GDPR plan `10.3` deferred the **cookie-consent banner** to "a separate UI ticket" → delivered by
  **[S04](S04-unified-consent-preference-ledger.md)** + **[S07](S07-cross-border-transfer-subprocessor-governance.md)**.

## Conventions

- **File naming:** `S{NN}-{kebab-slug}.md` (the `S` = *Standards*, mirroring `W##`/`C##`/`M##`).
- Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md). Because `docs/MISSING_FEATURES.md` was retired,
  the template's "Source" line points instead at the completed compliance set it hardens.
- A plan is **ready** when every template section is filled (no `…` placeholders).
- New DB migrations continue the repo's global sequence — the highest existing is `356_*`, so these plans
  reserve `357_*` onward (each plan states its number). Renumber on merge if the sequence has advanced.
- Compliance services live in `server/internal/service/<name>/`, repos in `server/internal/repos/<name>/`,
  HTTP handlers in `server/internal/httpserver/<name>_http.go` under `/api/v1/compliance/*`.

## Severity legend

- **BLOCKER** — we are unlawful / uninsurable / disqualified from procurement in the named market until fixed.
- **MAJOR** — RFP-losing or material fine/consent-order exposure.
- **MINOR** — parity, defence-in-depth, or auditor comfort.

## Story index

### Cross-cutting compliance engines (jurisdiction-agnostic backbone)

| ID | Plan | Severity | Markets |
|---|---|---|---|
| S01 | [Unified Data-Subject Rights (DSAR) orchestration](S01-unified-data-subject-rights-orchestration.md) | BLOCKER | Global |
| S02 | [Data retention & deletion schedule engine](S02-data-retention-deletion-engine.md) | BLOCKER | Global |
| S03 | [Global data-breach notification & incident response](S03-global-breach-notification-incident-response.md) | BLOCKER | Global |
| S04 | [Unified consent & preference management ledger](S04-unified-consent-preference-ledger.md) | BLOCKER | Global |
| S05 | [Records of Processing Activities & live data inventory](S05-ropa-data-inventory-mapping.md) | MAJOR | EU/UK · Global |
| S06 | [DPIA / PIA & algorithmic impact assessment automation](S06-dpia-pia-algorithmic-impact.md) | MAJOR | Global |
| S07 | [Cross-border transfer & subprocessor governance](S07-cross-border-transfer-subprocessor-governance.md) | BLOCKER | EU/UK · Global |
| S08 | [Children's privacy, age assurance & design codes](S08-childrens-privacy-age-assurance-design-codes.md) | BLOCKER | Global |

### United States

| ID | Plan | Severity | Markets |
|---|---|---|---|
| S09 | [FERPA hardening (deep 34 CFR Part 99)](S09-ferpa-hardening.md) | BLOCKER | K12 · HE |
| S10 | [PPRA — Protection of Pupil Rights Amendment](S10-ppra-pupil-rights.md) | BLOCKER | K12 |
| S11 | [US state privacy-law coverage expansion](S11-us-state-privacy-expansion.md) | MAJOR | K12 · SL · HE |

### European Union & United Kingdom

| ID | Plan | Severity | Markets |
|---|---|---|---|
| S12 | [GDPR / UK GDPR / Swiss FADP accountability hardening](S12-gdpr-uk-swiss-accountability-hardening.md) | BLOCKER | EU/UK |
| S13 | [EU AI Act — education as high-risk AI](S13-eu-ai-act-high-risk.md) | BLOCKER | EU |

### Rest of world

| ID | Plan | Severity | Markets |
|---|---|---|---|
| S14 | [Canada — PIPEDA + Quebec Law 25 + provincial PIPA](S14-canada-pipeda-quebec-law25.md) | MAJOR | CA |
| S15 | [Australia (Privacy Act/APPs/NDB) + New Zealand](S15-australia-nz-privacy.md) | MAJOR | AU · NZ |
| S16 | [Brazil LGPD (+ LATAM)](S16-brazil-lgpd-latam.md) | MAJOR | BR · LATAM |
| S17 | [India DPDP Act 2023](S17-india-dpdp.md) | MAJOR | IN |
| S18 | [China PIPL & data-localization](S18-china-pipl-localization.md) | MAJOR | CN |
| S19 | [APAC & Africa (APPI · PIPA · POPIA · NDPA)](S19-apac-africa-privacy.md) | MINOR | JP · KR · ZA · NG |

### Accessibility law & compliance program

| ID | Plan | Severity | Markets |
|---|---|---|---|
| S20 | [Accessibility legal mandates (ADA Title II/III · §508 · EAA/EN 301 549 · AODA)](S20-accessibility-legal-mandates.md) | BLOCKER | US · EU · CA |
| S21 | [Compliance evidence, continuous control monitoring & audit readiness](S21-compliance-evidence-continuous-monitoring.md) | MAJOR | Global |

---

## Master coverage matrix

Every regime we could be held to. **Baseline** = shipped in `docs/completed/10-*`. **Hardened by** =
this folder. Nothing in this table is left without an owning plan — that is the point of the folder.

### United States — federal

| Law / standard | Citation | Baseline | Hardened / added |
|---|---|---|---|
| FERPA | 20 U.S.C. §1232g; 34 CFR Part 99 | 10.1 | **S09** (deep §99: annual notice, §99.32 access record, §99.33 redisclosure, §99.31/§99.36 exceptions, de-identification) |
| PPRA | 20 U.S.C. §1232h | — | **S10** (new) |
| COPPA | 15 U.S.C. §6501; 16 CFR Part 312 (2025 amended rule) | 10.2 | **S08** (VPC hardening, data-retention limits, 2025 rule deltas) |
| CIPA | 47 U.S.C. §254(h) | — | **S11** (filtering/monitoring attestations) |
| ADA Title II (2024 web rule) | 28 CFR Part 35 | 10.7 (WCAG) | **S20** (legal deadlines, conformance attestations) |
| ADA Title III / §508 | 42 U.S.C. §12181; 29 U.S.C. §794d | 10.7, 10.8 | **S20** |

### United States — state

| Law / standard | Baseline | Hardened / added |
|---|---|---|
| CA SOPIPA, NY Ed Law §2-d, IL SOPPA | 10.6 | **S11** (extend + attestations) |
| CCPA / CPRA | 10.4 | **S01** (rights), **S04** (opt-out signals/GPC), **S11** |
| Comprehensive state acts (VA, CO, CT, UT, TX, OR, MT, FL, and 15+ more) | — | **S11** (new) |
| State kids'-code / AADC (CA AB 2273, and successors) | — | **S08** |
| SDPC / national DPA template | 10.5 | **S07** (subprocessor lifecycle), **S21** |

### European Union & United Kingdom

| Law / standard | Baseline | Hardened / added |
|---|---|---|
| GDPR / UK GDPR | 10.3 | **S12** (accountability, Art 22, DPO, lawful-basis registry), **S01**, **S05** |
| ePrivacy / cookies (PECR) | — | **S04** (consent banner + ledger) |
| International transfers (SCCs, UK IDTA, EU-US DPF, TIAs) | 10.3 *(deferred)* | **S07** (new engineering deliverable) |
| EU AI Act | — | **S13** (education = Annex III high-risk) |
| UK Age Appropriate Design Code; Ireland "Fundamentals" | — | **S08** |
| European Accessibility Act; EN 301 549 | 10.7 | **S20** |
| Swiss revFADP | — | **S12** |

### Rest of world

| Law / standard | Baseline | Hardened / added |
|---|---|---|
| Canada PIPEDA; Quebec Law 25; BC/AB PIPA | — | **S14** (new) |
| Australia Privacy Act 1988 / APPs / NDB scheme | — | **S15** (new) |
| New Zealand Privacy Act 2020 | — | **S15** |
| Brazil LGPD | — | **S16** (new) |
| India DPDP Act 2023 | — | **S17** (new) |
| China PIPL / DSL / CSL (+ localization) | — | **S18** (new) |
| Japan APPI; South Korea PIPA | — | **S19** (new) |
| South Africa POPIA; Nigeria NDPA | — | **S19** |
| Ontario AODA | — | **S20** |

### Cross-cutting obligations (appear in most regimes above)

| Obligation | Owning plan |
|---|---|
| One front door for access/erasure/portability/correction/objection | **S01** |
| Statutory retention + deletion + legal hold | **S02** |
| 72-hour (and equivalent) breach notification | **S03** |
| Lawful-basis / consent / opt-out ledger incl. Global Privacy Control | **S04** |
| Article 30 RoPA + live personal-data inventory | **S05** |
| DPIA / PIA / algorithmic-impact assessments | **S06** |
| Transfer mechanisms + subprocessor register + flow-downs | **S07** |
| Children / age assurance / design codes | **S08** |
| Continuous control monitoring + evidence for auditors | **S21** |

## Sequencing at a glance

```
S01 DSAR ──┐
S02 Retention ─┤
S03 Breach ────┼─► every jurisdiction plan (S09–S19) consumes these engines
S04 Consent ───┤
S05 RoPA ──────┤
S07 Transfer ──┘
S06 DPIA ──► S13 (AI Act) ──► S08 (kids' AI)
S20 Accessibility law  ─ independent
S21 Evidence ─ depends on all (aggregates their control state)
```
