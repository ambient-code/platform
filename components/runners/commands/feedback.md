---
displayName: Feedback
description: Submit feedback about this session
icon: message-square
order: 100
---

The user wants to submit feedback about this session. Follow these steps:

1. Ask the user two questions:
   - **Rating**: Is their experience positive or negative?
   - **Comment**: A brief description of their feedback (what went well, what could improve, etc.)

2. Once you have both, call the `submit_feedback` tool (via the feedback MCP server) with:
   - `rating`: "positive" or "negative"
   - `comment`: the user's feedback in their own words

3. Confirm to the user that their feedback has been recorded and thank them.
