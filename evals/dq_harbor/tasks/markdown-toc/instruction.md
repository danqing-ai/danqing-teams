Generate a table of contents for `/app/doc.md` into `/app/toc.md`.

Rules:
- Include only ATX headings `#` .. `######` at line start.
- Output lines as: `N: TITLE` where N is the heading level (1-6) and TITLE is the heading text trimmed.
- Preserve document order. Ignore Setext headings and headings inside fenced code blocks (``` ... ```).

Do not ask the user any questions.
