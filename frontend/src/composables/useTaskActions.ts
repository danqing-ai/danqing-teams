import { inject, type InjectionKey } from 'vue'

export type OpenCreateTaskFn = () => void
export type FocusComposerFn = () => void

export const OPEN_CREATE_TASK_KEY: InjectionKey<OpenCreateTaskFn> = Symbol('openCreateTask')
export const FOCUS_COMPOSER_KEY: InjectionKey<FocusComposerFn> = Symbol('focusComposer')

export function useTaskActions() {
  const openCreateTask = inject(OPEN_CREATE_TASK_KEY, () => {})
  const focusComposer = inject(FOCUS_COMPOSER_KEY, () => {})
  return { openCreateTask, focusComposer }
}
