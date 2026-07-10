---
slug: m2.how-machines-learn.data-features
title: Data and features
sort_order: 0
content_version: 1
---

# Data and features

Machine learning systems learn from **data**: examples that represent the world as numbers, text, images, or other signals.

## Features

A **feature** is an input the model uses — for example, the words in an email, pixel values in a photo, or a customer's past purchases. Choosing and preparing features is a large part of real ML work.

## Labels (when you have them)

In many tasks, each example also has a **label**: the answer you want the model to predict (spam / not spam; price; next word). Labeled data enables **supervised** learning (next page).

## Data quality matters

If the data is incomplete, outdated, or skewed, the model will learn those flaws. "Garbage in, garbage out" is not a slogan — it is the default failure mode.

Google's [Machine Learning Crash Course](https://developers.google.com/machine-learning/crash-course) and [Elements of AI](https://www.elementsofai.com/) both stress that data — not magic — drives learning.
