<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { apiBaseUrl } from '@/utils/desktop'

const props = defineProps<{ projectId: string }>()

const containerRef = ref<HTMLElement | null>(null)
const status = ref<'connecting' | 'connected' | 'closed'>('connecting')

let term: Terminal | null = null
let fit: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null

function wsUrl(): string {
  const httpBase = apiBaseUrl() || window.location.origin
  const u = new URL(`${httpBase}/api/v1/projects/${props.projectId}/terminal`)
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  return u.toString()
}

function sendResize() {
  if (!term || !ws || ws.readyState !== WebSocket.OPEN) return
  ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
}

function connect() {
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  status.value = 'connecting'
  const socket = new WebSocket(wsUrl())
  socket.binaryType = 'arraybuffer'
  socket.onopen = () => {
    status.value = 'connected'
    fit?.fit()
    sendResize()
    term?.focus()
  }
  socket.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      term?.write(ev.data)
    } else {
      term?.write(new Uint8Array(ev.data as ArrayBuffer))
    }
  }
  socket.onclose = () => {
    if (ws !== socket) return
    status.value = 'closed'
    term?.write('\r\n\x1b[90m[终端连接已断开]\x1b[0m\r\n')
  }
  ws = socket
}

function reconnect() {
  term?.reset()
  connect()
}

onMounted(() => {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 12,
    fontFamily: '"SF Mono", Monaco, Menlo, Consolas, "Liberation Mono", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
    },
    scrollback: 5000,
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  if (containerRef.value) {
    term.open(containerRef.value)
    fit.fit()
  }
  term.onData((data) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    }
  })
  term.onResize(() => sendResize())

  resizeObserver = new ResizeObserver(() => {
    if (!containerRef.value) return
    const { width, height } = containerRef.value.getBoundingClientRect()
    if (width > 0 && height > 0) fit?.fit()
  })
  if (containerRef.value) resizeObserver.observe(containerRef.value)

  connect()
})

watch(() => props.projectId, () => {
  term?.reset()
  connect()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
  term?.dispose()
  term = null
  fit = null
})
</script>

<template>
  <div class="terminal-panel">
    <div ref="containerRef" class="terminal-panel__term" />
    <div v-if="status === 'closed'" class="terminal-panel__overlay">
      <button class="terminal-panel__reconnect" @click="reconnect">重新连接</button>
    </div>
  </div>
</template>

<style scoped>
.terminal-panel {
  position: relative;
  flex: 1;
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #1e1e1e;
}

.terminal-panel__term {
  flex: 1;
  min-height: 0;
  padding: 6px 0 6px 8px;
}

.terminal-panel__term :deep(.xterm) {
  height: 100%;
}

.terminal-panel__overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
}

.terminal-panel__reconnect {
  padding: 6px 16px;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.08);
  color: #d4d4d4;
  font-size: var(--dq-font-size-footnote);
  cursor: pointer;
  transition: background 0.15s;
}

.terminal-panel__reconnect:hover {
  background: rgba(255, 255, 255, 0.16);
}
</style>
