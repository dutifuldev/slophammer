import { parse as parseYAML } from "yaml";

export type RepoFile = {
  readonly path: string;
  readonly content: string;
};

export type Snapshot = {
  readonly root: string;
  readonly files: ReadonlyMap<string, RepoFile>;
};

export function newSnapshot(root: string, files: Iterable<RepoFile>): Snapshot {
  const byPath = new Map<string, RepoFile>();
  for (const file of files) {
    byPath.set(normalizePath(file.path), { path: normalizePath(file.path), content: file.content });
  }
  return {
    root,
    files: new Map([...byPath.entries()].sort(([left], [right]) => left.localeCompare(right)))
  };
}

export function hasFile(snapshot: Snapshot, path: string): boolean {
  return snapshot.files.has(normalizePath(path));
}

export function hasFileNamed(snapshot: Snapshot, ...names: readonly string[]): boolean {
  const wanted = new Set(names.map((name) => name.toLowerCase()));
  for (const filePath of snapshot.files.keys()) {
    const parts = filePath.split("/");
    const baseName = parts[parts.length - 1];
    if (baseName !== undefined && wanted.has(baseName.toLowerCase())) {
      return true;
    }
  }
  return false;
}

export function filesWithSuffix(snapshot: Snapshot, suffix: string): readonly RepoFile[] {
  return [...snapshot.files.values()].filter((file) => file.path.endsWith(suffix));
}

export function filesNamed(snapshot: Snapshot, ...names: readonly string[]): readonly RepoFile[] {
  const wanted = new Set(names.map((name) => name.toLowerCase()));
  return [...snapshot.files.values()].filter((file) => {
    const parts = file.path.split("/");
    const baseName = parts[parts.length - 1];
    return baseName !== undefined && wanted.has(baseName.toLowerCase());
  });
}

export function workflowFiles(snapshot: Snapshot): readonly RepoFile[] {
  return [...snapshot.files.values()].filter(
    (file) =>
      file.path.split("/").length === 3 &&
      file.path.startsWith(".github/workflows/") &&
      (file.path.endsWith(".yml") || file.path.endsWith(".yaml"))
  );
}

export function commandFiles(snapshot: Snapshot): readonly RepoFile[] {
  const workflows = workflowFiles(snapshot)
    .map(workflowCommandFile)
    .filter((file) => !ignoredEvidencePath(file.path) && file.content.trim() !== "");
  const synthetic = syntheticRepoEvidenceFiles(snapshot);
  const rootEvidence = joinedContents([...workflows, ...synthetic]);
  const scripts = reachableScriptFiles(snapshot, rootEvidence);
  const packages = reachablePackageScriptFiles(snapshot, rootEvidence + joinedContents(scripts));
  return [...workflows, ...synthetic, ...scripts, ...packages].filter(
    (file) => !ignoredEvidencePath(file.path) && file.content.trim() !== ""
  );
}

// Synthetic __repo_ evidence files are produced by scoping a real root
// workflow or root package scripts into a project scope, so they are already
// filtered binding evidence and stay credited without a referencing workflow.
export const syntheticRepoEvidencePrefix = "scripts/__repo_";

function syntheticRepoEvidencePath(filePath: string): boolean {
  return filePath.startsWith(syntheticRepoEvidencePrefix);
}

function syntheticRepoEvidenceFiles(snapshot: Snapshot): readonly RepoFile[] {
  return [...snapshot.files.values()]
    .filter((file) => syntheticRepoEvidencePath(file.path))
    .map(shellCommandFile);
}

function joinedContents(files: readonly RepoFile[]): string {
  return files.map((file) => file.content).join("\n") + "\n";
}

// reachableScriptFiles credits scripts only when binding workflow evidence
// references them, following script-to-script references one level deep.
function reachableScriptFiles(snapshot: Snapshot, rootEvidence: string): readonly RepoFile[] {
  const candidates = scriptFiles(snapshot).map(shellCommandFile);
  const firstHop = candidates.filter((file) => referencesFile(rootEvidence, file.path));
  const extended = rootEvidence + joinedContents(firstHop);
  return candidates.filter((file) => referencesFile(extended, file.path));
}

function referencesFile(evidence: string, filePath: string): boolean {
  const parts = filePath.split("/");
  const baseName = parts[parts.length - 1] ?? filePath;
  return containsWord(evidence, baseName);
}

function containsWord(evidence: string, word: string): boolean {
  for (let index = evidence.indexOf(word); index >= 0; index = evidence.indexOf(word, index + 1)) {
    const before = index === 0 ? "" : (evidence[index - 1] ?? "");
    const after = evidence[index + word.length] ?? "";
    if (!wordCharacter(before) && !wordCharacter(after)) {
      return true;
    }
  }
  return false;
}

function wordCharacter(value: string): boolean {
  return /^[A-Za-z0-9_-]$/u.test(value);
}

