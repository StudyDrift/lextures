---
slug: m1.getting-started.install-local
title: Install Python locally (optional)
sort_order: 2
content_version: 1
---

# Install Python locally (optional)

Skip this page if you are happy with an online interpreter. Come back when you want to run `.py` files on your computer.

## Download

1. Go to [python.org](https://www.python.org/) and open **Downloads**.
2. Install the current **Python 3** for your operating system.
3. On Windows, enable the option to **Add Python to PATH** if the installer offers it.

## Check the install

Open a terminal (Terminal on macOS/Linux; Command Prompt or PowerShell on Windows) and run:

```bash
python3 --version
```

On some Windows installs the command is `python` instead of `python3`. You should see a version like `Python 3.12.x`.

## Create and run a file

1. Create a file named `hello.py` with this content:

```python
print("Hello from a file!")
```

2. In the terminal, `cd` to that folder and run:

```bash
python3 hello.py
```

**Expected output:**

```text
Hello from a file!
```

Official getting-started material lives on [python.org](https://www.python.org/) and in the [Python tutorial](https://docs.python.org/3/tutorial/index.html) (§1–2).
