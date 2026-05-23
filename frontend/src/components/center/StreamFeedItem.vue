<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, MagicStick, Tools } from '@danqing/dq-shell'
import type { StreamFeedItem } from '@/utils/stream-actors'
import { workerInitial } from '@/utils/stream-actors'
import { renderMarkdown } from '@/utils/markdown-render'
import { prepareReportMarkdown } from '@/utils/timeline'

const props = defineProps<{
  item: StreamFeedItem
}>()

const isUser = computed(() => props.item.actorKind === 'user')
const isDispatch = computed(() => Boolean(props.item.dispatchTo))
const showWorkerAvatar = computed(
  () => props.item.actorKind === 'worker' && Boolean(props.item.workerName),
)
const showWorkerName = computed(
  () => props.item.actorKind === 'worker' && Boolean(props.item.workerName) && !isDispatch.value,
)

const hasBubbleBody = computed(() => {
  if (!props.item.richBody) return Boolean(props.item.body?.trim())
  return Boolean(prepareReportMarkdown(props.item.body, props.item.workerName))
})

const showControllerName = computed(
  () =>
    (props.item.actorKind === 'controller' && !isDispatch.value) ||
    (isDispatch.value && hasBubbleBody.value),
)

const bodyHtml = computed(() => {
  if (!props.item.richBody || !props.item.body) return ''
  return renderMarkdown(prepareReportMarkdown(props.item.body, props.item.workerName))
})

const isDispatchOnly = computed(() => isDispatch.value && !hasBubbleBody.value)
</script>

<template>
  <article
    class="stream-row"
    :class="[
      `stream-row--${item.actorKind}`,
      {
        'stream-row--user': isUser,
        'stream-row--dispatch-only': isDispatchOnly,
      },
    ]"
  >
    <div
      class="stream-row__avatar"
      :class="`stream-row__avatar--${item.actorKind}`"
      :title="item.iconTitle"
      :aria-label="item.iconTitle"
    >
      <svg
        v-if="item.actorKind === 'user'"
        class="stream-row__glyph"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        aria-hidden="true"
      >
        <circle cx="12" cy="8" r="3.5" />
        <path d="M6 20c0-3.3 2.7-6 6-6s6 2.7 6 6" stroke-linecap="round" />
      </svg>
      <span v-else-if="showWorkerAvatar" class="stream-row__initial">
        {{ workerInitial(item.workerName!) }}
      </span>
      <DqIcon v-else-if="item.actorKind === 'controller'" class="stream-row__glyph" :size="16">
        <MagicStick />
      </DqIcon>
      <DqIcon v-else-if="item.actorKind === 'worker'" class="stream-row__glyph" :size="16">
        <Tools />
      </DqIcon>
      <svg
        v-else-if="item.actorKind === 'policy'"
        class="stream-row__glyph"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        aria-hidden="true"
      >
        <path d="M12 3 4 7v6c0 4.4 3.4 8.5 8 9 4.6-.5 8-4.6 8-9V7l-8-4z" stroke-linejoin="round" />
        <path d="M12 11v3" stroke-linecap="round" />
        <circle cx="12" cy="9" r="0.5" fill="currentColor" />
      </svg>
      <DqIcon v-else class="stream-row__glyph" :size="16">
        <Tools />
      </DqIcon>
    </div>

    <div class="stream-row__main">
      <div v-if="showControllerName" class="stream-row__sender stream-row__sender--controller">
        <span class="stream-row__sender-name">Team Controller</span>
      </div>
      <div v-else-if="showWorkerName" class="stream-row__sender stream-row__sender--worker">
        <span v-if="item.workerName" class="stream-row__sender-chip">
          {{ workerInitial(item.workerName) }}
        </span>
        <span class="stream-row__sender-name">{{ item.workerName }}</span>
      </div>

      <div
        class="stream-row__bubble"
        :class="{ 'stream-row__bubble--dispatch': isDispatchOnly }"
      >
        <div v-if="isDispatch" class="stream-row__dispatch" :class="{ 'is-solo': isDispatchOnly }">
          <span class="stream-row__dispatch-kind">{{ item.followUp ? '跟进' : '分派' }}</span>
          <DqIcon class="stream-row__dispatch-arrow" :size="12" aria-hidden="true">
            <ArrowRight />
          </DqIcon>
          <span v-if="item.dispatchTo" class="stream-row__dispatch-chip">
            {{ workerInitial(item.dispatchTo) }}
          </span>
          <span class="stream-row__dispatch-target">{{ item.dispatchTo }}</span>
        </div>
        <div
          v-if="bodyHtml"
          class="stream-row__body stream-row__body--md markdown-body"
          v-html="bodyHtml"
        />
        <p v-else-if="item.body" class="stream-row__body">{{ item.body }}</p>
        <div v-if="item.approvalId" class="stream-row__actions">
          <slot name="approval-actions" />
        </div>
      </div>
    </div>
  </article>
</template>
