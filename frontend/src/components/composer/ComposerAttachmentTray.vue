<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  formatBytes,
  type ComposerAttachment,
  type ElementComposerAttachment,
} from '@/types/composer-attachment'
import { chipLabel, chipTooltip } from '@/types/element-attachment'

defineProps<{
  attachments: ComposerAttachment[]
  editingId: string | null
  editingAnnotation: string
}>()

const emit = defineEmits<{
  remove: [id: string]
  'edit-start': [att: ElementComposerAttachment]
  'edit-save': []
  'edit-cancel': []
  'update:editingAnnotation': [value: string]
}>()

const { t } = useI18n()
</script>

<template>
  <div v-if="attachments.length || editingId" class="att-tray">
    <div v-if="attachments.length" class="att-tray__list">
      <div
        v-for="att in attachments"
        :key="att.id"
        class="att-card"
        :class="`att-card--${att.kind}`"
      >
        <!-- Image -->
        <template v-if="att.kind === 'image'">
          <div class="att-card__thumb" :style="{ backgroundImage: `url(${att.dataUrl})` }" />
          <div class="att-card__meta">
            <span class="att-card__name" :title="att.name">{{ att.name }}</span>
            <span class="att-card__sub">{{ formatBytes(att.size) }}</span>
          </div>
        </template>

        <!-- File placeholder -->
        <template v-else-if="att.kind === 'file'">
          <div class="att-card__icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
            </svg>
          </div>
          <div class="att-card__meta">
            <span class="att-card__name" :title="att.name">{{ att.name }}</span>
            <span class="att-card__sub">
              {{ formatBytes(att.size) }}
              <span class="att-card__badge">{{ t('composer.attachPending') }}</span>
            </span>
          </div>
        </template>

        <!-- DOM element -->
        <template v-else>
          <div
            v-if="att.data.screenshotDataUrl"
            class="att-card__thumb"
            :style="{ backgroundImage: `url(${att.data.screenshotDataUrl})` }"
            aria-hidden="true"
          />
          <div v-else class="att-card__icon att-card__icon--el" aria-hidden="true">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="22" y1="12" x2="18" y2="12" />
              <line x1="6" y1="12" x2="2" y2="12" />
              <line x1="12" y1="6" x2="12" y2="2" />
              <line x1="12" y1="22" x2="12" y2="18" />
            </svg>
          </div>
          <div class="att-card__meta" :title="chipTooltip(att.data)">
            <span class="att-card__name">{{ chipLabel(att.data) }}</span>
            <span class="att-card__sub">
              {{ t('composer.attachElement') }}
              <template v-if="att.data.annotation"> · {{ att.data.annotation }}</template>
            </span>
          </div>
          <button
            type="button"
            class="att-card__action"
            :title="t('composer.editAnnotation')"
            @click="emit('edit-start', att)"
          >
            ✎
          </button>
        </template>

        <button
          type="button"
          class="att-card__remove"
          :aria-label="t('composer.removeAttachment')"
          @click="emit('remove', att.id)"
        >
          ×
        </button>
      </div>
    </div>

    <div v-if="editingId" class="att-tray__edit">
      <input
        class="att-tray__edit-input"
        :value="editingAnnotation"
        :placeholder="t('composer.annotationPlaceholder')"
        @input="emit('update:editingAnnotation', ($event.target as HTMLInputElement).value)"
        @keydown.enter.prevent="emit('edit-save')"
        @keydown.esc.prevent="emit('edit-cancel')"
      />
      <button type="button" class="att-tray__edit-btn" @click="emit('edit-save')">{{ t('common.save_') }}</button>
      <button type="button" class="att-tray__edit-btn att-tray__edit-btn--ghost" @click="emit('edit-cancel')">
        {{ t('common.cancel') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.att-tray {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 14px 0;
}

.att-tray__list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.att-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 160px;
  max-width: 240px;
  padding: 6px 28px 6px 6px;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
  background: color-mix(in srgb, var(--dq-label-primary) 4%, transparent);
}

.att-card--image {
  border-color: color-mix(in srgb, var(--dq-accent) 28%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 6%, transparent);
}

.att-card--element {
  border-color: color-mix(in srgb, var(--dq-accent) 22%, transparent);
}

.att-card--file {
  border-style: dashed;
}

.att-card__thumb {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 7px;
  background: var(--dq-bg-base) center/cover no-repeat;
  border: 1px solid color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.att-card__icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 7px;
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  color: var(--dq-label-secondary);
}

.att-card__icon--el {
  color: var(--dq-accent);
}

.att-card__meta {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.att-card__name {
  font-size: var(--dq-font-size-caption);
  font-weight: 600;
  color: var(--dq-label-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.att-card__sub {
  font-size: 10px;
  color: var(--dq-label-tertiary);
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.att-card__badge {
  padding: 0 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--dq-system-orange) 16%, transparent);
  color: var(--dq-system-orange);
  font-weight: 650;
}

.att-card__action,
.att-card__remove {
  border: none;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  padding: 0 4px;
  font-size: 14px;
  line-height: 1;
}

.att-card__remove {
  position: absolute;
  top: 4px;
  right: 4px;
}

.att-card__action:hover,
.att-card__remove:hover {
  color: var(--dq-label-primary);
}

.att-tray__edit {
  display: flex;
  gap: 6px;
  align-items: center;
}

.att-tray__edit-input {
  flex: 1;
  height: 28px;
  padding: 0 8px;
  border-radius: 6px;
  border: 1px solid var(--dq-separator-light);
  background: transparent;
  color: var(--dq-label-primary);
  font-size: var(--dq-font-size-caption);
}

.att-tray__edit-btn {
  border: none;
  border-radius: 6px;
  padding: 0 10px;
  height: 28px;
  background: var(--dq-accent);
  color: var(--dq-color-white);
  cursor: pointer;
  font-size: var(--dq-font-size-caption);
}

.att-tray__edit-btn--ghost {
  background: transparent;
  color: var(--dq-label-secondary);
}
</style>
