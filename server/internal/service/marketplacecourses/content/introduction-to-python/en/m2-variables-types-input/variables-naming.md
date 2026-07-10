---
slug: m2.variables-types-input.variables-naming
title: Variables and naming
sort_order: 0
content_version: 1
---

# Variables and naming

A **variable** is a name that refers to a value. Assignment uses `=`:

```python
message = "Hello"
count = 3
print(message)
print(count)
```

**Expected output:**

```text
Hello
3
```

You can reassign:

```python
count = 3
count = count + 1
print(count)
```

**Expected output:**

```text
4
```

## Naming basics (PEP 8)

- Use lowercase with underscores for ordinary names: `user_name`, `total_score`.
- Names are case-sensitive: `Score` and `score` are different.
- Prefer descriptive names over single letters (except short loop indices).

See [PEP 8](https://peps.python.org/pep-0008/) for the full style guide (we cover more in Module 8).

## Common error: `=` vs `==`

- `=` **assigns** a value to a name.
- `==` **compares** two values for equality (Module 3–4).

```python
x = 5
print(x == 5)
```

**Expected output:**

```text
True
```
