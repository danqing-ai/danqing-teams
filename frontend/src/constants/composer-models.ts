// Composer model constants removed — models are now loaded dynamically from LLM store
export interface ComposerModelOption {
  id: string
  label: string
}

export const COMPOSER_MODELS: ComposerModelOption[] = [
  { id: 'openai/gpt-4o', label: 'openai/gpt-4o' },
  { id: 'openai/gpt-4o-mini', label: 'openai/gpt-4o-mini' },
  { id: 'anthropic/claude-sonnet-4', label: 'anthropic/claude-sonnet-4' },
]

export const DEFAULT_COMPOSER_MODEL_ID = COMPOSER_MODELS[0].id
