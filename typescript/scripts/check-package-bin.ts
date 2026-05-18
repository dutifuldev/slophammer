import { spawnSync } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

const root = process.cwd();
const temp = await mkdtemp(path.join(tmpdir(), "slophammer-ts-bin-"));
let tarball: string | undefined;

try {
  const pack = run("npm", ["pack", "--json"], root);
  const metadata = packedPackageMetadata(pack.stdout);
  assertPackageScope(metadata.files);
  tarball = path.join(root, metadata.filename);

  run("npm", ["install", "--silent", "--prefix", temp, tarball], root);

  assertHelpIncludesPublicUsage("slophammer-ts");
  assertFixtureCheckPasses("slophammer-ts");
} finally {
  if (tarball !== undefined) {
    await rm(tarball, { force: true });
  }
  await rm(temp, { recursive: true, force: true });
}

function assertHelpIncludesPublicUsage(name: string): void {
  const help = run(binPath(name), ["help"], root);
  if (!help.stdout.includes("slophammer-ts check")) {
    throw new Error(`${name} help did not print public usage:\n${help.stdout}`);
  }
}

function assertFixtureCheckPasses(name: string): void {
  const fixture = path.resolve(root, "..", "fixtures", "repos", "typescript-clean");
  const result = run(binPath(name), ["check", fixture, "--format", "json"], root);
  if (!result.stdout.includes('"ok": true')) {
    throw new Error(`${name} check did not pass the clean TypeScript fixture:\n${result.stdout}`);
  }
}

type PackedPackageMetadata = {
  readonly filename: string;
  readonly files: readonly PackedFile[];
};

type PackedFile = {
  readonly path: string;
};

function packedPackageMetadata(output: string): PackedPackageMetadata {
  const parsed = JSON.parse(output) as unknown;
  if (!Array.isArray(parsed) || parsed.length === 0) {
    throw new Error(`npm pack did not return package metadata: ${output}`);
  }
  const first = parsed[0] as unknown;
  if (!record(first) || typeof first["filename"] !== "string" || !Array.isArray(first["files"])) {
    throw new Error(`npm pack metadata did not include a filename: ${output}`);
  }
  return {
    filename: first["filename"],
    files: first["files"].map(packedFile)
  };
}

function packedFile(value: unknown): PackedFile {
  if (!record(value) || typeof value["path"] !== "string") {
    throw new Error(`npm pack metadata included an invalid file entry: ${JSON.stringify(value)}`);
  }
  return { path: value["path"] };
}

function assertPackageScope(files: readonly PackedFile[]): void {
  const paths = files.map((file) => file.path).sort();
  const required = ["README.md", "dist/src/cli/main.js", "dist/src/app/app.js", "package.json"];
  for (const requiredPath of required) {
    if (!paths.includes(requiredPath)) {
      throw new Error(`packed package is missing ${requiredPath}`);
    }
  }
  const forbidden = [
    /^src\//u,
    /^tests\//u,
    /^dist\/tests\//u,
    /^scripts\//u,
    /^coverage\//u,
    /^node_modules\//u,
    /^fixtures\//u,
    /^vitest/u,
    /^eslint\.config/u,
    /^tsconfig\.json$/u,
    /^stryker\.conf\.json$/u
  ];
  const leaked = paths.filter((filePath) => forbidden.some((pattern) => pattern.test(filePath)));
  if (leaked.length > 0) {
    throw new Error(`packed package includes development-only files:\n${leaked.join("\n")}`);
  }
}

function record(value: unknown): value is Readonly<Record<string, unknown>> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function binPath(name: string): string {
  const fileName = process.platform === "win32" ? `${name}.cmd` : name;
  return path.join(temp, "node_modules", ".bin", fileName);
}

function run(command: string, args: readonly string[], cwd: string): RunResult {
  const result = spawnSync(command, args, {
    cwd,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"]
  });
  if (result.status !== 0) {
    throw new Error(
      `${command} ${args.join(" ")} failed with ${String(result.status)}\n${result.stdout}${result.stderr}`
    );
  }
  return { stdout: result.stdout, stderr: result.stderr };
}

type RunResult = {
  readonly stdout: string;
  readonly stderr: string;
};
