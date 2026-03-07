# Mixed Diagram Types

This file contains multiple diagrams of different types.

## A simple broken box

```ascii
┌──────────────┐
│  line one    │
│ line two   │
└──────────────┘
```

## A tree (should pass through)

```ascii
project/
  ├── src/
  │   └── main.go
  └── tests/
      └── main_test.go
```

## A nested box with broken content

```ascii
┌────────────────────┐
│  ┌──────────────┐  │
│  │ component    │  │
│  │ short      │ │
│  └──────────────┘  │
└────────────────────┘
```

## A correct diagram (no changes needed)

```ascii
┌──────────┐
│  result  │
└──────────┘
```
