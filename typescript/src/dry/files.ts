import { readdir, readFile, stat } from "node:fs/promises";
import path from "node:path";

import type { SourceFile } from "./types.js";

const skippedDirectories = new Set([
  ".git",
  ".stryker-tmp",
  "node_modules",
  "dist",
  "coverage",
  "build",
  "target"
]);
const sourceExtensions = [".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"] as const;

export async function loadSourceFiles(
  root: string,
  paths: readonly string[],
  exclude: readonly string[]
): Promise<readonly SourceFile[]> {
  const seen = new Set<string>();
  const files: SourceFile[] = [];
  for (const target of paths.length === 0 ? ["."] : paths) {
    await collectTarget(root, target, exclude, seen, files);
  }
  return files.sort((left, right) => left.path.localeCompare(right.path));
}

async function collectTarget(
  root: string,
  target: string,
  exclude: readonly string[],
  seen: Set<string>,
  files: SourceFile[]
): Promise<void> {
  const absolute = path.resolve(root, target);
  const info = await stat(absolute);
  if (!info.isDirectory()) {
    await appendSourceFile(root, absolute, exclude, seen, files);
    return;
  }
  await walk(root, absolute, exclude, seen, files);
}

async function walk(
  root: string,
  current: string,
  exclude: readonly string[],
  seen: Set<string>,
  files: SourceFile[]
): Promise<void> {
  for (const entry of await readdir(current, { withFileTypes: true })) {
    const filePath = path.join(current, entry.name);
    if (entry.isDirectory()) {
      if (!skippedDirectories.has(entry.name)) {
        await walk(root, filePath, exclude, seen, files);
      }
      continue;
    }
    if (entry.isFile()) {
      await appendSourceFile(root, filePath, exclude, seen, files);
    }
  }
}

async function appendSourceFile(
  root: string,
  filePath: string,
  exclude: readonly string[],
  seen: Set<string>,
  files: SourceFile[]
): Promise<void> {
  const rel = path.relative(root, filePath).split(path.sep).join("/");
  if (!isSourcePath(rel) || seen.has(rel) || excluded(rel, exclude)) {
    return;
  }
  seen.add(rel);
  files.push({ path: rel, content: await readFile(filePath, "utf8") });
}

function isSourcePath(filePath: string): boolean {
  return sourceExtensions.some((extension) => filePath.endsWith(extension));
}

function excluded(filePath: string, patterns: readonly string[]): boolean {
  return patterns.some((pattern) => globLikeMatch(filePath, pattern));
}

function globLikeMatch(filePath: string, pattern: string): boolean {
  if (pattern.endsWith("/**")) {
    const directory = pattern.slice(0, -3);
    return filePath === directory || filePath.startsWith(`${directory}/`);
  }
  if (pattern.startsWith("**/*")) {
    return filePath.endsWith(pattern.slice(4));
  }
  if (pattern.includes("*")) {
    const parts = pattern.split("*");
    return parts.every((part) => part === "" || filePath.includes(part));
  }
  return filePath === pattern || filePath.startsWith(`${pattern}/`);
}
