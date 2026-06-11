import path from "node:path";

import type { Config } from "../config/config.js";
import type { Snapshot } from "../repo/repo.js";
import { typeScriptSourcePath } from "./source-paths.js";
import type { Definition, Finding, ScopeCoverage } from "./types.js";

// Configured scope must account for every production TypeScript file: each
// one is either inside a configured paths scope or covered by an exclude
// (conventional or reasoned). Anything else is a finding, so narrowing
// scope cannot hide code from checking.
export function scopeFindings(
  definition: Definition,
  snapshot: Snapshot,
  cfg: Config
): readonly Finding[] {
  const scopes = configuredScopePaths(cfg);
  if (scopes.length === 0) {
    return [];
  }
  const uncovered = uncoveredProductionDirs(snapshot, cfg, scopes);
  if (uncovered.length === 0) {
    return [];
  }
  return [
    {
      rule_id: definition.id,
      severity: definition.severity,
      path: definition.path,
      message: `${definition.message}: ${uncovered.join(", ")}`
    }
  ];
}

// scopeCounts reports configured-scope coverage over production TypeScript
// files for the report, or undefined when no scope is configured.
export function scopeCounts(snapshot: Snapshot, cfg: Config): ScopeCoverage | undefined {
  const scopes = configuredScopePaths(cfg);
  if (scopes.length === 0) {
    return undefined;
  }
  const production = productionTypeScriptFiles(snapshot);
  return {
    scanned: production.filter((filePath) => inScope(filePath, scopes)).length,
    production_files: production.length
  };
}

function configuredScopePaths(cfg: Config): readonly string[] {
  return [...cfg.typescript.dry.paths, ...cfg.typescript.coverage.paths];
}

function uncoveredProductionDirs(
  snapshot: Snapshot,
  cfg: Config,
  scopes: readonly string[]
): readonly string[] {
  const excludes = [...cfg.typescript.dry.exclude, ...cfg.typescript.coverage.exclude];
  const dirs = productionTypeScriptFiles(snapshot)
    .filter((filePath) => !inScope(filePath, scopes))
    .filter((filePath) => !excludes.some((pattern) => excludeMatch(filePath, pattern)))
    .map(parentDir);
  return [...new Set(dirs)].sort((left, right) => left.localeCompare(right));
}

function productionTypeScriptFiles(snapshot: Snapshot): readonly string[] {
  return [...snapshot.files.keys()].filter(
    (filePath) => typeScriptSourcePath(filePath) && !conventionalPath(filePath)
  );
}

// Path-level form of the conventional non-production list in
// specs/CONFIG.md; test, fixture, and template paths are already covered by
// the production source classification.
const conventionalDirs = new Set([
  "testdata",
  "dist",
  "build",
  "coverage",
  "target",
  "node_modules",
  "vendor",
  "scripts"
]);

function conventionalPath(filePath: string): boolean {
  if (filePath.includes("generated") || path.posix.basename(filePath).includes("_test.")) {
    return true;
  }
  return filePath.split("/").some((segment) => conventionalDirs.has(segment));
}

function parentDir(filePath: string): string {
  const dir = path.posix.dirname(filePath);
  return dir === "" ? "." : dir;
}

function inScope(filePath: string, scopes: readonly string[]): boolean {
  return scopes.some((scope) => {
    const normalized = scope.replace(/\/+$/u, "");
    return normalized === "." || filePath === normalized || filePath.startsWith(`${normalized}/`);
  });
}

function excludeMatch(filePath: string, pattern: string): boolean {
  return globRegExp(pattern).test(filePath);
}

// globRegExp supports the minimatch subset used by excludes: `**` crosses
// directories, `*` stays within one segment, and a literal pattern covers
// the path itself or anything below it.
function globRegExp(pattern: string): RegExp {
  const source = pattern
    .split("**")
    .map((part) => part.replace(/[.+^${}()|[\]\\]/gu, "\\$&").replaceAll("*", "[^/]*"))
    .join("\u0000")
    .replaceAll("\u0000/", "(?:.*/)?")
    .replaceAll("\u0000", ".*");
  const literalSuffix = pattern.includes("*") ? "" : "(?:/.*)?";
  return new RegExp(`^${source}${literalSuffix}$`, "u");
}
