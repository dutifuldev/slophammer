import { describe, expect, it } from "vitest";

import { formatText, groupFindings, rangesOverlap } from "../src/dry/report.js";
import type { DryFinding } from "../src/dry/types.js";

describe("DRY report helpers", () => {
  it("groups overlapping copied blocks", () => {
    const groups = groupFindings([
      dryFinding("a.ts", 1, 10, "b.ts", 1, 10),
      dryFinding("a.ts", 8, 15, "b.ts", 9, 18)
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]?.findings).toEqual([0, 1]);
    expect(groups[0]?.left).toEqual({ path: "a.ts", start_line: 1, end_line: 15 });
    expect(groups[0]?.right).toEqual({ path: "b.ts", start_line: 1, end_line: 18 });
  });

  it("formats text output", () => {
    const findings = [dryFinding("a.ts", 1, 3, "b.ts", 5, 7)];
    const report = { findings, groups: groupFindings(findings) };

    expect(formatText(report)).toContain("DRY dry-1 [copied-block]");
  });

  it("detects range overlap only inside the same file", () => {
    expect(
      rangesOverlap(
        { path: "a.ts", start_line: 1, end_line: 3 },
        { path: "a.ts", start_line: 3, end_line: 5 }
      )
    ).toBe(true);
    expect(
      rangesOverlap(
        { path: "a.ts", start_line: 1, end_line: 3 },
        { path: "b.ts", start_line: 3, end_line: 5 }
      )
    ).toBe(false);
  });
});

function dryFinding(
  leftPath: string,
  leftStart: number,
  leftEnd: number,
  rightPath: string,
  rightStart: number,
  rightEnd: number
): DryFinding {
  return {
    kind: "copied-block",
    left: { path: leftPath, start_line: leftStart, end_line: leftEnd },
    right: { path: rightPath, start_line: rightStart, end_line: rightEnd },
    tokens: 100,
    engine: "token-window"
  };
}
