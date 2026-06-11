import type { Snapshot } from "../repo/repo.js";
import { typeScriptSourcePath } from "./source-paths.js";
import type { Definition, Finding } from "./types.js";

// A bare lint or type suppression in production TypeScript code is a
// finding: directives must carry a description on the line or sit under a
// preceding explanatory comment. The scan is line-based and reports the
// first offense per file.
export function suppressionFindings(
  definition: Definition,
  snapshot: Snapshot
): readonly Finding[] {
  const findings: Finding[] = [];
  for (const file of snapshot.files.values()) {
    if (!typeScriptSourcePath(file.path)) {
      continue;
    }
    const line = bareSuppressionLine(file.content);
    if (line !== undefined) {
      findings.push(suppressionFinding(definition, file.path, line));
    }
  }
  return findings;
}

function suppressionFinding(definition: Definition, filePath: string, line: number): Finding {
  return {
    rule_id: definition.id,
    severity: definition.severity,
    path: filePath,
    message: `${definition.message} (line ${String(line)})`
  };
}

function bareSuppressionLine(content: string): number | undefined {
  let previousLineIsComment = false;
  for (const [index, line] of content.split("\n").entries()) {
    if (bareSuppressionDirective(line, previousLineIsComment)) {
      return index + 1;
    }
    previousLineIsComment = explanatoryComment(line);
  }
  return undefined;
}

const directiveMarkers = [
  "eslint-disable",
  "oxlint-disable",
  "@ts-expect-error",
  "@ts-ignore",
  "biome-ignore"
] as const;

type DirectiveMarker = (typeof directiveMarkers)[number];

function bareSuppressionDirective(line: string, previousLineIsComment: boolean): boolean {
  const comment = commentText(line);
  const marker = directiveMarkers.find((candidate) => comment.includes(candidate));
  if (marker === undefined || previousLineIsComment) {
    return false;
  }
  const rest = comment.slice(comment.indexOf(marker) + marker.length);
  return !justifiedOnLine(marker, rest);
}

// Suppression directives only take effect inside comments, so directive
// names in code or string literals are not findings.
function commentText(line: string): string {
  const lineComment = line.indexOf("//");
  const blockComment = line.indexOf("/*");
  const start = [lineComment, blockComment].filter((index) => index >= 0);
  return start.length === 0 ? "" : line.slice(Math.min(...start));
}

function justifiedOnLine(marker: DirectiveMarker, rest: string): boolean {
  switch (marker) {
    case "@ts-ignore":
      return false;
    case "eslint-disable":
    case "oxlint-disable":
      return /\s--\s+\S/u.test(rest);
    case "biome-ignore":
      return /:\s*\S/u.test(descriptionText(rest));
    case "@ts-expect-error":
      return /\S/u.test(descriptionText(rest).replace(/^[\s:-]+/u, ""));
  }
}

function descriptionText(rest: string): string {
  return rest.replace(/\*\/.*$/u, "").trim();
}

// explanatoryComment treats a preceding comment line as justification, but
// never a comment that is itself a suppression directive.
function explanatoryComment(line: string): boolean {
  const trimmed = line.trimStart();
  const comment = trimmed.startsWith("//") || trimmed.startsWith("/*") || trimmed.startsWith("*");
  return comment && !directiveMarkers.some((marker) => trimmed.includes(marker));
}
