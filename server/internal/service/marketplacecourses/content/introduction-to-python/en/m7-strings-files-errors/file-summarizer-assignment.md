---
slug: m7.strings-files-errors.file-summarizer
title: Assignment: File summarizer
sort_order: 3
kind: assignment
points: 10
group: Assignments
grade_policy: grader_agent
submission_modes: text,file
content_version: 1
---

# Assignment: File summarizer

**Goal:** Read a text file, count lines/words, and handle a missing file (learning outcome 6).

## Spec

1. Create a small text file named `sample.txt` in the same folder as your script with **exactly** these three lines:

```text
hello world
python is fun
keep practicing
```

2. Write a program that:
   - Opens `sample.txt` with `with` and `encoding="utf-8"`
   - Counts **lines** and **words** (split on whitespace)
   - Prints a short summary
3. Wrap the open/read in `try` / `except FileNotFoundError` and print a clear message if the file is missing.

## Expected output (when sample.txt exists)

```text
Lines: 3
Words: 8
```

(Word count: hello, world, python, is, fun, keep, practicing → 8 words if the file has no extra blank lines.)

## Expected output (when sample.txt is missing)

Something clear such as:

```text
File not found: sample.txt
```

## Self-check checklist

- [ ] Uses `with open(...)`
- [ ] Handles `FileNotFoundError`
- [ ] Line and word counts match for the provided sample
- [ ] No third-party packages

## Submit

Paste your program (and optionally note that you tested both the happy path and missing-file path). Upload a `.py` file if you prefer.

Auto-graded (`grader_agent`) for good-faith, on-spec work.
