---
slug: m6.limitations-bias.bias
title: Where bias comes from
sort_order: 0
content_version: 1
---

# Where bias comes from

In machine learning, **bias** often means systematic error or unfair outcomes — not only personal prejudice (though human prejudice can enter via data and design).

## Common sources

- **Historical data** that reflects past discrimination
- **Under-representation** of some groups in the training set
- **Proxy variables** that stand in for sensitive attributes
- **Objective mismatch** (optimizing clicks ≠ optimizing fairness)

Example: a hiring model rates candidates from one neighborhood lower. The most likely root cause is often **bias present in the historical training data** — not "randomness" or "too little compute."

The [NIST AI Risk Management Framework](https://www.nist.gov/itl/ai-risk-management-framework) treats managing harmful bias as part of trustworthy AI. The [Stanford HAI AI Index](https://hai.stanford.edu/ai-index) tracks research and reporting trends (statistics change yearly — see Module 7's state-of-AI page).
