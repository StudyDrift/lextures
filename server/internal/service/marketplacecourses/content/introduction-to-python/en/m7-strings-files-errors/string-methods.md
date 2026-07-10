---
slug: m7.strings-files-errors.string-methods
title: String methods
sort_order: 0
content_version: 1
---

# String methods

Strings have useful methods. They return **new** strings (strings are immutable).

```python
s = "  Hello, World  "
print(s.strip())
print(s.lower())
print("a,b,c".split(","))
print("-".join(["a", "b", "c"]))
print("hello".replace("l", "L"))
```

**Expected output:**

```text
Hello, World
  hello, world  
['a', 'b', 'c']
a-b-c
heLLo
```

Note: `s.lower()` does not change `s` unless you assign the result back.

```python
print("Python".startswith("Py"))
print("file.txt".endswith(".txt"))
```

**Expected output:**

```text
True
True
```

More in [tutorial §7](https://docs.python.org/3/tutorial/inputoutput.html) and the [string methods docs](https://docs.python.org/3/library/stdtypes.html#string-methods).
