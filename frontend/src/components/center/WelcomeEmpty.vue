<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useLLMStore } from '@/stores/llm'
import { useProjectsStore } from '@/stores/projects'
import { useSessionsStore } from '@/stores/sessions'

const emit = defineEmits<{
  pickPrompt: [text: string]
}>()

const { t } = useI18n()
const router = useRouter()
const llm = useLLMStore()
const projects = useProjectsStore()
const sessions = useSessionsStore()

const hasModels = computed(() => llm.modelsLoaded && llm.models.length > 0)
const hasProjects = computed(() => projects.sortedProjects.length > 0)

const prompts = computed(() => [
  t('sessions.welcomePrompt1'),
  t('sessions.welcomePrompt2'),
  t('sessions.welcomePrompt3'),
])

function onPick(text: string) {
  emit('pickPrompt', text)
}

function goSettings() {
  router.push({ name: 'settings' })
}
</script>

<template>
  <div class="welcome-empty">
    <div class="welcome-empty__hero">
      <h2 class="welcome-empty__title">{{ t('sessions.welcomeTitle') }}</h2>
      <p class="welcome-empty__subtitle">{{ t('sessions.welcomeSubtitle') }}</p>
    </div>

    <div v-if="!hasModels" class="welcome-empty__alert">
      <p>{{ t('sessions.welcomeNeedModel') }}</p>
      <DqButton type="primary" size="small" @click="goSettings">{{ t('sessions.welcomeConfigureModel') }}</DqButton>
    </div>

    <div v-else-if="!hasProjects" class="welcome-empty__alert">
      <p>{{ t('sessions.welcomeNeedProject') }}</p>
    </div>

    <div v-else class="welcome-empty__steps">
      <ol class="welcome-empty__list">
        <li>{{ t('sessions.welcomeStep1') }}</li>
        <li>{{ t('sessions.welcomeStep2') }}</li>
        <li>{{ t('sessions.welcomeStep3') }}</li>
      </ol>
      <div class="welcome-empty__chips">
        <button
          v-for="(p, i) in prompts"
          :key="i"
          type="button"
          class="welcome-empty__chip"
          @click="onPick(p)"
        >
          {{ p }}
        </button>
      </div>
      <p class="welcome-empty__hint">
        {{ sessions.selectedProjectId
          ? t('sessions.welcomeHintReady')
          : t('sessions.welcomeHintPickProject') }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.welcome-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 20px;
  min-height: 280px;
  padding: 40px 24px 120px;
  text-align: center;
}

.welcome-empty__hero {
  max-width: 420px;
}

.welcome-empty__title {
  margin: 0;
  font-size: var(--dq-font-size-title);
  font-weight: 700;
  color: var(--dq-label-primary);
  letter-spacing: -0.02em;
}

.welcome-empty__subtitle {
  margin: 8px 0 0;
  font-size: var(--dq-font-size-body);
  color: var(--dq-label-secondary);
  line-height: 1.5;
}

.welcome-empty__alert {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  max-width: 360px;
  padding: 16px 20px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--dq-system-orange) 10%, transparent);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-body);
}

.welcome-empty__steps {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  max-width: 480px;
  width: 100%;
}

.welcome-empty__list {
  margin: 0;
  padding: 0 0 0 1.2em;
  text-align: left;
  color: var(--dq-label-secondary);
  font-size: var(--dq-font-size-footnote);
  line-height: 1.7;
}

.welcome-empty__chips {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}

.welcome-empty__chip {
  max-width: 280px;
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 12%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-footnote);
  line-height: 1.4;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.welcome-empty__chip:hover {
  border-color: color-mix(in srgb, var(--dq-accent) 40%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 8%, transparent);
}

.welcome-empty__hint {
  margin: 0;
  font-size: var(--dq-font-size-caption);
  color: var(--dq-label-tertiary);
}
</style>
