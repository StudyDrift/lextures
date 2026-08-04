# Checklist link-health fetcher (CC.6)

`links.external-health` is the only checklist item that makes outbound network
calls. Implementation lives in
`server/internal/service/coursechecklist/linkhealth/`.

## Behaviour

- On checklist read/refresh, if the per-course cache is missing or older than
  24 hours, the item returns `unknown` (“Checking links…”) and enqueues
  `checklist-linkcheck` (at most one in-flight job per course via unique key).
- The worker extracts distinct `http(s)` URLs from the shared `ContentDoc`,
  caps at **200**, and checks with **8-way concurrency**, **2s** per request,
  **5s** total budget.
- Results land in `course.course_checklist_link_health` and are swept after
  30 days.

## Kill switch

```bash
CHECKLIST_LINKCHECK_ENABLED=false   # default — item stays unknown / no outbound calls
CHECKLIST_LINKCHECK_ENABLED=true    # enable after security review
```

## SSRF and crawl hygiene

- Blocks private, loopback, and link-local destinations **before dial** and on
  redirect.
- Prefers `HEAD`, falls back to `GET`, reads at most 64 KiB, sends no cookies or
  auth headers.
- User-Agent: `LexturesLinkHealth/1.0 (+https://docs.lextures.com/dev/checklist-linkhealth)`
- Failures degrade to `unknown` / `error`, never claim a link is dead because of
  a network hiccup.

## Metrics

- `coursechecklist_linkcheck_duration_seconds`
- `coursechecklist_linkcheck_urls_total{result=ok|dead|error|skipped}`
- `coursechecklist_linkcheck_blocked_total{reason=private_range|robots|rate_limit|…}`
