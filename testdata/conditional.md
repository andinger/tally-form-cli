---
name: "Conditional Test"
---

F1: Do you use AI?
> type: single-choice
- Yes, actively
- No, not yet
- Not interested

> show F2 when F1 is "Yes, actively"

F2: Which tools?
> type: long-text
> hidden: true
