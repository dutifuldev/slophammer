import { spawnSync } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

const root = process.cwd();
const temp = await mkdtemp(path.join(tmpdir(), "slophammer-ts-bin-"));
let tarball: string | undefined;

try {
  const pack = run("npm", ["pack", "--json"], root);
  tarball = path.join(root, packedFileName(pack.stdout));

  run("npm", ["install", "--silent", "--prefix", temp, tarball], root);

  assertHelpIncludesPublicUsage("slophammer-ts");
  assertHelpIncludesPublicUsage("slophammer");
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

function packedFileName(output: string): string {
  const parsed = JSON.parse(output) as unknown;
  if (!Array.isArray(parsed) || parsed.length === 0) {
    throw new Error(`npm pack did not return package metadata: ${output}`);
  }
  const first = parsed[0] as unknown;
  if (!record(first) || typeof first["filename"] !== "string") {
    throw new Error(`npm pack metadata did not include a filename: ${output}`);
  }
  return first["filename"];
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
