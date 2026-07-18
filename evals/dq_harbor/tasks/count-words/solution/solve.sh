#!/bin/bash
set -euo pipefail
wc -w </app/essay.txt | tr -d ' ' >/app/word_count.txt
