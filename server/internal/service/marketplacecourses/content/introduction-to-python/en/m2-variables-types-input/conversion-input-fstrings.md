---
slug: m2.variables-types-input.conversion-input-fstrings
title: Conversion, input, and f-strings
sort_order: 2
content_version: 1
---

# Conversion, input, and f-strings

## Type conversion

```python
print(int("42"))
print(float("3.5"))
print(str(99))
```

**Expected output:**

```text
42
3.5
99
```

## `input()` always returns a string

```python
# In a script or REPL, input waits for you to type and press Enter.
# Example (typed response: 21):
age_text = "21"  # stand-in for input("Age: ")
age = int(age_text)
print(age + 1)
```

**Expected output:**

```text
22
```

**Common bug:** comparing `input()` to a number without converting:

```python
# age = input("Age: ")
# if age > 18:  # TypeError: '>' not supported between str and int
age = int("21")
if age > 18:
    print("adult")
```

**Expected output:**

```text
adult
```

## f-strings (formatted string literals)

```python
name = "Ada"
score = 95
print(f"{name} scored {score}")
```

**Expected output:**

```text
Ada scored 95
```

Put expressions inside `{...}` inside an `f"..."` string.
