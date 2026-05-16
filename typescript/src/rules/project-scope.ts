import path from "node:path";

import { newSnapshot, type Snapshot } from "../repo/repo.js";
import { repoEvidenceFiles } from "./project-evidence.js";
import { compilerConfig, extendedConfigPaths } from "./typescript-config.js";

type ScopeFile = {
  readonly originalPath: string;
  readonly path: string;
  readonly content: string;
};

export function scopeSnapshot(
  snapshot: Snapshot,
  root: string,
  projectRoots: readonly string[]
): Snapshot {
  const files = scopeFiles(snapshot, root, projectRoots);
  const repoEvidence = repoEvidenceFiles(snapshot, root, projectRoots);
  return newSnapshot(snapshot.root, [
    ...repoEvidence,
    ...files
      .filter((file) => !repoEvidenceReplacesFile(root, projectRoots, file.originalPath))
      .map((file) => ({
        path: file.path,
        content: rootPackageContent(root, projectRoots, file)
      })),
    ...externalExtendedConfigFiles(snapshot, root, files)
  ]);
}

function scopeFiles(
  snapshot: Snapshot,
  root: string,
  projectRoots: readonly string[]
): readonly ScopeFile[] {
  return [...snapshot.files.values()]
    .filter((file) => fileBelongsToScope(file.path, root, projectRoots))
    .map((file) => ({
      originalPath: file.path,
      path: root === "." ? file.path : file.path.slice(root.length + 1),
      content: file.content
    }));
}

function fileBelongsToScope(
  filePath: string,
  root: string,
  projectRoots: readonly string[]
): boolean {
  if (root !== "." && !filePath.startsWith(`${root}/`)) {
    return false;
  }
  return !projectRoots.some((candidate) => nestedProjectContains(candidate, root, filePath));
}

function nestedProjectContains(candidate: string, root: string, filePath: string): boolean {
  if (candidate === root || candidate === ".") {
    return false;
  }
  if (root !== "." && !candidate.startsWith(`${root}/`)) {
    return false;
  }
  return filePath.startsWith(`${candidate}/`);
}

function repoEvidenceReplacesFile(
  root: string,
  projectRoots: readonly string[],
  filePath: string
): boolean {
  return (
    root === "." && nestedProjectRoots(projectRoots).length > 0 && rootCommandEvidence(filePath)
  );
}

function rootCommandEvidence(filePath: string): boolean {
  return (
    repoWorkflowPath(filePath) ||
    filePath.startsWith("scripts/") ||
    (!filePath.includes("/") && filePath.endsWith(".sh"))
  );
}

function repoWorkflowPath(filePath: string): boolean {
  return (
    filePath.split("/").length === 3 &&
    filePath.startsWith(".github/workflows/") &&
    (filePath.endsWith(".yml") || filePath.endsWith(".yaml"))
  );
}

function rootPackageContent(
  root: string,
  projectRoots: readonly string[],
  file: ScopeFile
): string {
  if (root !== "." || file.originalPath !== "package.json") {
    return file.content;
  }
  const nestedRoots = nestedProjectRoots(projectRoots);
  if (nestedRoots.length === 0) {
    return file.content;
  }
  return filterRootPackageScripts(file.content, nestedRoots);
}

function filterRootPackageScripts(content: string, nestedRoots: readonly string[]): string {
  try {
    const parsed: unknown = JSON.parse(content);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return content;
    }
    const root = parsed as Record<string, unknown>;
    if (
      typeof root["scripts"] !== "object" ||
      root["scripts"] === null ||
      Array.isArray(root["scripts"])
    ) {
      return content;
    }
    const scripts = root["scripts"] as Record<string, unknown>;
    root["scripts"] = Object.fromEntries(
      Object.entries(scripts).filter(([name, value]) => {
        return (
          typeof value !== "string" ||
          !nestedRoots.some((nestedRoot) => mentionsRoot(`${name}: ${value}`, nestedRoot))
        );
      })
    );
    return JSON.stringify(root);
  } catch {
    return content;
  }
}

function mentionsRoot(content: string, root: string): boolean {
  const normalized = content.toLowerCase();
  const candidate = root.toLowerCase();
  const pattern = new RegExp(
    `(?:^|[^a-z0-9._/-])(?:\\./)?${escapeRegExp(candidate)}(?:$|[\\/]|[^a-z0-9._-])`,
    "u"
  );
  return pattern.test(normalized);
}

function nestedProjectRoots(projectRoots: readonly string[]): readonly string[] {
  return projectRoots.filter((item) => item !== ".");
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function externalExtendedConfigFiles(
  snapshot: Snapshot,
  root: string,
  files: readonly ScopeFile[]
): readonly { readonly path: string; readonly content: string }[] {
  const collected = new Map<string, { readonly path: string; readonly content: string }>();
  for (const file of files.filter((item) => path.posix.basename(item.path) === "tsconfig.json")) {
    collectExtendedConfigFiles(snapshot, root, file.originalPath, collected, new Set());
  }
  return [...collected.values()];
}

function collectExtendedConfigFiles(
  snapshot: Snapshot,
  root: string,
  originalPath: string,
  collected: Map<string, { readonly path: string; readonly content: string }>,
  seen: ReadonlySet<string>
): void {
  if (seen.has(originalPath)) {
    return;
  }
  const file = snapshot.files.get(originalPath);
  if (file === undefined) {
    return;
  }
  for (const extendedPath of extendedConfigPaths(
    originalPath,
    compilerConfig(file.path, file.content)
  )) {
    const extended = snapshot.files.get(extendedPath);
    if (extended === undefined) {
      continue;
    }
    const scoped = root === "." ? extendedPath : path.posix.relative(root, extendedPath);
    collected.set(scoped, { path: scoped, content: extended.content });
    collectExtendedConfigFiles(
      snapshot,
      root,
      extendedPath,
      collected,
      new Set([...seen, originalPath])
    );
  }
}