// reachablePackageScriptFiles credits package.json scripts only when binding
// evidence invokes them by name, following one level of chained npm-run
// references inside invoked scripts.
function reachablePackageScriptFiles(snapshot: Snapshot, evidence: string): readonly RepoFile[] {
  return filesNamed(snapshot, "package.json")
    .map((file) => ({
      path: file.path,
      content: reachablePackageScriptContent(file.content, evidence)
    }))
    .map(shellCommandFile);
}

function reachablePackageScriptContent(content: string, evidence: string): string {
  const scripts = parsedPackageScripts(content);
  const invoked = new Set([...scripts.keys()].filter((name) => scriptInvoked(evidence, name)));
  const chained = [...invoked].flatMap((name) =>
    [...scripts.keys()].filter((candidate) => scriptInvoked(scripts.get(name) ?? "", candidate))
  );
  for (const name of chained) {
    invoked.add(name);
  }
  return [...scripts.entries()]
    .filter(([name]) => invoked.has(name))
    .map(([name, value]) => `${name}: ${value}`)
    .join("\n");
}

function parsedPackageScripts(content: string): ReadonlyMap<string, string> {
  try {
    const parsed: unknown = JSON.parse(content);
    const scripts = asRecord(asRecord(parsed)["scripts"]);
    return new Map(
      Object.entries(scripts).filter(
        (entry): entry is [string, string] => typeof entry[1] === "string"
      )
    );
  } catch {
    return new Map();
  }
}

function scriptInvoked(evidence: string, name: string): boolean {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const runner = new RegExp(
    `\\b(?:npm|pnpm|yarn|bun)(?:\\s+run)?\\s+(?:-[\\w-]+\\s+)*${escaped}(?![\\w-])`,
    "u"
  );
  if (runner.test(evidence)) {
    return true;
  }
  return name === "test" && /\b(?:npm|pnpm|yarn|bun)\s+test\b/u.test(evidence);
}

export function normalizePath(path: string): string {
  return path.replaceAll("\\", "/").replace(/^\.\/+/, "");
}

function scriptFiles(snapshot: Snapshot): readonly RepoFile[] {
  return [...snapshot.files.values()].filter(
    (file) =>
      !syntheticRepoEvidencePath(file.path) &&
      (file.path.startsWith("scripts/") ||
        file.path.includes("/scripts/") ||
        file.path.endsWith(".sh"))
  );
}

function workflowCommandFile(file: RepoFile): RepoFile {
  return {
    path: file.path,
    content: stripCommentLines(extractWorkflowRunContent(file.content))
  };
}

function shellCommandFile(file: RepoFile): RepoFile {
  return {
    path: file.path,
    content: stripCommentLines(file.content.replaceAll("\\\n", " "))
  };
}

function extractWorkflowRunContent(content: string): string {
  const workflow = workflowRecord(content);
  if (workflow !== undefined && Object.keys(asRecord(workflow["jobs"])).length > 0) {
    return bindingWorkflowTriggers(workflow["on"]) ? workflowCommands(workflow).join("\n") : "";
  }
  return lineBasedRunCommands(content).join("\n");
}

function lineBasedRunCommands(content: string): readonly string[] {
  const lines = content.replaceAll("\r\n", "\n").split("\n");
  const commands: string[] = [];
  for (let index = 0; index < lines.length; index += 1) {
    const entry = workflowRunEntry(lines[index] ?? "");
    if (entry === undefined) {
      continue;
    }
    if (blockScalar(entry.value)) {
      const block = collectIndentedBlock(lines, index + 1, entry.indent);
      commands.push(block.content);
      index = block.nextIndex - 1;
      continue;
    }
    const command = inlineRunCommand(entry.value);
    if (command !== undefined) {
      commands.push(command);
    }
  }
  return commands;
}

const matrixCommandExpressionPattern = /\$\{\{\s*matrix\.command\s*\}\}/u;

function workflowCommands(workflow: Readonly<Record<string, unknown>>): readonly string[] {
  return Object.values(asRecord(workflow["jobs"])).flatMap(jobWorkflowCommands);
}

function jobWorkflowCommands(job: unknown): readonly string[] {
  const record = asRecord(job);
  if (neutralizedEntry(record)) {
    return [];
  }
  const matrixCommands = jobMatrixCommands(record);
  return arrayValues(record["steps"]).flatMap((step) => stepWorkflowCommands(step, matrixCommands));
}

function stepWorkflowCommands(step: unknown, matrixCommands: readonly string[]): readonly string[] {
  const record = asRecord(step);
  if (neutralizedEntry(record)) {
    return [];
  }
  const evidence: string[] = [];
  const uses = stringValue(record["uses"]);
  if (uses !== "") {
    evidence.push(`uses: ${uses}`);
  }
  const command = stringValue(record["run"]);
  if (command === "") {
    return evidence;
  }
  if (!directMatrixCommand(command) || matrixCommands.length === 0) {
    return [...evidence, command];
  }
  return [...evidence, ...matrixCommands];
}

