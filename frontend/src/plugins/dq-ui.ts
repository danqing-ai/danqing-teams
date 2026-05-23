import type { App } from 'vue'
import { installDanQingFeedback } from '@danqing/dq-shell'
import {
  DqButton,
  DqCheckbox,
  DqDialog,
  DqDropdown,
  DqDropdownItem,
  DqDropdownMenu,
  DqEmpty,
  DqIcon,
  DqIconButton,
  DqInput,
  DqSectionTabPanel,
  DqSectionTabs,
  DqSectionTabTrigger,
  DqSurfaceCard,
  DqTag,
} from '@danqing/dq-shell'

const DQ_COMPONENTS = {
  DqSectionTabs,
  DqSectionTabTrigger,
  DqSectionTabPanel,
  DqButton,
  DqDialog,
  DqDropdown,
  DqDropdownMenu,
  DqDropdownItem,
  DqEmpty,
  DqIcon,
  DqIconButton,
  DqTag,
  DqSurfaceCard,
  DqInput,
  DqCheckbox,
} as const

export function installDanQingUi(app: App) {
  installDanQingFeedback(app)
  for (const [name, component] of Object.entries(DQ_COMPONENTS)) {
    app.component(name, component)
  }
}
