`/app/app.log` lines look like: `YYYY-MM-DDTHH:MM:SS LEVEL message...`

Filter to `/app/errors.txt` all lines where:
- LEVEL is `ERROR` or `FATAL`
- timestamp is within inclusive window `[2024-06-01T12:00:00, 2024-06-01T13:00:00]`

Preserve original line text and original relative order.

Do not ask the user any questions.
