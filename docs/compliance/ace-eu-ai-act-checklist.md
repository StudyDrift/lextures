# EU AI Act high-risk conformity checklist — Adaptive Content Engine

> Living artifact for AC.8 / Standards S13. Annex III education AI systems.

| Field | Value |
|---|---|
| **Feature** | Adaptive Content Engine |
| **Prompt / model** | `adaptive_content_variant` v1; course-setup model binding |
| **Last reviewed** | 2026-07-25 |

## Checklist

| Art. / obligation | Requirement | ACE evidence | Status |
|---|---|---|---|
| Art. 9 risk mgmt | Continuous risk process | DPIA + fairness/regression jobs + incident quarantine | Done |
| Art. 10 data governance | Training/eval data quality | Instructor base content + key terms; no learner PII to model | Done |
| Art. 11 technical docs | Technical documentation | This checklist + DPIA + runbooks | Done |
| Art. 12 record-keeping | Automatic logging | `adaptive_content_events` append-only trail (generate, serve, gate_block, contest, quarantine) | Done |
| Art. 13 transparency | Inform deployers/users | AI disclosure feature card; student Adapted banner; guardian/opt-out copy | Done |
| Art. 14 human oversight | Meaningful human involvement | Gates + force instructor approval for COPPA/EU high-risk; contest inbox; revoke | Done |
| Art. 15 accuracy / robustness | Accuracy & cybersecurity | Fidelity/safety/a11y gates; fail-closed serve; kill-switch | Done |
| Art. 52 transparency | AI interaction disclosure | Adapted-for-you indicator; disclosure page | Done |
| Efficacy monitoring | Ongoing performance | AC.7 effectiveness + AC.8 fairness disparity flags | Done |

## Human oversight modes

- **Default:** auto-serve after fidelity ≥ min, safety pass, no blocking a11y.
- **Forced pre-serve approval:** COPPA minors; `ACE_EU_AI_ACT_HIGH_RISK=true`; course `requireInstructorApproval`.
- **Post-hoc:** instructor revoke, contest path, fairness/regression alerts, quarantine, kill-switch.

## References

- [ACE DPIA](./ace-dpia.md)
- [Kill-switch runbook](../runbooks/adaptive-content-kill-switch.md)
- [Governance runbook](../runbooks/adaptive-content-governance.md)
