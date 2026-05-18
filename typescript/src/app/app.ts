import path from "node:path";

import { loadConfig, ruleSeverity, type Config } from "../config/config.js";
import { checkDry } from "../dry/dry.js";
import type { DryOptions } from "../dry/types.js";
import type { Snapshot } from "../repo/repo.js";
import { newReport, writeJSON, writeSARIF, writeText } from "../report/report.js";
import { scanRepo } from "../scan/scan.js";
import { defaultDefinitions } from "../rules/definitions.js";
import { ruleIDs } from "../rules/definitions.js";
import { explain as explainRule, runRules } from "../rules/rules.js";
import type { Finding } from "../rules/types.js";
import { executeTypeScriptChecks, execRunner, type Runner } from "../toolchecks/toolchecks.js";

export const exitOK = 0;
export const exitFindings = 1;
export const exitError = 2;

export type CheckOptions = {
  readonly root: string;
  readonly format: "text" | "json" | "sarif";
  readonly execute: boolean;
  readonly onlyRuleIDs?: readonly string[];
};

export async function check(
  options: CheckOptions,
  runner: Runner = execRunner
): Promise<{ readonly code: number; readonly stdout: string; readonly stderr: string }> {
  try {
    const snapshot = await scanRepo(options.root);
    const cfg = loadConfig(snapshot);
    const onlyRuleIDs = options.onlyRuleIDs ?? [];
    validateOnlyRuleIDs(onlyRuleIDs);
    const base = runRules(snapshot, cfg, { onlyRuleIDs });
    const findings =
      options.execute && shouldExecuteTypeScriptChecks(onlyRuleIDs)
        ? [
            ...base.findings,
            ...filterFindings(
              applySeverityOverrides(await executeChecks(snapshot, runner, onlyRuleIDs), cfg),
              onlyRuleIDs
            )
          ]
        : base.findings;
    const report = newReport(findings);
    return {
      code: report.ok ? exitOK : exitFindings,
      stdout: render(options.format, report),
      stderr: ""
    };
  } catch (error) {
    return { code: exitError, stdout: "", stderr: `check failed: ${errorMessage(error)}\n` };
  }
}

export async function boundaries(
  options: Omit<CheckOptions, "onlyRuleIDs">
): Promise<{ readonly code: number; readonly stdout: string; readonly stderr: string }> {
  return await check({ ...options, onlyRuleIDs: [ruleIDs.tsDependencyBoundariesRequired] });
}

function validateOnlyRuleIDs(onlyRuleIDs: readonly string[]): void {
  const known = new Set(defaultDefinitions.map((definition) => definition.id));
  const unknown = onlyRuleIDs.filter((ruleID) => !known.has(ruleID));
  if (unknown.length > 0) {
    throw new Error(`unknown rule: ${unknown.join(", ")}`);
  }
}

function filterFindings(
  findings: readonly Finding[],
  onlyRuleIDs: readonly string[]
): readonly Finding[] {
  if (onlyRuleIDs.length === 0) {
    return findings;
  }
  const wanted = new Set(onlyRuleIDs);
  return findings.filter((finding) => wanted.has(finding.rule_id));
}

const executableTypeScriptRuleIDs = new Set<string>([
  ruleIDs.tsFormatRequired,
  ruleIDs.tsLintRequired,
  ruleIDs.tsTypecheckRequired,
  ruleIDs.tsTestRequired,
  ruleIDs.tsCoverageRequired,
  ruleIDs.tsComplexityRequired,
  ruleIDs.tsDryRequired,
  ruleIDs.tsMutationRequired
]);

function shouldExecuteTypeScriptChecks(onlyRuleIDs: readonly string[]): boolean {
  return (
    onlyRuleIDs.length === 0 ||
    onlyRuleIDs.some((ruleID) => executableTypeScriptRuleIDs.has(ruleID))
  );
}

function applySeverityOverrides(findings: readonly Finding[], cfg: Config): readonly Finding[] {
  return findings.map((finding) => ({
    ...finding,
    severity: ruleSeverity(cfg, finding.rule_id, finding.severity)
  }));
}

async function executeChecks(
  snapshot: Snapshot,
  runner: Runner,
  onlyRuleIDs: readonly string[]
): Promise<readonly Finding[]> {
  const findings: Finding[] = [];
  for (const packageRoot of typeScriptPackageRoots(snapshot)) {
    const root = path.join(snapshot.root, packageRoot.split("/").join(path.sep));
    const packageFindings = await executeTypeScriptChecks(root, runner, snapshot.root, onlyRuleIDs);
    findings.push(...packageFindings.map((finding) => prefixPackageFinding(packageRoot, finding)));
  }
  return findings;
}

