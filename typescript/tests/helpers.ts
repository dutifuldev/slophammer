import { readFile } from "node:fs/promises";
import path from "node:path";

import type { Finding, Report } from "../src/rules/types.js";

export const repoRoot = path.resolve(import.meta.dirname, "../..");

export function fixturePath(name: string): string {
  return path.join(repoRoot, "fixtures", "repos", name);
}

export async function expectedReport(name: string): Promise<Report> {
  const content = await readFile(
    path.join(repoRoot, "fixtures", "expected", `${name}.json`),
    "utf8"
  );
  return parseReport(content);
}

// bindingScriptWorkflow returns a CI workflow with binding triggers that
// invokes the conventional package scripts, so package.json script evidence
// stays reachable under the binding CI evidence rules.
export function bindingScriptWorkflow(): string {
  return [
    "name: ci",
    "on: [push]",
    "jobs:",
    "  check:",
    "    steps:",
    "      - run: npm run format",
    "      - run: npm run lint",
    "      - run: npm run typecheck",
    "      - run: npm test",
    "      - run: npm run test:unit",
    "      - run: npm run coverage",
    "      - run: npm run dry",
    "      - run: npm run mutate",
    ""
  ].join("\n");
}

export function parseReport(content: string): Report {
  const parsed: unknown = JSON.parse(content);
  const root = asRecord(parsed);
  return {
    ok: root["ok"] === true,
    findings: asFindings(root["findings"])
  };
}

export function normalizeReport(report: Report): Report {
  return {
    ok: report.ok,
    findings: [...report.findings].sort((left, right) => {
      const byRule = left.rule_id.localeCompare(right.rule_id);
      return byRule === 0 ? left.path.localeCompare(right.path) : byRule;
    })
  };
}

function asFindings(value: unknown): readonly Finding[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.map(asFinding);
}

function asFinding(value: unknown): Finding {
  const root = asRecord(value);
  return {
    rule_id: asString(root["rule_id"]),
    severity: root["severity"] === "warn" ? "warn" : "error",
    path: asString(root["path"]),
    message: asString(root["message"])
  };
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}
