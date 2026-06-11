import { access, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

import type { Finding, Report } from "../rules/types.js";

export const baselineFileName = "slophammer-baseline.json";

export type BaselineMode = "off" | "check" | "write";

type BaselineEntry = {
  readonly rule_id: string;
  readonly path: string;
};

// applyBaselineCheck applies a checked-in baseline to a report: matched
// findings are marked baselined and stop affecting ok; stale entries are an
// error so the ratchet can only shrink. Matching is on rule_id plus path,
// never message.
export async function applyBaselineCheck(root: string, report: Report): Promise<Report> {
  const baseline = await readBaseline(root);
  const keys = new Set(baseline.map(entryKey));
  const findings = report.findings.map((finding) =>
    keys.has(entryKey(entryOf(finding))) ? { ...finding, baselined: true as const } : finding
  );
  const matched = new Set(
    findings
      .filter((finding) => finding.baselined === true)
      .map((finding) => entryKey(entryOf(finding)))
  );
  const stale = baseline.filter((entry) => !matched.has(entryKey(entry)));
  if (stale.length > 0) {
    throw new Error(`baseline contains resolved findings; rewrite it: ${joined(stale)}`);
  }
  return {
    ...report,
    ok: findings.every((finding) => finding.baselined === true),
    findings
  };
}

// writeBaseline records current findings as the baseline. It refuses to
// write a superset of an existing baseline and reports the added and
// removed entries, so debt is recorded once, reviewed, and only ever
// reduced.
export async function writeBaseline(root: string, report: Report): Promise<string> {
  const current = sortedUniqueEntries(report.findings.map(entryOf));
  const previous = await previousBaseline(root);
  const previousKeys = new Set(previous.map(entryKey));
  const currentKeys = new Set(current.map(entryKey));
  const added = current.filter((entry) => !previousKeys.has(entryKey(entry)));
  const removed = previous.filter((entry) => !currentKeys.has(entryKey(entry)));
  if (previous.length > 0 && added.length > 0 && removed.length === 0) {
    throw new Error(
      `baseline write would grow the baseline; fix the new findings instead: ${joined(added)}`
    );
  }
  const serialized = JSON.stringify({ version: 1, findings: current }, null, 2);
  await writeFile(path.join(root, baselineFileName), `${serialized}\n`);
  return writeSummary(current.length, added, removed);
}

export function debtLine(report: Report): string {
  const baselined = report.findings.filter((finding) => finding.baselined === true).length;
  const fresh = report.findings.length - baselined;
  return `${String(baselined)} findings baselined; ${String(fresh)} new\n`;
}

// Only a missing baseline file counts as the initial-write case; a present
// but malformed baseline propagates its parse error instead of being
// silently replaced.
async function previousBaseline(root: string): Promise<readonly BaselineEntry[]> {
  if (!(await baselinePresent(root))) {
    return [];
  }
  return readBaseline(root);
}

async function baselinePresent(root: string): Promise<boolean> {
  try {
    await access(path.join(root, baselineFileName));
    return true;
  } catch {
    return false;
  }
}

async function readBaseline(root: string): Promise<readonly BaselineEntry[]> {
  const content = await readBaselineContent(root);
  const parsed = parseBaselineRoot(content);
  if (parsed["version"] !== 1) {
    throw new Error("baseline version must be 1");
  }
  return parseBaselineEntries(parsed["findings"]);
}

async function readBaselineContent(root: string): Promise<string> {
  try {
    return await readFile(path.join(root, baselineFileName), "utf8");
  } catch {
    throw new Error(`baseline file ${baselineFileName} is missing`);
  }
}

function parseBaselineRoot(content: string): Readonly<Record<string, unknown>> {
  let parsed: unknown;
  try {
    parsed = JSON.parse(content);
  } catch (error) {
    throw new Error(
      `baseline parse failed: ${error instanceof Error ? error.message : String(error)}`
    );
  }
  const root = strictRecord(parsed, ["version", "findings"]);
  if (root === undefined) {
    throw new Error("baseline parse failed: baseline must be an object with version and findings");
  }
  return root;
}

function parseBaselineEntries(value: unknown): readonly BaselineEntry[] {
  if (!Array.isArray(value)) {
    throw new Error("baseline parse failed: findings must be a list");
  }
  return value.map((item) => {
    const entry = strictRecord(item, ["rule_id", "path"]);
    const ruleID = entry?.["rule_id"];
    const entryPath = entry?.["path"];
    if (entry === undefined || typeof ruleID !== "string" || typeof entryPath !== "string") {
      throw new Error("baseline parse failed: findings entries need rule_id and path strings");
    }
    return { rule_id: ruleID, path: entryPath };
  });
}

// strictRecord mirrors the deny-unknown-fields validation of slophammer.yml:
// unknown baseline keys are an error rather than silently ignored.
function strictRecord(
  value: unknown,
  allowed: readonly string[]
): Readonly<Record<string, unknown>> | undefined {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return undefined;
  }
  const record = value as Readonly<Record<string, unknown>>;
  return Object.keys(record).every((key) => allowed.includes(key)) ? record : undefined;
}

function entryOf(finding: Finding): BaselineEntry {
  return { rule_id: finding.rule_id, path: finding.path };
}

function entryKey(entry: BaselineEntry): string {
  return `${entry.rule_id}\u0000${entry.path}`;
}

function sortedUniqueEntries(entries: readonly BaselineEntry[]): readonly BaselineEntry[] {
  const byKey = new Map(entries.map((entry) => [entryKey(entry), entry]));
  return [...byKey.values()].sort(
    (left, right) =>
      left.rule_id.localeCompare(right.rule_id) || left.path.localeCompare(right.path)
  );
}

function joined(entries: readonly BaselineEntry[]): string {
  return sortedUniqueEntries(entries)
    .map((entry) => `${entry.rule_id} at ${entry.path}`)
    .join(", ");
}

function writeSummary(
  total: number,
  added: readonly BaselineEntry[],
  removed: readonly BaselineEntry[]
): string {
  const lines = [
    `baseline written: ${String(total)} finding(s)`,
    ...added.map((entry) => `added: ${entry.rule_id} at ${entry.path}`),
    ...removed.map((entry) => `removed: ${entry.rule_id} at ${entry.path}`)
  ];
  return `${lines.join("\n")}\n`;
}