function typeScriptPackageRoots(snapshot: Snapshot): readonly string[] {
  return [...snapshot.files.keys()]
    .filter((filePath) => filePath.endsWith("package.json") && !ignoredProjectDataPath(filePath))
    .map((filePath) => path.posix.dirname(filePath))
    .map((dir) => (dir === "." ? "" : dir))
    .filter((packageRoot) => packageRootHasTypeScriptEvidence(snapshot, packageRoot))
    .sort((left, right) => left.localeCompare(right));
}

function packageRootHasTypeScriptEvidence(snapshot: Snapshot, packageRoot: string): boolean {
  return [...snapshot.files.values()].some((file) => {
    if (
      packageRootForFile(snapshot, file.path) !== packageRoot ||
      ignoredProjectDataPath(file.path)
    ) {
      return false;
    }
    return (
      file.path.endsWith("tsconfig.json") ||
      typeScriptProductionSourcePath(file.path) ||
      (file.path.endsWith("package.json") && packageContentHasTypeScriptSignal(file.content))
    );
  });
}

function packageRootForFile(snapshot: Snapshot, filePath: string): string | undefined {
  for (let dir: string | undefined = path.posix.dirname(filePath); dir !== undefined; ) {
    const packagePath = dir === "." ? "package.json" : `${dir}/package.json`;
    if (snapshot.files.has(packagePath)) {
      return dir === "." ? "" : dir;
    }
    dir = parentDir(dir);
  }
  return undefined;
}

function parentDir(dir: string): string | undefined {
  return dir === "." ? undefined : path.posix.dirname(dir);
}

function packageContentHasTypeScriptSignal(content: string): boolean {
  try {
    const root = asRecord(JSON.parse(content) as unknown);
    const scripts = asRecord(root["scripts"]);
    const scriptContent = Object.entries(scripts)
      .filter((entry): entry is [string, string] => typeof entry[1] === "string")
      .map(([name, value]) => `${name}: ${value}`)
      .join("\n")
      .toLowerCase();
    return (
      scriptContent.includes("typecheck") ||
      /\b(?:tsc|tsgo)\b/u.test(scriptContent) ||
      packageDependenciesHaveTypeScriptSignal(root)
    );
  } catch {
    return false;
  }
}

function packageDependenciesHaveTypeScriptSignal(root: Readonly<Record<string, unknown>>): boolean {
  return [
    ...Object.keys(asRecord(root["dependencies"])),
    ...Object.keys(asRecord(root["devDependencies"])),
    ...Object.keys(asRecord(root["peerDependencies"])),
    ...Object.keys(asRecord(root["optionalDependencies"]))
  ].some((name) => typeScriptDependencyName(name));
}

function typeScriptDependencyName(name: string): boolean {
  return (
    name === "typescript" ||
    name === "@typescript/native-preview" ||
    name === "ts-node" ||
    name === "tsx" ||
    name === "ts-jest" ||
    name === "typescript-eslint" ||
    name.startsWith("@typescript-eslint/")
  );
}

function prefixPackageFinding(packageRoot: string, finding: Finding): Finding {
  return {
    ...finding,
    path: packageRoot === "" ? finding.path : path.posix.join(packageRoot, finding.path)
  };
}

function ignoredProjectDataPath(filePath: string): boolean {
  return filePath.startsWith("fixtures/") || filePath.startsWith("templates/");
}

function typeScriptProductionSourcePath(filePath: string): boolean {
  return (
    typeScriptSourceExtension(filePath) &&
    !typeScriptDeclarationPath(filePath) &&
    !testSourcePath(filePath) &&
    !typeScriptToolingConfigPath(filePath)
  );
}

function typeScriptSourceExtension(filePath: string): boolean {
  return (
    filePath.endsWith(".ts") ||
    filePath.endsWith(".tsx") ||
    filePath.endsWith(".mts") ||
    filePath.endsWith(".cts")
  );
}

function typeScriptDeclarationPath(filePath: string): boolean {
  return filePath.endsWith(".d.ts") || filePath.endsWith(".d.mts") || filePath.endsWith(".d.cts");
}

