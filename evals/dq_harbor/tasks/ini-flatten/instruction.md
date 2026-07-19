Flatten `/app/config.ini` into `/app/app.env` as KEY=VALUE lines.

Rules:
- Sections become prefixes: `[db]` key `host` → `DB_HOST`
- Section and key names uppercased; non-alnum → `_`
- Unsectioned keys use no prefix
- Keep file order: unsectioned keys first (file order), then sections in file order with their keys
- No spaces around `=`. Skip blank/comment lines (`;` or `#`).

Do not ask the user any questions.
