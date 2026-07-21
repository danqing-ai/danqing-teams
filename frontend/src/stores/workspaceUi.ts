import { defineStore } from 'pinia'
import { ref } from 'vue'

export type RightWorkspaceTab = 'plan' | 'files' | 'memory' | 'changes' | 'terminal' | 'browser'

export const useWorkspaceUiStore = defineStore('workspaceUi', () => {
  const rightTab = ref<RightWorkspaceTab>('plan')
  const changesCount = ref(0)
  const pendingApprovals = ref(0)
  const paletteOpen = ref(false)

  function setRightTab(tab: RightWorkspaceTab) {
    rightTab.value = tab
  }

  function openPalette() {
    paletteOpen.value = true
  }

  function closePalette() {
    paletteOpen.value = false
  }

  function togglePalette() {
    paletteOpen.value = !paletteOpen.value
  }

  return {
    rightTab,
    changesCount,
    pendingApprovals,
    paletteOpen,
    setRightTab,
    openPalette,
    closePalette,
    togglePalette,
  }
})
