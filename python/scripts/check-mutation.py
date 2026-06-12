#!/usr/bin/env python3
"""Run mutmut and fail below the kill-rate floor.

mutmut exits 0 regardless of survivors, so this gate parses the final run
statistics and enforces a minimum kill rate: killed plus timed-out mutants
over everything that could have been caught (survivors and uncovered
mutants count against the rate; skipped and suspicious mutants do not).
The floor ratchets up as survivors are triaged, never down.

    uv run python scripts/check-mutation.py --min-kill-rate 74
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys

STATS = re.compile(
    r"(?P<done>\d+)/(?P<total>\d+)\s+🎉 (?P<killed>\d+)\s+🫥 (?P<uncovered>\d+)"
    r"\s+⏰ (?P<timeout>\d+)\s+🤔 (?P<suspicious>\d+)\s+🙁 (?P<survived>\d+)"
    r"\s+🔇 (?P<skipped>\d+)"
)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--min-kill-rate", type=float, required=True)
    arguments = parser.parse_args()

    completed = subprocess.run(
        ["mutmut", "run"],
        capture_output=True,
        text=True,
        check=False,
    )
    output = completed.stdout + completed.stderr
    if completed.returncode != 0:
        sys.stderr.write(output)
        sys.stderr.write(
            f"mutation gate failed: mutmut exited {completed.returncode}\n"
        )
        return 2
    stats = last_stats(output)
    if stats is None:
        sys.stderr.write(output)
        sys.stderr.write("mutation gate failed: no mutmut statistics found\n")
        return 2
    if stats["done"] != stats["total"]:
        sys.stderr.write(
            f"mutation gate failed: only {stats['done']} of {stats['total']} mutants ran\n"
        )
        return 2

    caught = stats["killed"] + stats["timeout"]
    catchable = caught + stats["survived"] + stats["uncovered"]
    if catchable == 0:
        sys.stderr.write("mutation gate failed: no mutants were generated\n")
        return 2
    rate = 100 * caught / catchable
    line = (
        f"mutation kill rate: {rate:.1f}% "
        f"({caught} caught of {catchable} catchable; "
        f"{stats['survived']} survived, {stats['uncovered']} uncovered)\n"
    )
    sys.stdout.write(line)
    if rate < arguments.min_kill_rate:
        sys.stderr.write(
            f"mutation gate failed: kill rate {rate:.1f}% is below the "
            f"{arguments.min_kill_rate:.1f}% floor\n"
        )
        return 1
    return 0


def last_stats(output: str) -> dict[str, int] | None:
    matches = list(STATS.finditer(output.replace("\r", "\n")))
    if not matches:
        return None
    last = matches[-1]
    return {key: int(value) for key, value in last.groupdict().items()}


if __name__ == "__main__":
    raise SystemExit(main())
