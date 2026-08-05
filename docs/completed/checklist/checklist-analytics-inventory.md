# Analytics inventory entry — course checklist (CC.10 → S05)

Staff-facing product analytics for course readiness checklist. Listed for RoPA / data-inventory
mapping ([S05](../../plan/standards/S05-ropa-data-inventory-mapping.md)).

| Field | Value |
|---|---|
| **Store / channel** | (1) Client product-analytics event stream (listener bus; warehouse when wired); (2) `course.course_checklist_events` (server audit for dismiss/restore); (3) Prometheus operational counters |
| **Subjects** | Staff (instructors, designers, course owners, admins) only — no learners |
| **Categories** | Behavioural (product usage): views, expands, navigations, dismissals, assist accept rates; operational metrics for rule health |
| **Special category?** | No. Accommodation rules (`accommodations.honored`, `accommodations.reviewed`) are **excluded** from client analytics and from `item_status` counters |
| **Fields** | Enumerated in [event dictionary](checklist-event-dictionary.md); schema-enforced; no evidence content, course codes, user IDs, notes, or free text |
| **Purpose** | Rule quality (pass / dismissal rates), target-resolution health, assisted-fix efficacy, tier-promotion gates (FR-20) |
| **Lawful basis** | Legitimate interest (product improvement) / institutional staff tool usage — confirm with DPO for jurisdiction |
| **Retention** | Client events: platform product-analytics retention. Server checklist events: 400 days (CC.2). Prometheus: platform scrape retention |
| **Opt-out** | `navigator.doNotTrack` or `localStorage['lextures.analytics.opt-out']='1'` |
| **Related metrics** | `lextures_coursechecklist_*` (operational, not personal) |

When S05’s machine-readable inventory lands, register this store under `store_type = analytics` with
category `behavioural` and subjects `staff`.
