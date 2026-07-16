/** Structured DOM element attachment for Composer chips. */

export interface ElementBoundingBox {
  x: number
  y: number
  w: number
  h: number
}

export interface ElementViewport {
  w: number
  h: number
}

export interface ElementSelectors {
  css: string
  fallbacks: string[]
}

export interface ElementComponentHint {
  name: string
  file?: string | null
  framework?: string | null
}

export interface ElementPageContext {
  url: string
  title: string
  /** Project-relative source file when browsing via /projects/:id/raw/... */
  sourceFile?: string
}

/** Payload from iframe inspect script (before user annotation). */
export interface InspectElementPayload {
  type?: 'dq-inspect-selected'
  tag: string
  text?: string
  outerHTML?: string
  html?: string
  id?: string
  classes?: string[]
  role?: string
  ariaLabel?: string
  name?: string
  placeholder?: string
  testId?: string
  selectors?: ElementSelectors
  xpath?: string
  boundingBox?: ElementBoundingBox
  viewport?: ElementViewport
  attributes?: Record<string, string>
  component?: ElementComponentHint | null
  page?: ElementPageContext
}

export interface ElementAttachment {
  id: string
  kind: 'dom-element'
  annotation: string
  page: ElementPageContext
  tag: string
  text: string
  outerHTML: string
  elementId: string
  classes: string[]
  role: string
  ariaLabel: string
  name: string
  placeholder: string
  testId: string
  selectors: ElementSelectors
  xpath: string
  boundingBox: ElementBoundingBox
  viewport: ElementViewport
  attributes: Record<string, string>
  component: ElementComponentHint | null
}

