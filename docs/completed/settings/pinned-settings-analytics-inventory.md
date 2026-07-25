# Analytics inventory entry — pinned editor settings (PS.4 → S05)

Staff-facing product analytics for assignment/quiz authoring. Listed for RoPA / data-inventory mapping ([S05](../../plan/standards/S05-ropa-data-inventory-mapping.md)).

| Field | Value |
|---|---|
| **Store / channel** | Client product-analytics event stream (listener bus; warehouse when wired) |
| **Subjects** | Staff (instructors, admins) only — no learners |
| **Categories** | Behavioural (product usage): pin actions, search hashes, control interactions |
| **Special category?** | No |
| **Fields** | Enumerated in [event dictionary](pinned-settings-event-dictionary.md); schema-enforced; no content, course IDs, item IDs, or setting values |
| **Purpose** | Discoverability of buried settings; GA success metrics; IA review inputs |
| **Lawful basis** | Legitimate interest (product improvement) / institutional staff tool usage — confirm with DPO for jurisdiction |
| **Retention** | Inherits platform product-analytics retention |
| **Opt-out** | `navigator.doNotTrack` or `localStorage['lextures.analytics.opt-out']='1'` |
| **Related metrics** | Prometheus: `lextures_pinned_settings_*` (operational, not personal) |

When S05’s machine-readable inventory lands, register this store under `store_type = analytics` with category `behavioural` and subjects `staff`.
