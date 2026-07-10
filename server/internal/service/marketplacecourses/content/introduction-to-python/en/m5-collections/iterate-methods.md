---
slug: m5.collections.iterate-methods
title: Iteration and common methods
sort_order: 2
content_version: 1
---

# Iteration and common methods

```python
for x in [1, 2, 3]:
    print(x)

counts = {"a": 2, "b": 1}
for key, value in counts.items():
    print(key, value)
```

**Expected output:**

```text
1
2
3
a 2
b 1
```

Useful list methods: `append`, `extend`, `pop`, `sort` (see docs).  
Useful dict methods: `keys`, `values`, `items`, `get`.

**Mutability reminder:** assigning `b = a` for lists shares the same list object. Copy with `b = a.copy()` or `b = list(a)` when you need a separate list.
