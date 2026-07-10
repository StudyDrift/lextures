---
slug: m2.variables-types-input.core-types
title: Core types: int, float, str, bool
sort_order: 1
content_version: 1
---

# Core types: int, float, str, bool

Python values have **types**. Check with `type()`:

```python
print(type(3))
print(type(3.14))
print(type("hi"))
print(type(True))
```

**Expected output:**

```text
<class 'int'>
<class 'float'>
<class 'str'>
<class 'bool'>
```

| Type | Meaning | Examples |
|---|---|---|
| `int` | Whole numbers | `0`, `-2`, `42` |
| `float` | Numbers with a fractional part | `3.14`, `-0.5` |
| `str` | Text (string) | `"hi"`, `'ok'` |
| `bool` | Boolean | `True`, `False` |

Strings can use single or double quotes. Booleans are capitalized: `True` / `False`, not `true` / `false`.

Official overview: [Python tutorial §3](https://docs.python.org/3/tutorial/introduction.html).
