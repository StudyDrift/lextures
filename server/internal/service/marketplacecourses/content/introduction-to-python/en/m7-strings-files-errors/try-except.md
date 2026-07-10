---
slug: m7.strings-files-errors.try-except
title: try, except, and common exceptions
sort_order: 2
content_version: 1
---

# try, except, and common exceptions

Errors raise **exceptions**. Catch them with `try` / `except` when you can recover.

```python
try:
    with open("missing.txt", "r", encoding="utf-8") as f:
        print(f.read())
except FileNotFoundError:
    print("File not found")
```

**Expected output:**

```text
File not found
```

## Common beginner exceptions

| Exception | Typical cause |
|---|---|
| `SyntaxError` | Typos, missing `:` or quotes |
| `IndentationError` | Bad indent |
| `NameError` | Using a name before defining it |
| `TypeError` | Wrong types (e.g. `"1" + 1`) |
| `ValueError` | `int("hi")` |
| `KeyError` | Missing dict key |
| `FileNotFoundError` | Bad path / missing file |
| `ZeroDivisionError` | `/ 0` |

```python
try:
    print(int("hi"))
except ValueError:
    print("not a number")
```

**Expected output:**

```text
not a number
```

Do not use bare `except:` that swallows everything while learning — catch the specific error you expect.
