---
slug: m8.putting-it-together.capstone
title: Capstone: Number-guessing game
sort_order: 3
kind: assignment
points: 20
group: Assignments
grade_policy: grader_agent
submission_modes: text,file
content_version: 1
---

# Capstone: Number-guessing game

**Goal:** Design and write a complete small program from a specification (learning outcomes 1–8).

## Spec

Build an interactive **number-guessing game**:

1. Import `random` and choose a secret integer from **1 to 20** inclusive (`random.randint(1, 20)`).
2. Allow the player **up to 5 guesses**.
3. Each turn:
   - Prompt for a guess
   - If the input is not a valid integer, print a short message and let them try again **without** consuming a successful guess slot (or count it — either policy is fine if you document it in a comment)
   - If the guess is too low / too high, print a hint
   - If the guess is correct, congratulate the player and end
4. If the player uses all guesses without winning, reveal the secret.

## Sample interaction (illustrative)

Secret is 12 (your secret will differ):

```text
Guess a number 1-20 (5 tries): 10
Too low
Guess a number 1-20 (5 tries): 15
Too high
Guess a number 1-20 (5 tries): 12
Correct! You win.
```

Or on failure:

```text
...
Out of guesses. The number was 12.
```

## Self-check checklist

- [ ] Uses `random.randint`
- [ ] Loop with a clear exit on win or max attempts
- [ ] Hints for low/high
- [ ] Input validation with `try`/`except ValueError` (or equivalent)
- [ ] Readable names; PEP 8-ish formatting
- [ ] Standard library only

## Submit

1. Your full program (text or `.py` file)
2. A short note (2–4 sentences) on **what you would improve next** (for example difficulty levels, replay menu, or scoring)

## Rubric (full credit for good-faith, on-spec work)

| Criterion | Full credit |
|---|---|
| Core loop | Win path and out-of-guesses path both exist |
| Feedback | Low/high (or equivalent) hints |
| Validation | Non-integer input handled without crashing |
| Reflection | Improvement note present |
| Style | Readable structure; no third-party packages |

Auto-graded with `grader_agent` (full credit for complete, good-faith submissions; optional AI feedback when enabled).

**Privacy:** do not paste secrets into online interpreters while testing.
