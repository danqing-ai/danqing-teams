#!/usr/bin/env python3
"""Summarize compare_agents.sh logs into a markdown table.

Pass/fail is based on Harbor Mean reward (1.0 = PASS), not process exit code.
Harbor often exits 0 even when reward is 0.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


TASK_BLOCK = re.compile(
    r"======== task=(\S+) agent=\S+ ========.*?^\s*1/1 Mean:\s*([0-9.]+)",
    re.MULTILINE | re.DOTALL,
)


def parse_log(path: Path) -> dict:
    text = path.read_text(encoding="utf-8", errors="replace")
    results: dict[str, str] = {}
    means: dict[str, float] = {}
    for m in TASK_BLOCK.finditer(text):
        task, mean_s = m.group(1), m.group(2)
        mean = float(mean_s)
        means[task] = mean
        results[task] = "PASS" if mean >= 1.0 else "FAIL"
    p = sum(1 for v in results.values() if v == "PASS")
    f = sum(1 for v in results.values() if v == "FAIL")
    return {
        "tasks": results,
        "means": means,
        "summary": {"pass": p, "fail": f, "total": p + f},
        "path": str(path),
    }


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: summarize_compare.py <compare_results_dir>", file=sys.stderr)
        return 2
    root = Path(sys.argv[1])
    agents = ["danqing", "opencode", "openhands"]
    parsed = {}
    for a in agents:
        log = root / f"{a}.log"
        if log.is_file():
            parsed[a] = parse_log(log)

    all_tasks = sorted({t for p in parsed.values() for t in p["tasks"]})
    lines = [
        "# Terminal-Bench 2.0 compare",
        "",
        f"Dir: `{root}`",
        "",
        "Synced TB2 + `dq-harbor-base:local`. Pass = Harbor Mean reward ≥ 1.0",
        "",
        "| Task | DanQing | OpenCode | OpenHands |",
        "|------|---------|----------|-----------|",
    ]
    for t in all_tasks:
        row = [t]
        for a in agents:
            row.append(parsed.get(a, {}).get("tasks", {}).get(t, "—"))
        lines.append("| " + " | ".join(row) + " |")

    lines.append("")
    lines.append("## Totals")
    lines.append("")
    lines.append("| Agent | Pass | Fail | Total |")
    lines.append("|-------|------|------|-------|")
    for a in agents:
        s = parsed.get(a, {}).get("summary")
        if s:
            lines.append(f"| {a} | {s['pass']} | {s['fail']} | {s['total']} |")
        else:
            lines.append(f"| {a} | — | — | — |")

    lines.append("")
    lines.append("## Logs")
    lines.append("")
    for a in agents:
        p = parsed.get(a, {}).get("path")
        if p:
            lines.append(f"- `{a}`: `{p}`")

    print("\n".join(lines))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
