# Content Too Wide — No Trailing Space Slack

A box where the frame says width 11 (RightCol=10) but some content lines have
their right wall one column too far right AND the text fills right up to the
wrong wall with no trailing space. The repair should widen the box to fit the
content rather than truncating text.

```ascii
┌─────────┐
│ abcde   │
│ abcdefghi│
└─────────┘
```

After repair the frame and all lines should share a consistent right wall.
