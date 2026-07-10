---
slug: m4.generative-ai.tokens
title: Tokens and next-word prediction
sort_order: 0
content_version: 1
---

# Tokens and next-word prediction

A **large language model (LLM)** processes text as **tokens** — pieces of words or characters — and predicts likely next tokens given the context so far.

## Next-token prediction

At each step, the model estimates which token is likely to come next, samples or selects from that distribution, and continues. That loop produces paragraphs that look coherent.

**Important:** this is **prediction from patterns in training data**, not looking up a verified encyclopedia entry for every claim.

True or false in spirit: *An LLM chooses each next word by predicting likely continuations from patterns in its training data, not by looking up verified facts.* → **True.**

## Training on text

Training on huge text corpora teaches statistical associations among tokens. It does not install a guaranteed truth module. For a conceptual intro, see [Generative AI for Everyone](https://www.coursera.org/learn/generative-ai-for-everyone) and Google's LLM material in the [ML Crash Course](https://developers.google.com/machine-learning/crash-course).

The influential transformer architecture is described in Vaswani et al., ["Attention Is All You Need"](https://arxiv.org/abs/1706.03762) (optional deep dive — not required reading for this course).
