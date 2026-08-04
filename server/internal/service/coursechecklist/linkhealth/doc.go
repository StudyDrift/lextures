// Package linkhealth implements the outbound URL checker for checklist item
// links.external-health (CC.6 FR-16). It is isolated so SSRF defences and
// crawl budget rules can be tested without the checklist engine.
package linkhealth
