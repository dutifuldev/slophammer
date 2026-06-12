#!/usr/bin/env node
import { readdirSync, readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const skippedDirectories = new Set([
  ".venv",
  "mutants",
  ".git",
  "node_modules",
  "dist",
  "coverage",
  "target",
  "fixtures",
]);

const linkPattern = /!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/gu;

const failures = [];
for (const filePath of markdownFiles(root)) {
  for (const target of localLinkTargets(filePath)) {
    const resolved = path.resolve(path.dirname(filePath), target);
    if (!existsSync(resolved)) {
      failures.push(`${path.relative(root, filePath)} -> ${target}`);
    }
  }
}

if (failures.length > 0) {
  console.error("Broken local markdown links:");
  for (const failure of failures) {
    console.error(`  ${failure}`);
  }
  process.exit(1);
}
console.log("All local markdown links resolve.");

function markdownFiles(directory) {
  const found = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (!skippedDirectories.has(entry.name)) {
        found.push(...markdownFiles(path.join(directory, entry.name)));
      }
      continue;
    }
    if (entry.name.toLowerCase().endsWith(".md")) {
      found.push(path.join(directory, entry.name));
    }
  }
  return found;
}

function localLinkTargets(filePath) {
  const content = readFileSync(filePath, "utf8");
  const targets = [];
  for (const match of content.matchAll(linkPattern)) {
    const target = match[1] ?? "";
    if (externalLinkTarget(target)) {
      continue;
    }
    const withoutAnchor = target.split("#", 1)[0] ?? "";
    if (withoutAnchor.length > 0) {
      targets.push(decodeURIComponent(withoutAnchor));
    }
  }
  return targets;
}

function externalLinkTarget(target) {
  return (
    target.startsWith("http://") ||
    target.startsWith("https://") ||
    target.startsWith("mailto:") ||
    target.startsWith("#")
  );
}
