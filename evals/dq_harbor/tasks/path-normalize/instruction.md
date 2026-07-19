`/app/paths.txt` has one absolute or relative POSIX path per line (no spaces).

Normalize each path like a simpler `realpath` without needing the filesystem to exist:
- resolve `.` and `..`
- collapse repeated `/`
- result for absolute inputs stays absolute; relative inputs stay relative (may become `.` if empty)
- do **not** resolve symlinks

Write normalized paths to `/app/normalized.txt`, same order, one per line.

Do not ask the user any questions.
