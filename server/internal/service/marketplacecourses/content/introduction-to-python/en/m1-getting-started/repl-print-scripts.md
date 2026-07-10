---
slug: m1.getting-started.repl-print-scripts
title: REPL, print, scripts, and comments
sort_order: 3
content_version: 1
---

# REPL, print, scripts, and comments

## REPL vs script

| Mode | How you use it | Typical use |
|---|---|---|
| **REPL** (Read–Eval–Print Loop) | Type at `>>>`, see results immediately | Experiment, check one line |
| **Script** | Save code in a `.py` file and run it | Programs you keep and share |

Both run the same Python language.

## `print()`

```python
print("Hi")
print(2 + 2)
```

**Expected output:**

```text
Hi
4
```

`print` can take several values separated by commas:

```python
print("sum:", 1 + 2)
```

**Expected output:**

```text
sum: 3
```

## Comments

Lines starting with `#` are **comments**. Python ignores them. Use them for humans reading the code.

```python
# This is a comment
print("Still runs")  # trailing comment
```

**Expected output:**

```text
Still runs
```

## Common beginner error: missing quotes

```python
# Wrong — name Hello is not defined
# print(Hello)
print("Hello")  # correct
```

**Expected output** (of the correct line):

```text
Hello
```

Try the examples in your REPL or a small `.py` file before moving on.
