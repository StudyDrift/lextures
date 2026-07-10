---
slug: m3.operators-expressions.strings-len
title: String concatenation, repetition, and len
sort_order: 2
content_version: 1
---

# String concatenation, repetition, and len

```python
print("Hi" + " " + "there")
print("ha" * 3)
print(len("Python"))
```

**Expected output:**

```text
Hi there
hahaha
6
```

## Type mismatch

You cannot add a string and an int directly:

```python
# print("score: " + 10)  # TypeError
print("score: " + str(10))
```

**Expected output:**

```text
score: 10
```

Or use an f-string: `print(f"score: {10}")` — same output.
