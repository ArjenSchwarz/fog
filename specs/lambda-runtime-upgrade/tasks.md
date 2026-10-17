---
references:
    - specs/lambda-runtime-upgrade/smolspec.md
---
# Lambda runtime upgrade

- [x] 1. Supported report templates and Lambda library preserve report contracts, verified by handler and Python trigger tests <!-- id:zd468p1 -->

- [x] 2. Bootstrap archives initialize via the Runtime API and CI validates without feature-branch AWS publication <!-- id:zd468p2 -->
  - Blocked-by: zd468p1 (Supported report templates and Lambda library preserve report contracts, verified by handler and Python trigger tests)

- [x] 3. Upgrade builds and tests pass with migration prerequisites documented <!-- id:zd468p3 -->
  - Blocked-by: zd468p2 (Bootstrap archives initialize via the Runtime API and CI validates without feature-branch AWS publication)

- [x] 4. Both deployed architectures select matching bootstrap artifacts and pass native CI smoke checks <!-- id:zd468p4 -->
  - Blocked-by: zd468p2 (Bootstrap archives initialize via the Runtime API and CI validates without feature-branch AWS publication)
