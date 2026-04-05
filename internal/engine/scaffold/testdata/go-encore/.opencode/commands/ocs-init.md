---
description: Analyze project and generate optimal OpenCode workflow
agent: plan
subtask: false
---

Analyze this project and create the optimal OpenCode workflow.

Step 1: Discover project structure
```bash
!`ocs discover`
```

Step 2: Detect best template
```bash
!`ocs template detect`
```

Step 3: Generate the workflow
```bash
!`ocs init --template <detected-template> --force`
```

Step 4: Verify the setup
```bash
!`ocs doctor`
```

Report what was created and how to use it.
