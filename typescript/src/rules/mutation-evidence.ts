import { commandFiles, type Snapshot } from "../repo/repo.js";
import { asRecord, commandSegments, normalizeCommandContent } from "./command-content.js";
import { expandedPackageScriptSegments, packageScripts } from "./package-scripts.js";
import { ignoredProjectDataPath } from "./source-paths.js";

// Only an executing `stryker run` counts: init and help invocations never
// execute a mutant, and a dry run validates configuration without being
// able to fail on a survivor. Stryker itself exits zero on surviving
// mutants unless thresholds.break is configured, so the run only gates
// when the config it actually loads declares a positive breaking
// threshold.
export function hasTypeScriptMutationCommand(snapshot: Snapshot): boolean {
  const scripts = packageScripts(snapshot);
  return commandFiles(snapshot).some((file) =>
    commandSegments(file.content)
      .flatMap((segment) => expandedPackageScriptSegments(segment, scripts))
      .map((segment) => normalizeCommandContent(segment))
      .some(
        (segment) =>
          /\bstryker run\b/u.test(segment) &&
          !/--dry-?run-?only\b/u.test(segment) &&
          runDeclaresBreakThreshold(snapshot, segment)
      )
  );
}

// A run with a positional config file loads that file; otherwise Stryker
// loads its config from the directory it runs in, so only a project-root
// config gates the run - a copy under docs/ is never loaded.
function runDeclaresBreakThreshold(snapshot: Snapshot, segment: string): boolean {
  const explicit = /\bstryker run\s+([^\s-]\S*)/u.exec(segment)?.[1];
  if (explicit !== undefined) {
    const file = configFileNamed(snapshot, explicit);
    return file !== undefined && strykerConfigDeclaresBreak(file.content, file.path);
  }
  return hasRootStrykerBreakThreshold(snapshot);
}

// Segments are lowercased during normalization, so the lookup compares
// lowercased paths.
function configFileNamed(
  snapshot: Snapshot,
  configPath: string
): { readonly path: string; readonly content: string } | undefined {
  for (const file of snapshot.files.values()) {
    if (file.path.toLowerCase() === configPath && !ignoredProjectDataPath(file.path)) {
      return file;
    }
  }
  return undefined;
}

function hasRootStrykerBreakThreshold(snapshot: Snapshot): boolean {
  for (const file of snapshot.files.values()) {
    if (file.path.includes("/") || ignoredProjectDataPath(file.path)) {
      continue;
    }
    if (!/^stryker\.(?:conf|config)\./u.test(file.path)) {
      continue;
    }
    if (strykerConfigDeclaresBreak(file.content, file.path)) {
      return true;
    }
  }
  return false;
}

// JSON configs are parsed so malformed files never count; JS configs are
// matched after stripping comments so a commented-out threshold is not
// credited. A config that turns on dryRunOnly never executes a mutant,
// so its threshold does not gate anything.
function strykerConfigDeclaresBreak(content: string, name: string): boolean {
  if (name.endsWith(".json")) {
    const config = asRecord(parsedJSON(content));
    if (config["dryRunOnly"] === true) {
      return false;
    }
    const breakValue = asRecord(config["thresholds"])["break"];
    return typeof breakValue === "number" && breakValue > 0;
  }
  const stripped = content.replace(/\/\*[\s\S]*?\*\//gu, "").replace(/\/\/[^\n]*/gu, "");
  if (/\bdryRunOnly["']?\s*:\s*true\b/u.test(stripped)) {
    return false;
  }
  const match = /\bbreak["']?\s*:\s*(\d+(?:\.\d+)?)/u.exec(stripped);
  return match?.[1] !== undefined && Number(match[1]) > 0;
}

function parsedJSON(content: string): unknown {
  try {
    return JSON.parse(content);
  } catch {
    return undefined;
  }
}
