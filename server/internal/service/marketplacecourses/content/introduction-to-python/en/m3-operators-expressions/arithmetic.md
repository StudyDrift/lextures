---
slug: m3.operators-expressions.arithmetic
title: Arithmetic operators
sort_order: 0
content_version: 1
---

# Arithmetic operators

| Operator | Meaning | Example | Result |
|---|---|---|---|
| `+` | Add | `3 + 2` | `5` |
| `-` | Subtract | `3 - 2` | `1` |
| `*` | Multiply | `3 * 2` | `6` |
| `/` | Divide (float) | `7 / 2` | `3.5` |
| `//` | Floor divide | `7 // 2` | `3` |
| `%` | Remainder | `7 % 2` | `1` |
| `**` | Power | `2 ** 3` | `8` |

```python
print(7 / 2)
print(7 // 2)
print(7 % 2)
print(2 ** 3)
```

**Expected output:**

```text
3.5
3
1
8
```

`/` always produces a float in Python 3. Use `//` when you want an integer quotient (toward −∞ for negatives — see the docs if you need that edge case).
