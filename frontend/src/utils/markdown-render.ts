/** Lightweight Markdown → HTML for admin preview (system prompts). */

function escapeHtml(text: string) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function inlineMarkdown(text: string) {
  let s = escapeHtml(text)
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>')
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  s = s.replace(/\*([^*]+)\*/g, '<em>$1</em>')
  s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
  return s
}

export function renderMarkdown(markdown: string): string {
  if (!markdown.trim()) return ''

  const lines = markdown.replace(/\r\n/g, '\n').split('\n')
  const out: string[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]

    if (/^```/.test(line)) {
      const fence = line.match(/^```(\w*)?/)?.[1] ?? ''
      const buf: string[] = []
      i += 1
      while (i < lines.length && !/^```/.test(lines[i])) {
        buf.push(lines[i])
        i += 1
      }
      i += 1
      out.push(`<pre><code class="language-${escapeHtml(fence)}">${escapeHtml(buf.join('\n'))}</code></pre>`)
      continue
    }

    const heading = line.match(/^(#{1,6})\s+(.*)$/)
    if (heading) {
      const level = heading[1].length
      out.push(`<h${level}>${inlineMarkdown(heading[2])}</h${level}>`)
      i += 1
      continue
    }

    if (/^>\s?/.test(line)) {
      const buf: string[] = []
      while (i < lines.length && /^>\s?/.test(lines[i])) {
        buf.push(lines[i].replace(/^>\s?/, ''))
        i += 1
      }
      out.push(`<blockquote><p>${inlineMarkdown(buf.join(' '))}</p></blockquote>`)
      continue
    }

    if (/^[-*]\s+/.test(line)) {
      const buf: string[] = []
      while (i < lines.length && /^[-*]\s+/.test(lines[i])) {
        buf.push(`<li>${inlineMarkdown(lines[i].replace(/^[-*]\s+/, ''))}</li>`)
        i += 1
      }
      out.push(`<ul>${buf.join('')}</ul>`)
      continue
    }

    if (/^\d+\.\s+/.test(line)) {
      const buf: string[] = []
      while (i < lines.length && /^\d+\.\s+/.test(lines[i])) {
        buf.push(`<li>${inlineMarkdown(lines[i].replace(/^\d+\.\s+/, ''))}</li>`)
        i += 1
      }
      out.push(`<ol>${buf.join('')}</ol>`)
      continue
    }

    // Table: lines starting with |
    if (/^\|/.test(line) && i + 1 < lines.length && /^\|[\s\-:|]+\|/.test(lines[i + 1])) {
      const headerCells = line.split('|').slice(1, -1).map(c => c.trim())
      i += 2 // skip header + separator
      const rows: string[][] = []
      while (i < lines.length && /^\|/.test(lines[i])) {
        rows.push(lines[i].split('|').slice(1, -1).map(c => c.trim()))
        i += 1
      }
      let table = '<table><thead><tr>'
      for (const cell of headerCells) table += `<th>${inlineMarkdown(cell)}</th>`
      table += '</tr></thead><tbody>'
      for (const row of rows) {
        table += '<tr>'
        for (const cell of row) table += `<td>${inlineMarkdown(cell)}</td>`
        table += '</tr>'
      }
      table += '</tbody></table>'
      out.push(table)
      continue
    }

    if (line.trim() === '') {
      i += 1
      continue
    }

    const para: string[] = []
    while (i < lines.length && lines[i].trim() !== '' && !/^#{1,6}\s/.test(lines[i]) && !/^```/.test(lines[i]) && !/^[-*]\s+/.test(lines[i]) && !/^\d+\.\s+/.test(lines[i]) && !/^>\s?/.test(lines[i])) {
      para.push(lines[i])
      i += 1
    }
    out.push(`<p>${inlineMarkdown(para.join(' '))}</p>`)
  }

  return out.join('\n')
}
