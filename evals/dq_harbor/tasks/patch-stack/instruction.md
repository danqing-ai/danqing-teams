`/app/doc.txt` is the starting file.
`/app/patches/` contains unified diffs named `001.diff`, `002.diff`, ... (zero-padded, sorted ascending).

Apply **every** patch in sorted filename order onto the working copy of `doc.txt`.
Each diff is a normal unified diff against the **current** content after previous patches (not against the original).

Write the final file to `/app/final.txt` (exact bytes/lines).

Do not ask the user any questions.
