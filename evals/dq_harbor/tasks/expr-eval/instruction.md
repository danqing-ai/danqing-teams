`/app/vars.env` has integer variables as `NAME=VALUE` (VALUE may be negative).
`/app/exprs.txt` has one arithmetic expression per line.

Evaluate each expression with these rules:
- Operators: `+`, `-`, `*`, `/`, parentheses `()`.
- `/` is **integer division toward zero** (e.g. `7/3=2`, `-7/3=-2`).
- Unary minus is allowed (e.g. `-z`, `x * -2`, `x - -y`).
- `*` and `/` bind tighter than `+` and `-`; unary minus binds tighter than binary ops.
- No spaces are required; spaces may appear between tokens.
- Identifiers are variable names from `vars.env`.

Write `/app/values.txt` with one integer result per input line (same order), no spaces.

Do not ask the user any questions.
