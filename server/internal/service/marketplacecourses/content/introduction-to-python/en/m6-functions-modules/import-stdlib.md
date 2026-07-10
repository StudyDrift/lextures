---
slug: m6.functions-modules.import-stdlib
title: import and the standard library
sort_order: 1
content_version: 1
---

# import and the standard library

```python
import math
import random
import statistics

print(math.sqrt(9))
print(statistics.mean([1, 2, 3]))
# random.randint is non-deterministic; show the call shape only in comments for quizzes
print(1 <= random.randint(1, 3) <= 3)
```

**Expected output** (last line is always True):

```text
3.0
2
True
```

- `import math` loads the module; use `math.sqrt`
- Or `from math import sqrt` then call `sqrt(9)`

Library reference: [docs.python.org/3/library](https://docs.python.org/3/library/index.html).

**Privacy:** `random` is fine for games and demos — not for security-critical secrets.
