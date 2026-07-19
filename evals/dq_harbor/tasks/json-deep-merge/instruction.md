Deep-merge `/app/base.json` and `/app/override.json` into `/app/merged.json`.

Rules:
- Objects merge recursively by key.
- For conflicting scalars / arrays: **override wins**.
- Keys only in base are kept.
- Pretty-print is optional; values/types must match.

Do not ask the user any questions.
