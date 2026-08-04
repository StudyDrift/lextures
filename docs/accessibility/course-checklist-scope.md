# Course checklist accessibility scope

The course checklist includes automated checks for authored-content accessibility
(alt text, captions hints, heading order, link text, tables, contrast on custom
themes, PDF text-layer heuristics, and related UDL nudges).

## What these checks are

- Fast, deterministic heuristics over content the platform can read (markdown,
  syllabus sections, course settings, file metadata).
- A helper for instructors and accessibility coordinators to find likely gaps.

## What these checks are not

- **Not WCAG conformance.** Passing every checklist item does **not** mean the
  course meets WCAG 2.1 AA, Section 508, EN 301 549, or the EAA.
- **Not a substitute for manual testing** with assistive technology, keyboard
  navigation, or an ACR/VPAT.
- **Not a full document audit.** Uploaded PDFs are checked with a text-layer
  heuristic only; complex PDF tag structure is out of scope.

For manual test scripts and platform accessibility work, see the rest of
`docs/accessibility/` and `docs/vpat/`.
