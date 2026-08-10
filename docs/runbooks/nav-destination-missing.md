# Runbook: navigation destination missing for a user

## Symptoms

User reports a sidebar link is gone (e.g. Gradebook, Event log, Ask AI).

## Quick triage

1. **Feature flag** — Settings → Global platform (or course features). Flag off
   removes the destination entirely (by design, UX.7 FR-10).
2. **Permission / role** — Student audience never sees instructor analytics
   (FR-8). “View as: Test Student” switches the sidebar to the student model.
3. **Personalisation** — User may have **hidden** the item.
   - Ask them to open **Customise** at the bottom of the sidebar → unhide, or
     **Reset to default**.
   - Or call `DELETE /api/v1/nav/preferences?scope=global` (or
     `course:<code>`) as the user.
4. **V2 overflow** — If `ffNavigationV2` is on, look under the section’s
   **More** control or **Pinned**.
5. **Registry** — Engineer: destination must exist in
   `clients/web/src/lib/nav/registry-*.ts` with section + priority; `npm run nav:check`.

## Inspect prefs (SQL)

```sql
SELECT scope, pinned, hidden, collapsed, updated_at
FROM settings.user_nav_preferences
WHERE user_id = $1;
```

## Reset prefs (SQL)

```sql
DELETE FROM settings.user_nav_preferences
WHERE user_id = $1 AND scope = $2;
```
