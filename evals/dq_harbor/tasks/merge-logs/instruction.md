Two log files exist: `/app/a.log` and `/app/b.log`. Each line starts with an ISO-like timestamp `YYYY-MM-DDTHH:MM:SS` then a space then a message.

Merge both files into `/app/merged.log`, sorted by timestamp ascending. If timestamps tie, keep `a.log` lines before `b.log` lines for the same timestamp. Preserve the full original lines.

Do not ask the user any questions.
