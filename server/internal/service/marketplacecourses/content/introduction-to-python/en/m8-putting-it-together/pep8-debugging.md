---
slug: m8.putting-it-together.pep8-debugging
title: PEP 8 basics and debugging
sort_order: 1
content_version: 1
---

# PEP 8 basics and debugging

## PEP 8 (starter subset)

From [PEP 8](https://peps.python.org/pep-0008/):

- 4-space indentation; no mixed tabs
- `snake_case` for functions and variables
- Spaces around `=` in assignments and after commas: `a = 1`, `f(x, y)`
- Blank lines between top-level functions
- Descriptive names over cryptic abbreviations

You do not need to memorize every rule — aim for readable consistency.

## Debugging checklist

1. Read the **exception type and line number**.
2. Print intermediate values (`print(repr(x))`) when stuck.
3. Check **off-by-one** in `range` and slices.
4. Check **`=` vs `==`** and missing conversions from `input()`.
5. Run the smallest example that still fails.

```python
# Spot the bug: comparison used assignment
# if guess = 7:  # SyntaxError in modern Python; use ==
guess = 7
if guess == 7:
    print("ok")
```

**Expected output:**

```text
ok
```
