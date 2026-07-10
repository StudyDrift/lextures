---
slug: m8.putting-it-together.program-design
title: Program design from a specification
sort_order: 0
content_version: 1
---

# Program design from a specification

Before typing a long program:

1. **Restate the goal** in one sentence.
2. **List inputs and outputs** (what the user types; what you print).
3. **Break into steps** (validate input → loop → update state → exit).
4. **Name functions** for the steps you will reuse.
5. **Write a tiny vertical slice** that runs, then grow it.

Example sketch for a guessing game:

- Pick a secret number (`random.randint`)
- Loop: read a guess, validate it is an int, compare, give hint
- Stop on correct guess or after N tries
- Print a closing message

Keep each function short. Print user-facing messages in one place when you can.
