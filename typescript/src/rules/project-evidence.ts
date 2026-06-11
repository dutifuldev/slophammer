import { bindingWorkflowTriggers, type RepoFile, type Snapshot } from "../repo/repo.js";
import { parse as parseYAML } from "yaml";

const rootESLintConfigPaths = new Set([
  "eslint.config.js",
  "eslint.config.mjs",
  "eslint.config.cjs",
  ".eslintrc",
  ".eslintrc.js",
  ".eslintrc.cjs",
  ".eslintrc.json",
  ".eslintrc.yml",
  ".eslintrc.yaml"
]);

const rootOxlintConfigPaths = new Set([
  ".oxlintrc.json",
  ".oxlintrc.jsonc",
  "oxlint.config.json",
  "oxlint.config.jsonc"
]);

const rootTestRunnerConfigPaths = new Set([
  "vite.config.ts",
  "vite.config.mts",
  "vite.config.cts",
  "vite.config.js",
  "vite.config.mjs",
  "vite.config.cjs",
  "vitest.config.ts",
  "vitest.config.mts",
  "vitest.config.cts",
  "vitest.config.js",
  "vitest.config.mjs",
  "vitest.config.cjs",
  "jest.config.ts",
  "jest.config.js",
  "jest.config.mjs",
  "jest.config.cjs"
]);

// repoEvidenceFiles synthesizes root evidence into a project scope. Scoped
// command evidence is credited unconditionally downstream, so synthetics are
// only emitted where the scope does not retain the original file: the root
// scope keeps its own files and only needs replacements for the command
// evidence that nested project roots displace.
export function repoEvidenceFiles(
  snapshot: Snapshot,
  root: string,
  projectRoots: readonly string[]
): readonly RepoFile[] {
  return [...snapshot.files.values()].flatMap((file) => repoEvidenceFile(file, root, projectRoots));
}

function repoEvidenceFile(
  file: RepoFile,
  root: string,
  projectRoots: readonly string[]
): readonly RepoFile[] {
  if (root === ".") {
    return rootScopeEvidenceFile(file, root, projectRoots);
  }
  if (file.path === "package.json") {
    return [
      {
        path: "scripts/__repo_package_scripts.sh",
        content: scopedCommandContent(packageScriptContent(file), root, projectRoots)
      }
    ];
  }
  if (!rootEvidencePath(file.path)) {
    return [];
  }
  return [
    {
      path: repoEvidencePath(file.path),
      content: repoWorkflowPath(file.path)
        ? scopedWorkflowCommandContent(file.content, root, projectRoots)
        : repoScriptPath(file.path)
          ? scopedCommandContent(file.content, root, projectRoots)
          : file.content
    }
  ];
}

// The root scope retains its own package.json, configs, and — when no
// nested project roots exist — its own workflows and scripts, so only the
// displaced command evidence is re-synthesized in scoped form.
function rootScopeEvidenceFile(
  file: RepoFile,
  root: string,
  projectRoots: readonly string[]
): readonly RepoFile[] {
  if (nestedProjectRoots(root, projectRoots).length === 0) {
    return [];
  }
  if (repoWorkflowPath(file.path)) {
    return [
      {
        path: repoEvidencePath(file.path),
        content: scopedWorkflowCommandContent(file.content, root, projectRoots)
      }
    ];
  }
  if (repoScriptPath(file.path)) {
    return [
      {
        path: file.path,
        content: scopedCommandContent(file.content, root, projectRoots)
      }
    ];
  }
  return [];
}

function repoEvidencePath(filePath: string): string {
  if (repoWorkflowPath(filePath)) {
    return `scripts/__repo_workflow_${sanitizePath(filePath)}.sh`;
  }
  if (repoScriptPath(filePath)) {
    return filePath;
  }
  return `__repo__/${filePath}`;
}

function rootEvidencePath(filePath: string): boolean {
  return (
    repoWorkflowPath(filePath) ||
    repoScriptPath(filePath) ||
    rootESLintConfigPaths.has(filePath) ||
    rootOxlintConfigPaths.has(filePath) ||
    rootTestRunnerConfigPaths.has(filePath)
  );
}

function repoWorkflowPath(filePath: string): boolean {
  return (
    filePath.split("/").length === 3 &&
    filePath.startsWith(".github/workflows/") &&
    (filePath.endsWith(".yml") || filePath.endsWith(".yaml"))
  );
}

