import { onMounted, onUnmounted, ref } from 'vue'

export type ResizeEdge = 'left' | 'right'

export function useResizableWidth(
  storageKey: string,
  defaultWidth: number,
  min: number,
  max: number,
  edge: ResizeEdge = 'left',
) {
  const width = ref(defaultWidth)

  onMounted(() => {
    const saved = Number(localStorage.getItem(storageKey))
    if (!Number.isNaN(saved) && saved >= min && saved <= max) {
      width.value = saved
    }
  })

  function persist() {
    localStorage.setItem(storageKey, String(Math.round(width.value)))
  }

  function onResizePointerDown(event: PointerEvent) {
    event.preventDefault()
    const startX = event.clientX
    const startWidth = width.value

    const onMove = (e: PointerEvent) => {
      const delta = e.clientX - startX
      const next =
        edge === 'left'
          ? startWidth + delta
          : startWidth - delta
      width.value = Math.min(max, Math.max(min, next))
    }

    const onUp = () => {
      persist()
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      document.body.classList.remove('teams-is-resizing')
    }

    document.body.classList.add('teams-is-resizing')
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  onUnmounted(() => {
    document.body.classList.remove('teams-is-resizing')
  })

  return { width, onResizePointerDown }
}
