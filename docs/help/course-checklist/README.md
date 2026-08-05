# Designing a good course — checklist help hub

Instructor-facing guidance for every course checklist item. Machine-readable catalog:
[items.json](./items.json) (also copied to `clients/packages/checklist-help.json`).

**Support URL pattern:** `/help/course-checklist#<slug>` where `HelpRef` is
`course-checklist#<slug>`.

Each item covers:

1. **What** the check looks at  
2. **Why** it matters (with rubric citations)  
3. **How** to satisfy it  
4. **When** it is reasonable to dismiss  

Source chips link to the full rule-to-standard mapping:
[research.md](./research.md) (in-app: `/help/course-checklist/research`).

## Standards credited

- [Quality Matters](https://www.qualitymatters.org/qa-resources/rubric-standards)
- [SUNY OSCQR](https://oscqr.suny.edu/)
- [NSQ Online Courses](https://nsqol.org/the-standards/quality-online-courses/)
- [CAST UDL 3.0](https://udlguidelines.cast.org/)
- WCAG 2.1 AA

The checklist automates the **machine-checkable subset** of these standards. It is not QM
certification, an OSCQR review, or a WCAG conformance claim.

## Regenerating the catalog

From `server/`:

```bash
go run ./internal/service/coursechecklist/cmd/genhelp/
```

Registry tests fail if any `HelpRef` is dangling (`TestHelpRefResolves`).
