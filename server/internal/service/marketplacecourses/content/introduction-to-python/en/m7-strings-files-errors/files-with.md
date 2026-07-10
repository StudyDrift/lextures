---
slug: m7.strings-files-errors.files-with
title: Reading and writing text files
sort_order: 1
content_version: 1
---

# Reading and writing text files

Use a `with` block so the file closes automatically.

```python
# Write (creates or overwrites)
with open("notes.txt", "w", encoding="utf-8") as f:
    f.write("line one\n")
    f.write("line two\n")

# Read
with open("notes.txt", "r", encoding="utf-8") as f:
    content = f.read()
print(content)
```

**Expected output:**

```text
line one
line two
```

Read line by line:

```python
with open("notes.txt", "r", encoding="utf-8") as f:
    for line in f:
        print(line.strip())
```

**Expected output:**

```text
line one
line two
```

Prefer `encoding="utf-8"` for portable text. Paths are relative to the process working directory — know where you run the script from.

See [tutorial §7](https://docs.python.org/3/tutorial/inputoutput.html#reading-and-writing-files) and [Automate the Boring Stuff](https://automatetheboringstuff.com/) for practical file projects.
