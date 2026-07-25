# DPIA / Algorithmic Impact Assessment — Adaptive Content Engine (ACE)

> Living artifact for AC.8 / Standards S06. Keep current with prompt and model version changes.

| Field | Value |
|---|---|
| **Feature** | Adaptive Content Engine (`adaptive_content`) |
| **Prompt version** | `v1` (`settings.system_prompts` key `adaptive_content_variant`) |
| **Assessment date** | 2026-07-25 |
| **Owner** | Trust & safety + DPO |
| **Status** | Complete for GA readiness (AC.8 AC-7) |

## 1. Processing description

ACE generates per-learner variants of course content pages using generative AI, keyed to an adaptation profile (emphasis mode, concept gaps, misconceptions, reading/modality preferences). Variants are fidelity/safety/a11y gated, optionally instructor-approved, served transparently with opt-out and contest paths, and measured for effectiveness against a holdout.

## 2. Data categories

| Category | Examples | Lawful basis (typical) |
|---|---|---|
| Education records | profiles, servings, outcomes, contests, opt-outs | Legitimate interest / public task (education); FERPA school official |
| AI inference inputs | base content, profile signature, axis set | Same; disclosed via AI disclosure |
| Proxy fairness attrs | locale, section, accommodation flag | Legitimate interest for equity monitoring; aggregated + small-cell suppressed |
| Minors | COPPA-gated accounts | Parental consent + conservative defaults; no profiling/generation without consent |

Demographics are **never** sent to the model.

## 3. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Hallucinated / unsafe content reaches learner | Serve-time re-check of fidelity ≥ min, safety flags, blocking a11y; fail-closed to base; quarantine + kill-switch |
| Automated educational decision without human involvement | Default auto-serve after gates; forced `require_instructor_approval` for COPPA minors and EU AI Act high-risk (`ACE_EU_AI_ACT_HIGH_RISK`); contest + post-hoc review |
| Systematic disadvantage | Fairness audit (coverage, fidelity, lift by language/section/accommodation) with disparity alerts |
| Minor profiling without consent | aigateway COPPA deny; base content; no indicator |
| Opacity | Adapted banner, AI disclosure feature card, contest + opt-out |

## 4. Retention & DSAR

ACE artifacts are included in DSAR export/erasure (`adaptation_profiles`, `adaptation_servings`, `adaptation_outcomes`, opt-outs, contests, events). RoPA activity: “Adaptive content personalization”.

## 5. Change control

Any prompt or model change must:

1. Bump `prompt_version` / model binding.
2. Re-run fidelity eval fixtures and fairness regression.
3. Update this DPIA delta section and the [EU AI Act checklist](./ace-eu-ai-act-checklist.md).
