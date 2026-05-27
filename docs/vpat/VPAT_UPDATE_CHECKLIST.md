# VPAT Update Checklist

Run this checklist at each major Lextures release to keep the VPAT current.

## Trigger

This checklist should be executed whenever:
- A major or minor release ships (e.g., `1.1`, `2.0`)
- The 10.7 WCAG audit produces new findings
- A "Partially Supports" item is remediated or its status changes
- A new feature introduces UI accessible to end users

A GitHub issue is auto-created from `.github/ISSUE_TEMPLATE/vpat-update.yml` on each release.

---

## Pre-Work

- [ ] Obtain the latest axe-core CI report from the most recent release branch run
- [ ] Review any screen-reader testing notes from the release QA cycle
- [ ] Check the 10.7 audit backlog for recently closed remediation items
- [ ] Note the new product version and release date

---

## VPAT Source Document (`docs/vpat/VPAT_2.5_INT_Lextures_YYYY-MM.md`)

- [ ] Duplicate the previous VPAT file with the new `YYYY-MM` date slug
- [ ] Update **Product Version** and **Report Date** in the Product Information table
- [ ] Update **Report Version** (increment minor version if content changed)
- [ ] For each WCAG criterion marked "Partially Supports":
  - [ ] Check whether the underlying issue has been closed
  - [ ] If resolved: change to "Supports" and update the Remarks field
  - [ ] If still open: update the remediation timeline in Remarks
- [ ] Review all "Supports" entries for any regressions introduced in this release
- [ ] Review any new features for new WCAG exposure (e.g., new file upload, new video player)
- [ ] Update Section 508 / EN 301 549 entries if platform-level behavior changed

---

## Web Application (`clients/web/src/lib/vpat-data.ts`)

- [ ] Mirror all conformance-level changes from the VPAT source document into `vpat-data.ts`
- [ ] Update the `EVAL_DATE` constant in `clients/web/src/pages/vpat-page.tsx`
- [ ] Update `PRODUCT_VERSION` in `clients/web/src/pages/vpat-page.tsx` if the version changed

---

## Public Download Asset (`clients/web/public/vpat/`)

- [ ] Copy the new VPAT source file to `clients/web/public/vpat/VPAT_2.5_INT_Lextures_YYYY-MM.md`
- [ ] Update the download `href` in `vpat-page.tsx` to point to the new file
- [ ] Archive the previous version (keep old file; do not delete)

---

## QA

- [ ] Run `npm run lint` in `clients/web/` — no new errors
- [ ] Run `npm test` in `clients/web/` — all VPAT unit tests pass
- [ ] Run the e2e suite: `npx playwright test vpat.spec.ts` — all VPAT e2e tests pass
- [ ] Manually open `/accessibility/vpat` in Chrome, Firefox, and Safari — page renders correctly
- [ ] Manually verify download link returns the new VPAT file
- [ ] Run axe-core on `/accessibility/vpat` — no new violations

---

## Communication

- [ ] Notify the accessibility specialist that the updated VPAT is live
- [ ] Update the sales enablement deck with the new VPAT link (if version changed)
- [ ] Close the GitHub issue created from the VPAT update template

---

## Version Archive

Old VPAT versions are retained at their original filenames in `clients/web/public/vpat/`. Do not delete previous versions.
