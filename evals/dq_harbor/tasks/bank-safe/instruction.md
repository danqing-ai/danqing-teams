`/app/system.txt` describes a Banker's-algorithm resource system.

Lines:
- `AVAILABLE a b c ...` — currently available units per resource type
- `#` comments
- Process lines: `NAME max0 max1 ... alloc0 alloc1 ...`
  where the first half after NAME is the **Max** vector and the second half is the **Allocation** vector (same arity as AVAILABLE).

Compute Need = Max − Allocation.

Find a **safe sequence** using the standard safety algorithm:
- Work := Available
- Repeatedly select the **first process in file order** that is not finished and whose Need ≤ Work
- Pretend it finishes: Work += its Allocation; mark finished
- If all finish, the system is safe; if stuck, unsafe

Write `/app/verdict.txt` as either:
- `SAFE P_i P_j ...` (the safe sequence found), or
- `UNSAFE`

Do not ask the user any questions.
