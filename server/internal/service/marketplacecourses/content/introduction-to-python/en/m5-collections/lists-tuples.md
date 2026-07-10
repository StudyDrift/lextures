---
slug: m5.collections.lists-tuples
title: Lists and tuples
sort_order: 0
content_version: 1
---

# Lists and tuples

## Lists (mutable sequences)

```python
nums = [10, 20, 30]
print(nums[0])
print(nums[-1])
print(nums[0:2])
nums.append(40)
print(nums)
```

**Expected output:**

```text
10
30
[10, 20]
[10, 20, 30, 40]
```

- Indexing starts at **0**
- Slicing `a:b` includes `a`, excludes `b`
- Lists can grow and change (`append`, item assignment)

## Tuples (immutable sequences)

```python
point = (3, 4)
print(point[0])
# point[0] = 99  # TypeError — tuples cannot be changed
```

**Expected output:**

```text
3
```

Use tuples for fixed records; use lists when you need to modify contents.

See [tutorial §5](https://docs.python.org/3/tutorial/datastructures.html).
