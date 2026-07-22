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
  /** Parent + selected + sibling summaries for layout context. */
  neighborhoodHTML?: string
  /** Design-relevant computed CSS (property → value). */
  computedStyles?: Record<string, string>
  /** Cropped element screenshot as data URL (jpeg/png). */
  screenshot?: string
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
  neighborhoodHTML: string
  computedStyles: Record<string, string>
  /** Present when inspect captured a crop; also mirrored as a composer image. */
  screenshotDataUrl?: string
  screenshotName?: string
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
  const styleKeys = Object.keys(att.computedStyles || {})
  if (styleKeys.length) {
    lines.push(`Styles: ${styleKeys.slice(0, 8).join(', ')}${styleKeys.length > 8 ? '…' : ''}`)
  }
  if (att.screenshotDataUrl) lines.push('Screenshot: attached')
  return lines.join('\n')
}

function screenshotFileName(tag: string, id: string): string {
  const safeTag = (tag || 'el').replace(/[^a-z0-9_-]/gi, '') || 'el'
  const short = id.replace(/^el_/, '').slice(0, 10)
  return `ui-element-${safeTag}-${short}.jpg`
}

export function fromInspectPayload(
  raw: InspectElementPayload,
  opts: { annotation?: string; sourceFile?: string; pageUrl?: string } = {},
): ElementAttachment {
  const pageUrl = opts.pageUrl || raw.page?.url || ''
  const pageTitle = raw.page?.title || ''
  const sourceFile = opts.sourceFile || raw.page?.sourceFile
  const html = (raw.outerHTML || raw.html || '').slice(0, 1200)
  const neighborhood = (raw.neighborhoodHTML || '').slice(0, 1200)
  const styles = raw.computedStyles && typeof raw.computedStyles === 'object' ? raw.computedStyles : {}
  const id = createElementAttachmentId()
  const screenshot =
    typeof raw.screenshot === 'string' && raw.screenshot.startsWith('data:image/')
      ? raw.screenshot
      : undefined

  return {
    id,
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
    neighborhoodHTML: neighborhood,
    computedStyles: styles,
    ...(screenshot
      ? { screenshotDataUrl: screenshot, screenshotName: screenshotFileName(raw.tag || 'el', id) }
      : {}),
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

function clip(s: string, max: number): string {
  const t = s.trim()
  if (t.length <= max) return t
  return t.slice(0, max) + '…'
}

/** Compact prompt block for the agent — identity + HTML only. */
export function serializeElementAttachment(att: ElementAttachment): string {
  const lines: string[] = ['## Selected UI Element']
  if (att.annotation) lines.push(`Request: ${att.annotation}`)

  const target: string[] = [`<${att.tag}>`]
  if (att.selectors?.css) target.push(att.selectors.css)
  else if (att.elementId) target.push(`#${att.elementId}`)
  else if (att.classes.length) target.push(`.${att.classes.slice(0, 3).join('.')}`)
  const label = (att.text || att.ariaLabel || att.testId || '').trim()
  if (label) target.push(`"${clip(label, 80)}"`)
  lines.push(`Target: ${target.join(' ')}`)

  if (att.page.sourceFile) lines.push(`File: ${att.page.sourceFile}`)
  else if (att.page.url) lines.push(`Page: ${att.page.url}`)

  if (att.component?.name) {
    const loc = att.component.file ? ` (${att.component.file})` : ''
    lines.push(`Component: ${att.component.name}${loc}`)
  }

  const html = clip(att.neighborhoodHTML || att.outerHTML || '', 1200)
  if (html) {
    lines.push('HTML:')
    lines.push('```html')
    lines.push(html)
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