// neutralizedEntry reports whether a job or step cannot run or cannot fail:
// a literal-false if condition or a literal continue-on-error. Non-literal
// expressions stay credited.
function neutralizedEntry(record: Readonly<Record<string, unknown>>): boolean {
  const continueOnError = record["continue-on-error"];
  if (continueOnError === true || literalExpressionValue(stringValue(continueOnError)) === "true") {
    return true;
  }
  const condition = record["if"];
  if (condition === false) {
    return true;
  }
  return literalExpressionValue(stringValue(condition)) === "false";
}

function literalExpressionValue(value: string): string {
  let trimmed = value.trim();
  if (trimmed.startsWith("${{") && trimmed.endsWith("}}")) {
    trimmed = trimmed.slice(3, -2).trim();
  }
  return trimmed;
}

// bindingWorkflowTriggers reports whether a workflow can fire for
// integration: pull requests, merge groups, schedules, or pushes whose
// branch filter is absent, wildcarded, or names an integration branch.
export function bindingWorkflowTriggers(on: unknown): boolean {
  if (typeof on === "string") {
    return bindingTriggerName(on);
  }
  if (Array.isArray(on)) {
    return on.some((name) => bindingTriggerName(stringValue(name)));
  }
  const record = asRecord(on);
  return Object.entries(record).some(([name, value]) => bindingTriggerEntry(name, value));
}

function bindingTriggerName(name: string): boolean {
  return ["push", "pull_request", "pull_request_target", "merge_group", "schedule"].includes(name);
}

function bindingTriggerEntry(name: string, value: unknown): boolean {
  if (["pull_request", "pull_request_target", "merge_group", "schedule"].includes(name)) {
    return true;
  }
  if (name !== "push") {
    return false;
  }
  const record = asRecord(value);
  const branches = record["branches"];
  if (branches === undefined) {
    // A tags-only push filter never fires for branch pushes, so it is a
    // release trigger, not integration CI.
    return !("tags" in record);
  }
  const patterns = Array.isArray(branches) ? branches.map(stringValue) : [stringValue(branches)];
  return patterns.some(integrationBranchPattern);
}

function integrationBranchPattern(pattern: string): boolean {
  if (pattern.includes("*")) {
    return true;
  }
  return ["main", "master", "trunk", "develop"].includes(pattern);
}

function directMatrixCommand(command: string): boolean {
  return (
    matrixCommandExpressionPattern.test(command) &&
    command.replace(matrixCommandExpressionPattern, "").trim() === ""
  );
}

function workflowRunEntry(
  line: string
): { readonly indent: number; readonly value: string } | undefined {
  const match = /^(\s*)(?:-\s*)?run:\s*(.*)$/u.exec(line);
  if (match === null) {
    return undefined;
  }
  return {
    indent: match[1]?.length ?? 0,
    value: (match[2] ?? "").trimEnd()
  };
}

function workflowRecord(content: string): Readonly<Record<string, unknown>> | undefined {
  try {
    return asRecord(parseYAML(content) as unknown);
  } catch {
    return undefined;
  }
}

function jobMatrixCommands(job: unknown): readonly string[] {
  const matrix = asRecord(asRecord(asRecord(job)["strategy"])["matrix"]);
  const includeCommands = arrayValues(matrix["include"])
    .map((item) => stringValue(asRecord(item)["command"]))
    .filter((item) => item !== "");
  const directCommands = arrayValues(matrix["command"])
    .map(stringValue)
    .filter((item) => item !== "");
  return [...includeCommands, ...directCommands];
}

function inlineRunCommand(value: string): string | undefined {
  const command = value.trim();
  return command === "" ? undefined : unquote(command);
}

function blockScalar(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.startsWith("|") || trimmed.startsWith(">");
}

function collectIndentedBlock(
  lines: readonly string[],
  startIndex: number,
  parentIndent: number
): { readonly content: string; readonly nextIndex: number } {
  const kept: string[] = [];
  let blockIndent: number | undefined;
  let index = startIndex;
  for (; index < lines.length; index += 1) {
    const line = lines[index] ?? "";
    if (line.trim() === "") {
      kept.push("");
      continue;
    }
    const indent = leadingSpaces(line);
    if (indent <= parentIndent) {
      break;
    }
    blockIndent ??= indent;
    if (indent < blockIndent) {
      break;
    }
    kept.push(line.slice(blockIndent));
  }
  return { content: kept.join("\n"), nextIndex: index };
}

function leadingSpaces(line: string): number {
  return /^\s*/u.exec(line)?.[0].length ?? 0;
}

function unquote(value: string): string {
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1);
  }
  return value;
}

function stripCommentLines(content: string): string {
  return content
    .split("\n")
    .map((line) => line.split("#", 1)[0] ?? "")
    .filter((line) => line.trim() !== "")
    .join("\n");
}

function ignoredEvidencePath(filePath: string): boolean {
  return filePath.startsWith("fixtures/") || filePath.startsWith("templates/");
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

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}
