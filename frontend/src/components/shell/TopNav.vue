<script setup lang="ts">
import { Grid, MagicStick, Setting } from '@danqing/dq-shell'

export type AppModule = 'teams' | 'agents' | 'settings'

defineProps<{
  activeModule: AppModule
}>()

const emit = defineEmits<{
  navigate: [module: AppModule]
}>()

const navItems: { id: AppModule; icon: typeof Grid; label: string }[] = [
  { id: 'teams', icon: Grid, label: 'Teams' },
  { id: 'agents', icon: MagicStick, label: 'Agents' },
  { id: 'settings', icon: Setting, label: '设置' },
]

function onNavSelect(id: AppModule) {
  emit('navigate', id)
}
</script>

<template>
  <header class="dq-top-nav-bar teams-top-nav">
    <div class="header-brand">
      <DqIcon class="dq-top-nav-brand-icon" :size="28"><Grid /></DqIcon>
      <span class="brand-title">DanQing Teams</span>
    </div>

    <nav class="nav-menu dq-top-nav-menu" role="navigation" aria-label="主模块">
      <button
        v-for="item in navItems"
        :key="item.id"
        type="button"
        class="dq-top-nav-menu__item"
        :class="{ 'is-active': activeModule === item.id }"
        @click="onNavSelect(item.id)"
      >
        <DqIcon><component :is="item.icon" /></DqIcon>
        <span>{{ item.label }}</span>
      </button>
    </nav>

    <div class="header-actions" aria-hidden="true" />
  </header>
</template>
