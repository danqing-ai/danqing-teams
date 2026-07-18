#!/usr/bin/env python3
"""Summarize DanQing Harbor trial turn logs for failure analysis.

Scans ~/.harbor/jobs (or a given job/trial path) for:
  agent/turnlogs/analysis.json
  agent/FAILURE_ANALYSIS.md
  agent/report.json

Usage:
  python3 evals/dq_harbor/analyze_failures.py
  python3 evals/dq_harbor/analyze_failures.py ~/.harbor/jobs/<job>
  python3 evals/dq_harbor/analyze_failures.py --failed-only
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


def find_trials(root: Path) -> list[Path]:
    trials: list[Path] = []
    if (root / "trials").is_dir():
        trials.extend(sorted(p for p in (root / "trials").iterdir() if p.is_dir()))
    elif (root / "agent").is_dir() or (root / "result.json").is_file():
        trials.append(root)
    else:
        # jobs root: scan each job
        for job in sorted(root.iterdir() if root.is_dir() else []):
            if (job / "trials").is_dir():
                trials.extend(sorted(p for p in (job / "trials").iterdir() if p.is_dir()))
    return trials


def reward_of(trial: Path) -> str | None:
    for cand in (
        trial / "reward.txt",
        trial / "verifier" / "reward.txt",
        trial / "logs" / "verifier" / "reward.txt",
    ):
        if cand.is_file():
            return cand.read_text(encoding="utf-8", errors="replace").strip()
    # Harbor sometimes stores in result.json
    result = trial / "result.json"
    if result.is_file():
        try:
            data = json.loads(result.read_text(encoding="utf-8"))
            if "reward" in data:
                return str(data["reward"])
            if isinstance(data.get("verifier_result"), dict) and "reward" in data["verifier_result"]:
                return str(data["verifier_result"]["reward"])
        except json.JSONDecodeError:
            pass
    return None


def load_analysis(trial: Path) -> dict | None:
    for cand in (
        trial / "agent" / "turnlogs" / "analysis.json",
        trial / "logs" / "agent" / "turnlogs" / "analysis.json",
        trial / "agent" / "analysis.json",
    ):
        if cand.is_file():
            try:
                return json.loads(cand.read_text(encoding="utf-8"))
            except json.JSONDecodeError:
                return {"parse_error": str(cand)}
    return None


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "path",
        nargs="?",
        default=str(Path.home() / ".harbor" / "jobs"),
        help="Harbor jobs dir, single job, or trial path",
    )
    ap.add_argument("--failed-only", action="store_true", help="only trials with reward!=1")
    args = ap.parse_args()
    root = Path(args.path).expanduser()
    if not root.exists():
        print(f"path not found: {root}", file=sys.stderr)
        return 1

    trials = find_trials(root)
    if not trials:
        print(f"no trials under {root}", file=sys.stderr)
        return 1

    failed = 0
    ok = 0
    missing_logs = 0
    print(f"scanning {len(trials)} trial(s) under {root}\n")

    for trial in trials:
        reward = reward_of(trial)
        analysis = load_analysis(trial)
        is_fail = reward not in ("1", "1.0", "1.00")
        if args.failed_only and not is_fail:
            continue

        status = "FAIL" if is_fail else "PASS"
        if is_fail:
            failed += 1
        else:
            ok += 1

        print(f"## [{status}] {trial}")
        print(f"   reward={reward!r}")
        if analysis is None:
            missing_logs += 1
            print("   turnlogs: MISSING (no analysis.json)")
        else:
            reasons = analysis.get("failureReasons") or []
            tools = analysis.get("toolErrors") or []
            print(f"   report={analysis.get('reportStatus')!r} model={analysis.get('model')!r}")
            if reasons:
                print("   reasons:")
                for r in reasons[:12]:
                    print(f"     - {r}")
            if tools:
                print("   tool errors:")
                for t in tools[:8]:
                    print(f"     - {t}")
            md = trial / "agent" / "FAILURE_ANALYSIS.md"
            if not md.is_file():
                md = trial / "agent" / "turnlogs" / "analysis.md"
            if md.is_file():
                print(f"   details: {md}")
        print()

    print(f"summary: pass={ok} fail={failed} missing_turnlogs={missing_logs}")
    return 0 if failed == 0 or not args.failed_only else 1


if __name__ == "__main__":
    raise SystemExit(main())
