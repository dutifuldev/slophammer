import { commandFiles, type Snapshot } from "../repo/repo.js";
import { asRecord, commandSegments, normalizeCommandContent } from "./command-content.js";
import { expandedPackageScriptSegments, packageScripts } from "./package-scripts.js";
import { ignoredProjectDataPath } from "./source-paths.js";

// Only an executing `stryker run` counts: init and help invocations never
// execute a mutant, and a dry run validates configuration without being
// able to fail on a survivor. Stryker itself exits zero on surviving
// mutants unless thresholds.break is configured, so the run only gates
// beside a positive breaking threshold.
export function hasTypeScriptMutationCommand(snapshot: Snapshot): boolean {
  if (!hasStrykerBreakThreshold(snapshot)) {
    return false;
  }
  const scripts = packageScripts(snapshot);
  return commandFiles(snapshot).some((file) =>
    commandSegments(file.content)
      .flatMap((segment) => expandedPackageScriptSegments(segment, scripts))
      .map((segment) => normalizeCommandContent(segment))
      .some((segment) => /\bstryker run\b/u.test(segment) && !/--dry-?run-?only\b/u.test(segment))
  );
}

function hasStrykerBreakThreshold(snapshot: Snapshot): boolean {
  for (const file of snapshot.files.values()) {
    if (ignoredProjectDataPath(file.path)) {
      continue;
    }
    const name = file.path.split("/").at(-1) ?? "";
    if (!/^stryker\.(?:conf|config)\./u.test(name)) {
      continue;
    }
    if (strykerConfigDeclaresBreak(file.content, name)) {
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
