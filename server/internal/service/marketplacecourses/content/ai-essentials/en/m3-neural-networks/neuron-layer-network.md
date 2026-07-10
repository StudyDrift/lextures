---
slug: m3.neural-networks.analogy
title: From neuron to network (analogy)
sort_order: 0
content_version: 1
---

# From neuron to network (analogy)

A **neural network** is a stack of simple computing units (loosely inspired by neurons) organized in **layers**. Each unit combines inputs and passes a transformed signal forward.

You do **not** need biology or calculus to use this analogy:

1. **Input layer** — receives features (pixels, tokens, numbers).
2. **Hidden layers** — transform representations step by step.
3. **Output layer** — produces predictions or scores.

In a simple feed-forward sketch: inputs flow left-to-right through stacked layers until the output. Real systems vary widely (skip connections, attention, and more), but the layered pipeline is enough for literacy.

[Elements of AI](https://www.elementsofai.com/) and [Google ML Crash Course](https://developers.google.com/machine-learning/crash-course) introduce networks at this conceptual level.
