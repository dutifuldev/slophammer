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
  return [
    ...workflowFiles(snapshot).map(workflowCommandFile),
    ...scriptFiles(snapshot).map(shellCommandFile),
    ...packageScriptFiles(snapshot).map(shellCommandFile)
  ].filter((file) => !ignoredEvidencePath(file.path) && file.content.trim() !== "");
}

export function normalizePath(path: string): string {
  return path.replaceAll("\\", "/").replace(/^\.\/+/, "");
}

function scriptFiles(snapshot: Snapshot): readonly RepoFile[] {
  return [...snapshot.files.values()].filter(
    (file) =>
      file.path.startsWith("scripts/") ||
      file.path.includes("/scripts/") ||
      file.path.endsWith(".sh")
  );
}

function packageScriptFiles(snapshot: Snapshot): readonly RepoFile[] {
  return filesNamed(snapshot, "package.json").map((file) => ({
    path: file.path,
    content: packageScriptContent(file.content)
  }));
}

function packageScriptContent(content: string): string {
  try {
    const parsed: unknown = JSON.parse(content);
    const scripts = asRecord(asRecord(parsed)["scripts"]);
    return Object.entries(scripts)
      .filter((entry): entry is [string, string] => typeof entry[1] === "string")
      .map(([name, value]) => `${name}: ${value}`)
      .join("\n");
  } catch {
    return "";
  }
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
  if (workflow !== undefined) {
    return workflowCommands(workflow).join("\n");
  }
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
  return commands.join("\n");
}

const matrixCommandExpressionPattern = /\$\{\{\s*matrix\.command\s*\}\}/u;

function workflowCommands(workflow: Readonly<Record<string, unknown>>): readonly string[] {
  return Object.values(asRecord(workflow["jobs"])).flatMap(jobWorkflowCommands);
}

function jobWorkflowCommands(job: unknown): readonly string[] {
  const record = asRecord(job);
  const matrixCommands = jobMatrixCommands(record);
  return arrayValues(record["steps"]).flatMap((step) => stepWorkflowCommands(step, matrixCommands));
}

function stepWorkflowCommands(step: unknown, matrixCommands: readonly string[]): readonly string[] {
  const command = stringValue(asRecord(step)["run"]);
  if (command === "") {
    return [];
  }
  if (!directMatrixCommand(command) || matrixCommands.length === 0) {
    return [command];
  }
  return matrixCommands;
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
