---
slug: m4.decisions-looping.if-elif-else
title: if, elif, and else
sort_order: 0
content_version: 1
---

# if, elif, and else

Decisions use `if`, optional `elif`, and optional `else`. **Indentation** (usually 4 spaces) defines the block.

```python
score = 85
if score >= 90:
    print("A")
elif score >= 80:
    print("B")
else:
    print("below B")
```

**Expected output:**

```text
B
```

## Truthiness

Values like `0`, `""`, `[]`, and `None` are falsey; most other values are truthy.

```python
if "":
    print("yes")
else:
    print("no")
```

**Expected output:**

```text
no
```

## Common errors

- Forgetting the colon after `if condition:`
- Mixing tabs and spaces (IndentationError)
- Using `=` instead of `==` in a condition

Official reference: [tutorial §4](https://docs.python.org/3/tutorial/controlflow.html).
