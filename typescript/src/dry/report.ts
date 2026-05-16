import type { DryFinding, DryGroup, DryReport, SourceRange } from "./types.js";

export function groupFindings(findings: readonly DryFinding[]): readonly DryGroup[] {
  const groups: DryGroup[] = [];
  findings.forEach((finding, index) => {
    const existing = groups.findIndex((group) => overlapsGroup(finding, group));
    if (existing >= 0) {
      const group = groups[existing];
      if (group !== undefined) {
        groups[existing] = appendToGroup(group, finding, index);
      }
      return;
    }
    groups.push({
      id: `dry-${String(groups.length + 1)}`,
      findings: [index],
      kinds: [finding.kind],
      left: finding.left,
      right: finding.right
    });
  });
  return groups;
}

export function formatText(report: DryReport): string {
  const lines = [
    `DRY findings: ${String(report.findings.length)}`,
    `Groups: ${String(report.groups.length)}`,
    `Copied block findings: ${String(report.findings.length)}`,
    ""
  ];
  for (const group of report.groups) {
    lines.push(`DRY ${group.id} [${group.kinds.join(", ")}]`);
    lines.push(`  ${formatRange(group.left)}`);
    lines.push(`  ${formatRange(group.right)}`);
  }
  return `${lines.join("\n")}\n`;
}

export function writeJSON(report: DryReport): string {
  return `${JSON.stringify(report, null, 2)}\n`;
}

export function rangesOverlap(left: SourceRange, right: SourceRange): boolean {
  return (
    left.path === right.path &&
    Math.max(left.start_line, right.start_line) <= Math.min(left.end_line, right.end_line)
  );
}

function appendToGroup(group: DryGroup, finding: DryFinding, index: number): DryGroup {
  return {
    ...group,
    findings: [...group.findings, index],
    kinds: group.kinds.includes(finding.kind) ? group.kinds : [...group.kinds, finding.kind].sort(),
    left: mergeRange(group.left, finding.left),
    right: mergeRange(group.right, finding.right)
  };
}

function overlapsGroup(finding: DryFinding, group: DryGroup): boolean {
  return (
    (rangesOverlap(finding.left, group.left) && rangesOverlap(finding.right, group.right)) ||
    (rangesOverlap(finding.left, group.right) && rangesOverlap(finding.right, group.left))
  );
}

function mergeRange(left: SourceRange, right: SourceRange): SourceRange {
  if (left.path !== right.path) {
    return left;
  }
  return {
    path: left.path,
    start_line: Math.min(left.start_line, right.start_line),
    end_line: Math.max(left.end_line, right.end_line)
  };
}

function formatRange(range: SourceRange): string {
  return `${range.path}:${String(range.start_line)}-${String(range.end_line)}`;
}
