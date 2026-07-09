---
slug: m1.welcome.what-is-this
title: What is an official marketplace course?
sort_order: 0
content_version: 1
---

# What is an official marketplace course?

Official marketplace courses are **first-party**, free, self-paced courses that Lextures authors in the repository and provisions on every deploy. They appear in the in-app marketplace so any learner can claim them in one click.

Unlike the Welcome to Lextures intro course, these courses are **opt-in** — you claim them when you want them. Content is version-controlled Markdown and JSON, reviewed like code, and re-synced safely without duplicating structure items.

## Why this course exists

This smoke-test course proves the authoring harness works:

1. Manifest metadata lands on the course row (`price_cents = 0`, `marketplace_listed = true`)
2. Modules, pages, quizzes, and the syllabus reconcile idempotently
3. Learners can claim the course through the existing free-claim path

For real subject courses (AI Essentials, Introduction to Python, Personal Finance), see the marketplace courses epic plans.