export function createElementAttachmentId(): string {
  return `el_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
}

export function chipLabel(att: ElementAttachment): string {
  const tag = att.tag || 'el'
  const text = (att.text || att.ariaLabel || att.testId || '').trim()
  if (text) {
    const short = text.length > 18 ? text.slice(0, 18) + '…' : text
    return `<${tag}> ${short}`
  }
  if (att.selectors?.css) {
    const sel = att.selectors.css
    return `<${tag}> ${sel.length > 22 ? sel.slice(0, 22) + '…' : sel}`
  }
  return `<${tag}>`
}

export function chipTooltip(att: ElementAttachment): string {
  const lines: string[] = []
  if (att.annotation) lines.push(`批注: ${att.annotation}`)
  if (att.page?.url || att.page?.title) {
    lines.push(`页面: ${att.page.title || att.page.url}`)
  }
  if (att.page?.sourceFile) lines.push(`文件: ${att.page.sourceFile}`)
  lines.push(`Tag: <${att.tag}>`)
  if (att.role) lines.push(`Role: ${att.role}`)
  if (att.text) lines.push(`Text: ${att.text.slice(0, 120)}`)
  if (att.selectors?.css) lines.push(`CSS: ${att.selectors.css}`)
  if (att.xpath) lines.push(`XPath: ${att.xpath}`)
  if (att.boundingBox) {
    const b = att.boundingBox
    lines.push(`Box: ${b.x},${b.y} ${b.w}×${b.h}`)
  }
  if (att.component?.name) {
    const loc = att.component.file ? ` @ ${att.component.file}` : ''
    lines.push(`Component: ${att.component.name}${loc}`)
  }
  return lines.join('\n')
}

export function fromInspectPayload(
  raw: InspectElementPayload,
  opts: { annotation?: string; sourceFile?: string; pageUrl?: string } = {},
): ElementAttachment {
  const pageUrl = opts.pageUrl || raw.page?.url || ''
  const pageTitle = raw.page?.title || ''
  const sourceFile = opts.sourceFile || raw.page?.sourceFile
  const html = (raw.outerHTML || raw.html || '').slice(0, 1500)

  return {
    id: createElementAttachmentId(),
    kind: 'dom-element',
    annotation: (opts.annotation || '').trim(),
    page: {
      url: pageUrl,
      title: pageTitle,
      ...(sourceFile ? { sourceFile } : {}),
    },
    tag: (raw.tag || 'unknown').toLowerCase(),
    text: (raw.text || '').trim(),
    outerHTML: html,
    elementId: raw.id || '',
    classes: Array.isArray(raw.classes) ? raw.classes : [],
    role: raw.role || '',
    ariaLabel: raw.ariaLabel || '',
    name: raw.name || '',
    placeholder: raw.placeholder || '',
    testId: raw.testId || '',
    selectors: {
      css: raw.selectors?.css || '',
      fallbacks: raw.selectors?.fallbacks || [],
    },
    xpath: raw.xpath || '',
    boundingBox: raw.boundingBox || { x: 0, y: 0, w: 0, h: 0 },
    viewport: raw.viewport || { w: 0, h: 0 },
    attributes: raw.attributes || {},
    component: raw.component ?? null,
  }
}

export function serializeElementAttachment(att: ElementAttachment): string {
  const lines: string[] = ['### Attached UI Element']
  if (att.annotation) lines.push(`- Annotation: ${att.annotation}`)
  if (att.page.url) lines.push(`- Page: ${att.page.url}`)
  if (att.page.title) lines.push(`- Title: ${att.page.title}`)
  if (att.page.sourceFile) lines.push(`- Source: ${att.page.sourceFile}`)

  const tagParts = [`Tag: ${att.tag}`]
  if (att.role) tagParts.push(`Role: ${att.role}`)
  if (att.text) tagParts.push(`Text: "${att.text.slice(0, 200)}"`)
  if (att.ariaLabel) tagParts.push(`Aria: "${att.ariaLabel}"`)
  if (att.testId) tagParts.push(`TestId: ${att.testId}`)
  lines.push(`- ${tagParts.join(' | ')}`)

  if (att.elementId) lines.push(`- Id: #${att.elementId}`)
  if (att.classes.length) lines.push(`- Classes: ${att.classes.slice(0, 12).join(' ')}`)
  if (att.selectors.css) lines.push(`- CSS: ${att.selectors.css}`)
  if (att.selectors.fallbacks?.length) {
    lines.push(`- CSS fallbacks: ${att.selectors.fallbacks.slice(0, 3).join(' | ')}`)
  }
  if (att.xpath) lines.push(`- XPath: ${att.xpath}`)

  const b = att.boundingBox
  const v = att.viewport
  if (b.w || b.h) {
    lines.push(
      `- Box: x=${b.x} y=${b.y} w=${b.w} h=${b.h}` +
        (v.w || v.h ? ` (viewport ${v.w}x${v.h})` : ''),
    )
  }

  if (att.component?.name) {
    const loc = att.component.file ? ` @ ${att.component.file}` : ''
    const fw = att.component.framework ? ` (${att.component.framework})` : ''
    lines.push(`- Component: ${att.component.name}${loc}${fw}`)
  }

  const attrKeys = Object.keys(att.attributes || {})
  if (attrKeys.length) {
    const attrs = attrKeys
      .slice(0, 12)
      .map((k) => `${k}="${att.attributes[k]}"`)
      .join(' ')
    lines.push(`- Attrs: ${attrs}`)
  }

  if (att.outerHTML) {
    lines.push('- HTML:')
    lines.push('```html')
    lines.push(att.outerHTML)
    lines.push('```')
  }

  return lines.join('\n')
}

export function serializeElementAttachments(atts: ElementAttachment[]): string {
  if (!atts.length) return ''
  return atts.map(serializeElementAttachment).join('\n\n')
}

export function buildUserInputWithAttachments(
  text: string,
  atts: ElementAttachment[],
): string {
  const body = text.trim()
  const blocks = serializeElementAttachments(atts)
  if (body && blocks) return `${body}\n\n${blocks}`
  return body || blocks
}
