Redact PII in `/app/message.txt` and write the result to `/app/redacted.txt`.

Rules (apply to the whole file, including across punctuation):
1. Email addresses (`local@domain` with a dot in domain) → `[EMAIL]`
2. US phone numbers of form `NNN-NNN-NNNN` → `[PHONE]`
3. SSN of form `NNN-NN-NNNN` → `[SSN]`

Do not change any other characters. Preserve newlines. Do not ask the user any questions.
