---
slug: m4.decisions-looping.while-for-range
title: while, for, and range
sort_order: 1
content_version: 1
---

# while, for, and range

## `while`

```python
n = 0
while n < 3:
    print(n)
    n = n + 1
```

**Expected output:**

```text
0
1
2
```

Watch for infinite loops: the condition must eventually become false (or you `break`).

## `for` and `range`

```python
for i in range(3):
    print(i)
```

**Expected output:**

```text
0
1
2
```

`range(3)` yields `0, 1, 2` — it stops **before** the end value (common off-by-one source).

```python
for i in range(2, 5):
    print(i)
```

**Expected output:**

```text
2
3
4
```
