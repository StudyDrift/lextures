---
slug: m6.functions-modules.def-return-scope
title: def, parameters, return, and scope
sort_order: 0
content_version: 1
---

# def, parameters, return, and scope

```python
def add(a, b):
    return a + b

print(add(2, 3))
```

**Expected output:**

```text
5
```

## Defaults and docstrings

```python
def greet(name, greeting="Hello"):
    """Return a greeting line."""
    return f"{greeting}, {name}"

print(greet("Ada"))
print(greet("Ada", greeting="Hi"))
```

**Expected output:**

```text
Hello, Ada
Hi, Ada
```

## Scope

Names assigned inside a function are local unless declared otherwise. Prefer passing values in and returning results out.

```python
def double(n):
    return n * 2

x = 4
print(double(x))
print(x)
```

**Expected output:**

```text
8
4
```

See [tutorial §4.7–6](https://docs.python.org/3/tutorial/controlflow.html#defining-functions).
