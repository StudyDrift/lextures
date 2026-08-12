# Content lifecycle

Every new blog, help, comparison, or research page has an `owner` (an author slug is accepted) and `reviewDue` date. Weekly, `npm run seo:lifecycle` creates `dist/seo-lifecycle.md` and fails when more than 10% of help or comparison content is stale.

At review, choose one outcome: refresh and advance `updated`/`reviewDue`; consolidate into a stronger page and add a permanent redirect; or retire with HTTP 410 and sitemap removal. Never silently delete a published URL. Record consolidations, retirements, exceptions, and owners in the lifecycle log.
