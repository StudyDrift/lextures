---
slug: m6.functions-modules.write-reusable
title: Writing reusable functions
sort_order: 2
content_version: 1
---

# Writing reusable functions

Good beginner habits:

1. **One job per function** — convert units, compute a total, format a line.
2. **Name clearly** — `celsius_to_fahrenheit`, not `do_it`.
3. **Return values** instead of only printing when the result will be reused.
4. **Document** with a short docstring.

```python
def celsius_to_fahrenheit(c):
    """Convert Celsius to Fahrenheit."""
    return c * 9 / 5 + 32

for temp in [0, 100]:
    print(temp, "C ->", celsius_to_fahrenheit(temp), "F")
```

**Expected output:**

```text
0 C -> 32.0 F
100 C -> 212.0 F
```
