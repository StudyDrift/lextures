---
slug: m2.how-machines-learn.training-inference
title: Training vs. inference
sort_order: 1
content_version: 1
---

# Training vs. inference

Two phases matter for almost every ML system:

## Training

During **training**, the system adjusts internal parameters using training data so that its predictions improve on that task. Training can be expensive and is usually done by the model developer, not by you when you type a prompt.

## Inference

During **inference** (sometimes called prediction or serving), a **trained** model takes a new input and produces an output — a class label, a score, or generated text. When you use a chatbot, you are almost always doing inference on a model someone else already trained.

## A model is a pattern learned from data

A useful mental model: the trained model is a compressed set of patterns found in data. It is not a database of verified facts and not a person with beliefs.

See [Google ML Crash Course](https://developers.google.com/machine-learning/crash-course) for a fuller treatment of training and prediction.
