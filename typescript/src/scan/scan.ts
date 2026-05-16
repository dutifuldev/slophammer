import { readdir, readFile, stat } from "node:fs/promises";
import path from "node:path";

import { newSnapshot, type RepoFile, type Snapshot } from "../repo/repo.js";

const maxFileBytes = 1 << 20;
const skippedDirectories = new Set([
  ".git",
  "node_modules",
  ".venv",
  ".mypy_cache",
  ".pytest_cache",
  ".ruff_cache",
  ".stryker-tmp",
  "__pycache__",
  "dist",
  "coverage"
]);

export async function scanRepo(root: string): Promise<Snapshot> {
  const absoluteRoot = path.resolve(root);
  const files: RepoFile[] = [];
  await walk(absoluteRoot, absoluteRoot, files);
  return newSnapshot(absoluteRoot, files);
}

async function walk(root: string, current: string, files: RepoFile[]): Promise<void> {
  const entries = await readdir(current, { withFileTypes: true });
  for (const entry of entries) {
    const filePath = path.join(current, entry.name);
    if (entry.isDirectory()) {
      if (!skippedDirectories.has(entry.name)) {
        await walk(root, filePath, files);
      }
      continue;
    }
    if (!entry.isFile()) {
      continue;
    }
    const file = await readSmallTextFile(root, filePath);
    if (file !== undefined) {
      files.push(file);
    }
  }
}

async function readSmallTextFile(root: string, filePath: string): Promise<RepoFile | undefined> {
  const info = await stat(filePath);
  if (info.size > maxFileBytes) {
    return undefined;
  }
  const content = await readFile(filePath, "utf8");
  if (content.includes("\0")) {
    return undefined;
  }
  return {
    path: path.relative(root, filePath).split(path.sep).join("/"),
    content
  };
}
