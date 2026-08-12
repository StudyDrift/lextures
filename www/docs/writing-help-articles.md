# Writing help articles

Help content describes behavior that is already shipped. Before publishing, perform the procedure in a test organization using the release named by `verifiedAgainst`; never infer a workflow from a plan alone.

## Required metadata

Create or clone an article in the Marketing Content workspace. Keep `category`, `roles`, `segments`, `updated`, `verifiedAgainst`, `supportTicketThemes`, and three to six `relatedTo` paths current. Compliance articles also require `reviewedBy` and a `reviewedAt` date no more than 90 days old.

## Article contract

Answer the primary question in the first 40–60 words. Use question-style headings, a real `::: steps` block for procedures, three to five takeaways, three to six FAQs, primary sources, and descriptive internal links. The workspace publish gate requires a score of at least 8.0.

State which roles can act and what changes for K–12, higher education, or homeschool. Describe permission and configuration dependencies. Never promise a certification or a data-handling behavior that the current legal and product owners have not verified.

## Screenshots

Text is authoritative; an image cannot contain the only copy of an instruction. Add a capture spec under `scripts/screenshots`, use the synthetic demo seed, and run `npm run help:screenshots`. Alt text must describe the relevant state, not merely name the screen. The capture command warns and reuses checked-in images when the demo environment is unavailable.

## Freshness and review

Set `updated` only after repeating the workflow against the named version or release date. The workspace Health view reports articles older than 180 days and the governance dashboard tracks the overdue ratio. When a user-visible product change affects a published article, update it in the workspace or identify the follow-up owner.
