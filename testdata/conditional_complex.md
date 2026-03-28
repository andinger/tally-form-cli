---
name: "Complex Conditional"
---

F1: KI Role?
> type: single-choice
- Strategic
- In use
- Experimenting
- Individual use
- Want to use
- No topic

> show F2, F3 when F1 is_not_any_of "Want to use", "No topic" and F1 is_not_empty

F2: Rules?
> type: single-choice
> hidden: true
- Yes
- No

F3: Data usage?
> type: single-choice
> hidden: true
- Yes
- No
