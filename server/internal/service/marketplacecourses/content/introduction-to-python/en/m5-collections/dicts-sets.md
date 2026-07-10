---
slug: m5.collections.dicts-sets
title: Dictionaries and sets
sort_order: 1
content_version: 1
---

# Dictionaries and sets

## Dictionaries (key → value)

```python
ages = {"Ada": 36, "Grace": 85}
print(ages["Ada"])
ages["Alan"] = 41
print(ages)
```

**Expected output:**

```text
36
{'Ada': 36, 'Grace': 85, 'Alan': 41}
```

Keys must be unique. Looking up a missing key raises `KeyError` (you will handle errors in Module 7).

## Sets (unique unordered items)

```python
letters = {"a", "b", "a"}
print(letters)
print("a" in letters)
```

**Expected output** (order may vary):

```text
{'a', 'b'}
True
```

## When to use which

| Need | Prefer |
|---|---|
| Ordered, changeable sequence | `list` |
| Fixed sequence | `tuple` |
| Lookup by unique key | `dict` |
| Unique membership tests | `set` |
