import { inject, type InjectionKey } from 'vue'

export type OpenCreateSessionFn = () => void
export type FocusComposerFn = () => void

export const OPEN_CREATE_SESSION_KEY: InjectionKey<OpenCreateSessionFn> = Symbol('openCreateSession')
export const FOCUS_COMPOSER_KEY: InjectionKey<FocusComposerFn> = Symbol('focusComposer')

export function useSessionActions() {
  const openCreateSession = inject(OPEN_CREATE_SESSION_KEY, () => {})
  const focusComposer = inject(FOCUS_COMPOSER_KEY, () => {})
  return { openCreateSession, focusComposer }
}
