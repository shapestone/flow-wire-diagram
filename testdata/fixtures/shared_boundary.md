# Shared-Boundary Limitation

When a child box's right column equals the parent's right column the strict
containment check (`other.RightCol > b.RightCol`) fails. Both boxes become
roots and content lines pass through unchanged.

```ascii
┌──────────────────┐
│ ┌────────────────┐
│ │ shared right   │
│ └────────────────┘
└──────────────────┘
```
