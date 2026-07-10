---
slug: m6.functions-modules.temperature
title: Assignment: Temperature converter function
sort_order: 3
kind: assignment
points: 10
group: Assignments
grade_policy: grader_agent
submission_modes: text,file
content_version: 1
---

# Assignment: Temperature converter function

**Goal:** Write a reusable function and call it in a loop (learning outcome 5).

## Spec

1. Define `celsius_to_fahrenheit(c)` that returns `c * 9 / 5 + 32`.
2. Include a short **docstring**.
3. Given a list of Celsius values `[0, 20, 37, 100]`, loop and print each conversion.
4. You may `import` from the standard library if useful (not required).

## Expected output

```text
0 C -> 32.0 F
20 C -> 68.0 F
37 C -> 98.6 F
100 C -> 212.0 F
```

(Equivalent formatting is fine if the numbers match.)

## Self-check checklist

- [ ] Function has a docstring
- [ ] Function **returns** the Fahrenheit value (not only prints inside the function)
- [ ] Loop uses the function for each list element
- [ ] Output numbers match the expected conversions

## Submit

Paste your program or upload a `.py` file.

Auto-graded (`grader_agent`) for good-faith, on-spec work.
