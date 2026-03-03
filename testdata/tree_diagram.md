# Tree Diagram Test

A tree diagram using ├── └── — should pass through unchanged.

```ascii
users
  ├── tasks (1:many)
  │   ├── title
  │   ├── status
  │   └── due_date
  └── goals (1:many)
      ├── title
      └── target_date
```
