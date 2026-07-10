---
slug: m4.decisions-looping.even-odd
title: Assignment: Even / Odd printer
sort_order: 3
kind: assignment
points: 10
group: Assignments
grade_policy: grader_agent
submission_modes: text,file
content_version: 1
---

# Assignment: Even / Odd printer

**Goal:** Practice `for`, `range`, and `if`/`else` (learning outcomes 3).

## Spec

Write a Python program that prints the integers from **1 through 20** (inclusive). For each number:

- If it is even, print `even` on that line (you may also include the number; see expected output).
- If it is odd, print `odd`.

Use this exact output format (number, space, label):

## Expected output

```text
1 odd
2 even
3 odd
4 even
5 odd
6 even
7 odd
8 even
9 odd
10 even
11 odd
12 even
13 odd
14 even
15 odd
16 even
17 odd
18 even
19 odd
20 even
```

## Self-check checklist

- [ ] Uses a loop (recommended: `for n in range(1, 21):`)
- [ ] Uses a conditional to choose even vs odd (`n % 2 == 0`)
- [ ] Output matches the 20 lines above when you run it
- [ ] No third-party packages

## Submit

Paste your full program as text (or upload a `.py` file). Optionally add one sentence on what you would improve.

This assignment is auto-graded (`grader_agent`). Full credit for a good-faith submission that meets the spec and self-check.
