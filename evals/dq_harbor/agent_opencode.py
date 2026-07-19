"""Harbor OpenCode agent that skips nvm/npm when the base image already has OpenCode.

Use with task images based on ``dq-harbor-base:local`` (see ``images/base/``).

    PYTHONPATH=evals harbor run ... --agent dq_harbor.agent_opencode:OpenCodePrebuilt
"""

from __future__ import annotations

import re
from typing import override

from harbor.agents.installed.opencode import OpenCode
from harbor.environments.base import BaseEnvironment

_ANSI = re.compile(r"\x1b\[[0-9;]*m")


class OpenCodePrebuilt(OpenCode):
    """OpenCode with install short-circuit for prebaked Node/OpenCode images."""

    @override
    async def install(self, environment: BaseEnvironment) -> None:
        check = await environment.exec(
            command=(
                'bash -lc \''
                'export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"; '
                '[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"; '
                'command -v opencode >/dev/null && opencode --version'
                '\''
            ),
        )
        if check.return_code == 0:
            raw = _ANSI.sub("", check.stdout or "")
            ver = raw.strip().splitlines()[-1:] or ["unknown"]
            self.logger.info("OpenCode preinstalled (%s); skipping nvm/npm install", ver[0])
            return
        self.logger.warning(
            "OpenCode not found in image (rc=%s); falling back to Harbor install",
            check.return_code,
        )
        await super().install(environment)
