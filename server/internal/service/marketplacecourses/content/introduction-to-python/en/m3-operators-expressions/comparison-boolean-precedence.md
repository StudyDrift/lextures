---
slug: m3.operators-expressions.comparison-boolean-precedence
title: Comparison, boolean, and precedence
sort_order: 1
content_version: 1
---

# Comparison, boolean, and precedence

## Comparison

```python
print(3 < 5)
print(3 == 5)
print(3 != 5)
print(3 <= 3)
```

**Expected output:**

```text
True
False
True
True
```

## Boolean operators

```python
print(True and False)
print(True or False)
print(not True)
```

**Expected output:**

```text
False
True
False
```

## Precedence (order of operations)

Multiplication and division happen before addition (like school math). Use parentheses to be clear:

```python
print(3 + 4 * 2)
print((3 + 4) * 2)
```

**Expected output:**

```text
11
14
```

Full rules: [Python docs — operator precedence](https://docs.python.org/3/reference/expressions.html#operator-precedence).