function testSourcePath(filePath: string): boolean {
  const baseName = path.posix.basename(filePath);
  return (
    filePath.startsWith("test/") ||
    filePath.startsWith("tests/") ||
    filePath.includes("/test/") ||
    filePath.includes("/tests/") ||
    baseName.includes(".test.") ||
    baseName.includes(".spec.")
  );
}

function typeScriptToolingConfigPath(filePath: string): boolean {
  return path.posix.basename(filePath).includes(".config.");
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}

export function explain(ruleID: string): {
  readonly code: number;
  readonly stdout: string;
  readonly stderr: string;
} {
  const text = explainRule(ruleID);
  if (text === undefined) {
    return { code: exitError, stdout: "", stderr: `unknown rule: ${ruleID}\n` };
  }
  return { code: exitOK, stdout: `${text}\n`, stderr: "" };
}

export function rules(): {
  readonly code: number;
  readonly stdout: string;
  readonly stderr: string;
} {
  return ruleCatalog({ format: "text" });
}

export type RuleCatalogOptions = {
  readonly format: "text" | "json";
};

export function ruleCatalog(options: RuleCatalogOptions): {
  readonly code: number;
  readonly stdout: string;
  readonly stderr: string;
} {
  if (options.format === "json") {
    return {
      code: exitOK,
      stdout: `${JSON.stringify(defaultDefinitions, null, 2)}\n`,
      stderr: ""
    };
  }
  return {
    code: exitOK,
    stdout: `${ruleCatalogText()}\n`,
    stderr: ""
  };
}

function ruleCatalogText(): string {
  const header = ["RULE ID", "CATEGORY", "SEVERITY", "STATUS", "TOOL"];
  const rows = [
    header,
    ...defaultDefinitions.map((definition) => [
      definition.id,
      definition.category,
      definition.severity,
      definition.status,
      definition.tool ?? ""
    ])
  ];
  const widths = header.map((_, column) =>
    Math.max(...rows.map((row) => row[column]?.length ?? 0))
  );
  return rows
    .map((row) =>
      row
        .map((cell, column) => cell.padEnd(widths[column] ?? 0))
        .join("  ")
        .trimEnd()
    )
    .join("\n");
}

export async function typescriptDry(
  options: DryOptions
): Promise<{ readonly code: number; readonly stdout: string; readonly stderr: string }> {
  try {
    const result = await checkDry(await applyDryConfig(options));
    return { code: result.code, stdout: result.output, stderr: "" };
  } catch (error) {
    return { code: exitError, stdout: "", stderr: `dry check failed: ${errorMessage(error)}\n` };
  }
}

async function applyDryConfig(options: DryOptions): Promise<DryOptions> {
  const snapshot = await scanRepo(options.root === "" ? "." : options.root);
  const cfg = loadConfig(snapshot).typescript.dry;
  return {
    ...options,
    root: snapshot.root,
    paths: cfg.paths.length > 0 ? cfg.paths : options.paths,
    exclude: cfg.exclude.length > 0 ? cfg.exclude : options.exclude,
    ...dryBudgetOptions(options, cfg),
    ...dryCopiedBlockOptions(options, cfg)
  };
}

type TypeScriptDryConfig = ReturnType<typeof loadConfig>["typescript"]["dry"];

function dryBudgetOptions(options: DryOptions, cfg: TypeScriptDryConfig): Partial<DryOptions> {
  return {
    maxFindings: options.maxFindingsSet ? options.maxFindings : cfg.maxFindings,
    maxFindingsSet: options.maxFindingsSet || cfg.maxFindingsSet
  };
}

function dryCopiedBlockOptions(options: DryOptions, cfg: TypeScriptDryConfig): Partial<DryOptions> {
  return {
    copiedBlockEnabled: cfg.copiedBlocks.enabledSet
      ? cfg.copiedBlocks.enabled
      : options.copiedBlockEnabled,
    copiedBlockSet: options.copiedBlockSet || cfg.copiedBlocks.enabledSet,
    copiedBlockTokens:
      cfg.copiedBlocks.minTokens > 0 ? cfg.copiedBlocks.minTokens : options.copiedBlockTokens
  };
}

function render(format: CheckOptions["format"], report: Parameters<typeof writeJSON>[0]): string {
  switch (format) {
    case "text":
      return writeText(report);
    case "json":
      return writeJSON(report);
    case "sarif":
      return writeSARIF(report);
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
