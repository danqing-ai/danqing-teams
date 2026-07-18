FROM ubuntu:24.04
RUN apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq python3 >/dev/null \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY package.json /app/package.json