function repoScriptPath(filePath: string): boolean {
  return filePath.startsWith("scripts/") || (!filePath.includes("/") && filePath.endsWith(".sh"));
}

function sanitizePath(filePath: string): string {
  return filePath.replace(/[^a-zA-Z0-9]+/gu, "_").replace(/^_+|_+$/gu, "");
}

function scopedCommandContent(
  content: string,
  root: string,
  projectRoots: readonly string[]
): string {
  if (root === ".") {
    const nestedRoots = nestedProjectRoots(root, projectRoots);
    if (nestedRoots.length === 0) {
      return content;
    }
    return content
      .split("\n")
      .flatMap((line) => scopedCommandLine(line, root, projectRoots))
      .join("\n");
  }
  if (!multipleProjectScopes(projectRoots)) {
    return content;
  }
  return content
    .split("\n")
    .flatMap((line) => scopedCommandLine(line, root, projectRoots))
    .join("\n");
}

function scopedWorkflowCommandContent(
  content: string,
  root: string,
  projectRoots: readonly string[]
): string {
  const workflow = workflowRecord(content);
  if (workflow === undefined) {
    return scopedCommandContent(content, root, projectRoots);
  }
  if (nonBindingWorkflow(workflow)) {
    return "";
  }
  const commands = workflowCommands(workflow);
  if (root === ".") {
    const nestedRoots = nestedProjectRoots(root, projectRoots);
    if (nestedRoots.length === 0) {
      return commands.map((command) => command.run).join("\n");
    }
    return commands
      .filter((command) => !mentionsAnyRoot(command.workingDirectory, nestedRoots))
      .map((command) => scopedCommandContent(command.run, root, projectRoots))
      .filter((command) => command !== "")
      .join("\n");
  }
  if (!multipleProjectScopes(projectRoots)) {
    return commands.map((command) => command.run).join("\n");
  }
  return commands
    .flatMap((command) => scopedWorkflowCommand(command, root, projectRoots))
    .join("\n");
}

type WorkflowCommand = {
  readonly run: string;
  readonly workingDirectory: string;
};

function workflowCommands(workflow: Readonly<Record<string, unknown>>): readonly WorkflowCommand[] {
  const defaultWorkingDirectory = workflowDefaultWorkingDirectory(workflow);
  return Object.values(asRecord(workflow["jobs"])).flatMap((job) =>
    jobWorkflowCommands(job, defaultWorkingDirectory)
  );
}

function jobWorkflowCommands(
  job: unknown,
  workflowWorkingDirectory: string
): readonly WorkflowCommand[] {
  const record = asRecord(job);
  const defaultWorkingDirectory =
    stringValue(asRecord(asRecord(record["defaults"])["run"])["working-directory"]) ||
    workflowWorkingDirectory;
  const matrixCommands = jobMatrixCommands(record);
  return arrayValues(record["steps"]).flatMap((step) =>
    stepWorkflowCommand(step, defaultWorkingDirectory, matrixCommands)
  );
}

function jobMatrixCommands(job: Readonly<Record<string, unknown>>): readonly string[] {
  const matrix = asRecord(asRecord(job["strategy"])["matrix"]);
  const includeCommands = arrayValues(matrix["include"])
    .map((item) => stringValue(asRecord(item)["command"]))
    .filter((item) => item !== "");
  const directCommands = arrayValues(matrix["command"])
    .map(stringValue)
    .filter((item) => item !== "");
  return [...includeCommands, ...directCommands];
}

function workflowDefaultWorkingDirectory(workflow: Readonly<Record<string, unknown>>): string {
  return stringValue(asRecord(asRecord(workflow["defaults"])["run"])["working-directory"]);
}

function stepWorkflowCommand(
  step: unknown,
  defaultWorkingDirectory: string,
  matrixCommands: readonly string[]
): readonly WorkflowCommand[] {
  const record = asRecord(step);
  const run = stringValue(record["run"]);
  if (run === "") {
    return [];
  }
  const workingDirectory = stringValue(record["working-directory"]) || defaultWorkingDirectory;
  if (executableMatrixCommand(run) && matrixCommands.length > 0) {
    return matrixCommands.map((command) => ({
      run: run.replace(matrixCommandExpressionPattern, command),
      workingDirectory
    }));
  }
  return [{ run, workingDirectory }];
}

const matrixCommandExpressionPattern = /\$\{\{\s*matrix\.command\s*\}\}/gu;
const matrixCommandExpressionTestPattern = /\$\{\{\s*matrix\.command\s*\}\}/u;

