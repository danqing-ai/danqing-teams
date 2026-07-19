# Shared base for DanQing Harbor tasks — preinstalls Node (nvm) + OpenCode + Python.
# Build once: make eval-harbor-base
# Tasks use: FROM dq-harbor-base:local
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive \
    NVM_DIR=/root/.nvm

# Check-Valid-Until=false: podman machine clock skew can reject InRelease as "not valid yet".
RUN apt-get -o Acquire::Check-Valid-Until=false -o Acquire::Check-Date=false update -qq \
  && apt-get -o Acquire::Check-Valid-Until=false -o Acquire::Check-Date=false install -y -qq \
       curl ca-certificates python3 python3-venv xz-utils git \
  && rm -rf /var/lib/apt/lists/*

# Match Harbor's OpenCode installer: nvm + Node 22 + opencode-ai (baked at build time).
RUN bash -lc 'set -euo pipefail \
  && curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.2/install.sh | bash \
  && . "$NVM_DIR/nvm.sh" \
  && nvm install 22 \
  && nvm alias default 22 \
  && npm i -g opencode-ai@latest \
  && opencode --version \
  && printf "%s\n" \
       "export NVM_DIR=\"\$HOME/.nvm\"" \
       "[ -s \"\$NVM_DIR/nvm.sh\" ] && . \"\$NVM_DIR/nvm.sh\"" \
       >> /root/.bashrc'

WORKDIR /app
