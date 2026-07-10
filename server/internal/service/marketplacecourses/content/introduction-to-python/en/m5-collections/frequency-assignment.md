---
slug: m5.collections.frequency
title: Assignment: Letter frequency
sort_order: 3
kind: assignment
points: 10
group: Assignments
grade_policy: grader_agent
submission_modes: text,file
content_version: 1
---

# Assignment: Letter frequency

**Goal:** Practice lists/strings, dictionaries, and iteration (learning outcome 4).

## Spec

Write a program that:

1. Starts from this string (you may hard-code it):

```python
text = "banana"
```

2. Builds a **frequency dictionary** mapping each character to how many times it appears.
3. Prints the dictionary (order of keys may vary).
4. Prints the **most common character** and its count. If there is a tie, any of the tied characters is acceptable.

## Expected behavior (example)

For `text = "banana"`, frequencies are `b:1`, `a:3`, `n:2`. Most common is `a` with `3`.

Example output (dict key order may differ):

```text
{'b': 1, 'a': 3, 'n': 2}
Most common: a (3)
```

## Self-check checklist

- [ ] Uses a `dict` to count characters
- [ ] Iterates over the string (or a list of characters)
- [ ] Prints frequencies and a most-common result consistent with the counts
- [ ] Standard library only

## Submit

Paste your program as text or upload a `.py` file.

Auto-graded (`grader_agent`) for good-faith, on-spec work.