function executableMatrixCommand(command: string): boolean {
  return splitCommandSegments(command).some(
    (segment) => segment.replace(matrixCommandExpressionTestPattern, "").trim() === ""
  );
}

function scopedWorkflowCommand(
  command: WorkflowCommand,
  root: string,
  projectRoots: readonly string[]
): readonly string[] {
  if (mentionsRoot(command.workingDirectory, root)) {
    return [command.run];
  }
  const otherRoots = projectRoots.filter((item) => item !== "." && item !== root);
  if (command.workingDirectory !== "" && mentionsAnyRoot(command.workingDirectory, otherRoots)) {
    return [];
  }
  const scoped = scopedCommandContent(command.run, root, projectRoots);
  return scoped === "" ? [] : [scoped];
}

function workflowRecord(content: string): Readonly<Record<string, unknown>> | undefined {
  try {
    return asRecord(parseYAML(content) as unknown);
  } catch {
    return undefined;
  }
}

// nonBindingWorkflow mirrors the binding filter in repo.ts commandFiles:
// scoped evidence is credited unconditionally downstream, so workflows that
// cannot fire for integration must be dropped at the scoping step.
function nonBindingWorkflow(workflow: Readonly<Record<string, unknown>>): boolean {
  return (
    Object.keys(asRecord(workflow["jobs"])).length > 0 && !bindingWorkflowTriggers(workflow["on"])
  );
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

function mentionsAnyRoot(content: string, roots: readonly string[]): boolean {
  return roots.some((root) => mentionsRoot(content, root));
}

function scopedCommandLine(
  line: string,
  root: string,
  projectRoots: readonly string[]
): readonly string[] {
  if (root === ".") {
    const nestedRoots = nestedProjectRoots(root, projectRoots);
    if (!mentionsAnyRoot(line, nestedRoots)) {
      return [line];
    }
    const scoped = scopedRootCommandSegments(line, nestedRoots);
    return scoped === "" ? [] : [scoped];
  }
  if (!mentionsRoot(line, root)) {
    return [];
  }
  const otherRoots = projectRoots.filter((item) => item !== "." && item !== root);
  if (!mentionsAnyRoot(line, otherRoots)) {
    return [line];
  }
  const scoped = scopedCommandSegments(line, root, projectRoots);
  return scoped === "" ? [] : [scoped];
}

function scopedCommandSegments(
  line: string,
  root: string,
  projectRoots: readonly string[]
): string {
  const concreteRoots = projectRoots.filter((item) => item !== ".");
  const kept: string[] = [];
  let activeRoot = "";
  for (const segment of splitCommandSegments(line)) {
    const segmentRoot = concreteRoots.find((item) => mentionsRoot(segment, item));
    if (segmentRoot !== undefined) {
      activeRoot = segmentRoot;
    }
    if (activeRoot === root) {
      kept.push(segment);
    }
  }
  return kept.join(" && ");
}

function scopedRootCommandSegments(line: string, nestedRoots: readonly string[]): string {
  const kept: string[] = [];
  let nestedScope = false;
  for (const segment of splitCommandSegments(line)) {
    if (mentionsAnyRoot(segment, nestedRoots)) {
      nestedScope = true;
    }
    if (!nestedScope) {
      kept.push(segment);
    }
  }
  return kept.join(" && ");
}

function splitCommandSegments(line: string): readonly string[] {
  return line
    .replaceAll("\\\n", " ")
    .split(/&&|;/u)
    .map((segment) => segment.trim())
    .filter((segment) => segment !== "");
}

function nestedProjectRoots(root: string, projectRoots: readonly string[]): readonly string[] {
  if (root === ".") {
    return projectRoots.filter((item) => item !== ".");
  }
  return projectRoots.filter((item) => item !== root && item.startsWith(`${root}/`));
}

function multipleProjectScopes(projectRoots: readonly string[]): boolean {
  return projectRoots.length > 1;
}

function packageScriptContent(file: RepoFile): string {
  try {
    const parsed: unknown = JSON.parse(file.content);
    const scripts = asRecord(asRecord(parsed)["scripts"]);
    return Object.entries(scripts)
      .filter((entry): entry is [string, string] => typeof entry[1] === "string")
      .map(([name, value]) => `${name}: ${value}`)
      .join("\n");
  } catch {
    return "";
  }
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

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
