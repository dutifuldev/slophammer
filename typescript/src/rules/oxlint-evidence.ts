import { commandFiles, filesNamed, type Snapshot } from "../repo/repo.js";
import { expandedPackageScriptSegments, packageScripts } from "./package-scripts.js";
import ts from "typescript";

export function hasOxlintRule(snapshot: Snapshot, ruleName: string): boolean {
  const warningsFail = hasOxlintDenyWarnings(snapshot);
  return oxlintRuleValues(snapshot, ruleName).some((value) =>
    enforcingStructuredRuleValue(value, warningsFail)
  );
}

export function hasTypeAwareOxlintRule(snapshot: Snapshot, ruleName: string): boolean {
  if (!hasTypeAwareOxlintCommand(snapshot)) {
    return false;
  }
  const warningsFail = hasOxlintDenyWarnings(snapshot, { requireTypeAware: true });
  return oxlintRuleValues(snapshot, ruleName).some((value) =>
    enforcingStructuredRuleValue(value, warningsFail)
  );
}

export function hasOxlintComplexityRule(snapshot: Snapshot, maximum: number): boolean {
  const warningsFail = hasOxlintDenyWarnings(snapshot);
  return oxlintRuleValues(snapshot, "eslint/complexity").some(
    (value) =>
      enforcingStructuredRuleValue(value, warningsFail) && structuredComplexityLimit(value, maximum)
  );
}

function oxlintRuleValues(snapshot: Snapshot, ruleName: string): readonly unknown[] {
  const values = oxlintConfigRoots(snapshot).flatMap((root) =>
    collectStructuredRuleValues(root, ruleName)
  );
  const effective = values[values.length - 1];
  return effective === undefined ? [] : [effective];
}

function oxlintConfigRoots(snapshot: Snapshot): readonly Readonly<Record<string, unknown>>[] {
  return orderedOxlintConfigFiles(
    productionFilesNamed(
      snapshot,
      ".oxlintrc.json",
      ".oxlintrc.jsonc",
      "oxlint.config.json",
      "oxlint.config.jsonc"
    )
  ).flatMap((file) => {
    try {
      return [parseJSONConfig(file.content)];
    } catch {
      return [];
    }
  });
}

function orderedOxlintConfigFiles(
  files: readonly {
    readonly path: string;
    readonly content: string;
  }[]
): readonly {
  readonly path: string;
  readonly content: string;
}[] {
  return [...files].sort((left, right) => {
    const byOrigin = configOriginOrder(left.path) - configOriginOrder(right.path);
    return byOrigin === 0 ? left.path.localeCompare(right.path) : byOrigin;
  });
}

function configOriginOrder(filePath: string): number {
  return filePath.startsWith("__repo__/") ? 0 : 1;
}

function productionFilesNamed(
  snapshot: Snapshot,
  ...names: readonly string[]
): readonly {
  readonly path: string;
  readonly content: string;
}[] {
  return filesNamed(snapshot, ...names).filter((file) => !ignoredProjectDataPath(file.path));
}

function ignoredProjectDataPath(filePath: string): boolean {
  return filePath.startsWith("fixtures/") || filePath.startsWith("templates/");
}

function parseJSONConfig(content: string): Readonly<Record<string, unknown>> {
  try {
    return asRecord(JSON.parse(content) as unknown);
  } catch {
    const parsed = ts.parseConfigFileTextToJson("oxlint.jsonc", content);
    if (parsed.error !== undefined) {
      return asRecord(JSON.parse(stripJSONComments(content)) as unknown);
    }
    return asRecord(parsed.config as unknown);
  }
}

function collectStructuredRuleValues(
  root: Readonly<Record<string, unknown>>,
  ruleName: string
): readonly unknown[] {
  const values: unknown[] = [];
  collectRuleFromRecord(root);
  for (const override of arrayValues(root["overrides"]).map(asRecord)) {
    if (overrideAppliesToProductionTypeScript(override["files"])) {
      collectRuleFromRecord(override);
    }
  }
  return values;

  function collectRuleFromRecord(record: Readonly<Record<string, unknown>>): void {
    const rules = asRecord(record["rules"]);
    if (Object.hasOwn(rules, ruleName)) {
      values.push(rules[ruleName]);
    }
  }
}

function overrideAppliesToProductionTypeScript(files: unknown): boolean {
  const patterns = stringPatterns(files);
  return patterns.length === 0 || patterns.some((pattern) => !testOnlyPattern(pattern));
}

function stringPatterns(value: unknown): readonly string[] {
  if (typeof value === "string") {
    return [value];
  }
  return arrayValues(value).filter((item): item is string => typeof item === "string");
}

function testOnlyPattern(pattern: string): boolean {
  const normalized = pattern.toLowerCase();
  return (
    normalized.includes("test") || normalized.includes("spec") || normalized.includes("integration")
  );
}

function enforcingStructuredRuleValue(value: unknown, warningsFail: boolean): boolean {
  const severity = Array.isArray(value) ? (value as readonly unknown[])[0] : value;
  return (
    severity === "error" ||
    severity === 2 ||
    (warningsFail && (severity === "warn" || severity === 1))
  );
}

function structuredComplexityLimit(value: unknown, maximum: number): boolean {
  if (!Array.isArray(value)) {
    return false;
  }
  return value
    .slice(1)
    .map(complexityLimitValue)
    .some((item) => item !== undefined && Number.isInteger(item) && item > 0 && item <= maximum);
}

function complexityLimitValue(value: unknown): number | undefined {
  if (typeof value === "number") {
    return value;
  }
  const max = asRecord(value)["max"];
  return typeof max === "number" ? max : undefined;
}

function stripJSONComments(content: string): string {
  return content
    .replace(/\/\*[\s\S]*?\*\//gu, "")
    .split("\n")
    .map((line) => line.replace(/(^|[^:])\/\/.*$/u, "$1"))
    .join("\n");
}

function hasTypeAwareOxlintCommand(snapshot: Snapshot): boolean {
  const scripts = packageScripts(snapshot);
  return commandFiles(snapshot).some((file) =>
    commandSegments(file.content)
      .flatMap((segment) => expandedPackageScriptSegments(segment, scripts))
      .some((segment) => /\boxlint\b/u.test(segment) && /--type-aware\b/u.test(segment))
  );
}

function hasOxlintDenyWarnings(
  snapshot: Snapshot,
  options: { readonly requireTypeAware?: boolean } = {}
): boolean {
  const scripts = packageScripts(snapshot);
  return commandFiles(snapshot).some((file) =>
    commandSegments(file.content)
      .flatMap((segment) => expandedPackageScriptSegments(segment, scripts))
      .some(
        (segment) =>
          /\boxlint\b/u.test(segment) &&
          /--deny-warnings\b/u.test(segment) &&
          (options.requireTypeAware !== true || /--type-aware\b/u.test(segment))
      )
  );
}

function commandSegments(content: string): readonly string[] {
  return content
    .replaceAll("\\\n", " ")
    .split(/\n|&&|;/u)
    .map((segment) => normalizeCommandContent(segment).trim())
    .filter((segment) => !segment.includes("||"))
    .filter((segment) => segment.length > 0);
}

function normalizeCommandContent(content: string): string {
  return content.replaceAll("\\\n", " ").replace(/\s+/g, " ").toLowerCase();
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}

function arrayValues(value: unknown): readonly unknown[] {
  return Array.isArray(value) ? value : [];
}
