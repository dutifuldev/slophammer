import { filesNamed, type Snapshot } from "../repo/repo.js";

export function expandedPackageScriptSegments(
  segment: string,
  scripts: ReadonlyMap<string, string>,
  seen: ReadonlySet<string> = new Set()
): readonly string[] {
  const invocation = packageScriptInvocation(segment);
  if (invocation === undefined || seen.has(invocation.name)) {
    return [segment];
  }
  const script = scripts.get(invocation.name);
  if (script === undefined) {
    return [segment];
  }
  const expanded = commandSegments(`${script} ${invocation.forwardedArgs}`.trim());
  const nextSeen = new Set([...seen, invocation.name]);
  return expanded.length === 0
    ? [segment]
    : expanded.flatMap((item) => expandedPackageScriptSegments(item, scripts, nextSeen));
}

export function packageScripts(snapshot: Snapshot): ReadonlyMap<string, string> {
  const scripts = new Map<string, string>();
  for (const file of filesNamed(snapshot, "package.json").filter(
    (item) => !ignoredProjectDataPath(item.path)
  )) {
    try {
      const root = asRecord(JSON.parse(file.content) as unknown);
      for (const [name, value] of Object.entries(asRecord(root["scripts"]))) {
        if (typeof value === "string") {
          scripts.set(name.toLowerCase(), value);
        }
      }
    } catch {
      continue;
    }
  }
  return scripts;
}

type PackageScriptInvocation = {
  readonly name: string;
  readonly forwardedArgs: string;
};

function packageScriptInvocation(segment: string): PackageScriptInvocation | undefined {
  const normalized = stripPackageScriptLabel(normalizeCommandContent(segment));
  const words = normalized.split(" ");
  const invocation = scriptInvocationWords(words);
  return invocation === undefined
    ? undefined
    : { name: invocation, forwardedArgs: forwardedScriptArgs(normalized) };
}

function stripPackageScriptLabel(segment: string): string {
  return segment.replace(/^[a-z0-9:_-]+:\s+/u, "");
}

function scriptInvocationWords(words: readonly string[]): string | undefined {
  const manager = words[0];
  const command = words[1];
  if (manager === "npm") {
    return npmScriptInvocation(command, words[2]);
  }
  if (manager === "pnpm") {
    return pnpmScriptInvocation(command, words[2]);
  }
  if (manager === "yarn") {
    return yarnScriptInvocation(command, words[2]);
  }
  return undefined;
}

function npmScriptInvocation(
  command: string | undefined,
  script: string | undefined
): string | undefined {
  if (command === "run" || command === "run-script") {
    return validScriptName(script) ? npmScriptAlias(script) : undefined;
  }
  return npmScriptShorthands.has(command ?? "") ? npmScriptAlias(command ?? "") : undefined;
}

function pnpmScriptInvocation(
  command: string | undefined,
  script: string | undefined
): string | undefined {
  if (command === "run") {
    return validScriptName(script) ? npmScriptAlias(script) : undefined;
  }
  return npmScriptShorthands.has(command ?? "") ? npmScriptAlias(command ?? "") : undefined;
}

function yarnScriptInvocation(
  command: string | undefined,
  script: string | undefined
): string | undefined {
  if (command === "run") {
    return validScriptName(script) ? npmScriptAlias(script) : undefined;
  }
  if (!validScriptName(command) || yarnBuiltinCommands.has(command)) {
    return undefined;
  }
  return npmScriptAlias(command);
}

function validScriptName(value: string | undefined): value is string {
  return value !== undefined && /^[a-z0-9:_-]+$/u.test(value);
}

function npmScriptAlias(name: string): string {
  return name === "t" || name === "tst" ? "test" : name;
}

const npmScriptShorthands = new Set(["start", "stop", "restart", "test", "t", "tst"]);

const yarnBuiltinCommands = new Set([
  "add",
  "config",
  "dlx",
  "exec",
  "init",
  "install",
  "node",
  "npm",
  "remove",
  "set",
  "upgrade",
  "workspace",
  "workspaces"
]);

function forwardedScriptArgs(segment: string): string {
  return /\s--\s+(.+)$/u.exec(segment)?.[1] ?? "";
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

function ignoredProjectDataPath(filePath: string): boolean {
  return filePath.startsWith("fixtures/") || filePath.startsWith("templates/");
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}
