---
slug: m4.decisions-looping.break-continue-nesting
title: break, continue, and nesting
sort_order: 2
content_version: 1
---

# break, continue, and nesting

## `break` and `continue`

```python
for i in range(5):
    if i == 2:
        continue
    if i == 4:
        break
    print(i)
```

**Expected output:**

```text
0
1
3
```

- `continue` skips the rest of this iteration.
- `break` exits the loop entirely.

## Nesting

You can put loops inside `if`, or `if` inside loops. Keep indentation consistent.

```python
for n in range(1, 4):
    if n % 2 == 0:
        print(n, "even")
    else:
        print(n, "odd")
```

**Expected output:**

```text
1 odd
2 even
3 odd
```
