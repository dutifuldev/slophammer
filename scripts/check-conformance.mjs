#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const repoFixtures = [
  "clean",
  "missing-agents",
  "missing-ci",
  "missing-readme",
];
const goFixtures = [
  ...repoFixtures,
  "go-clean",
  "go-bad-dependency",
  "go-missing-complexity",
  "go-missing-coverage",
  "go-missing-crap",
  "go-missing-dry",
  "go-missing-lint",
  "go-missing-module",
  "go-missing-mutation",
  "go-missing-tests",
  "go-missing-vet",
];
const typeScriptFixtures = [
  ...repoFixtures,
  "typescript-clean",
  "typescript-bad-dependency",
  "typescript-duplicate-blocks",
  "typescript-missing-any-rule",
  "typescript-missing-complexity",
  "typescript-missing-coverage",
  "typescript-missing-dry",
  "typescript-missing-format",
  "typescript-missing-lint",
  "typescript-missing-mutation",
  "typescript-missing-package",
  "typescript-missing-strict",
  "typescript-missing-tests",
  "typescript-missing-typecheck",
  "typescript-missing-unsafe-types",
  "adoption-before",
  "adoption-after",
];
const rustFixtures = [
  ...repoFixtures,
  "rust-clean",
  "rust-bad-dependency",
  "rust-missing-audit",
  "rust-missing-ci",
  "rust-missing-clippy",
  "rust-missing-coverage",
  "rust-missing-dry",
  "rust-missing-fmt",
  "rust-missing-msrv",
  "rust-missing-mutation",
  "rust-missing-tests",
  "rust-unsafe",
];
const rustErrorFixtures = ["rust-invalid-config", "rust-unknown-config"];

run("npm", ["run", "build"], path.join(root, "typescript"), [0]);

for (const fixture of goFixtures) {
  assertFixture({
    implementation: "go",
    fixture,
    command: "go",
    args: [
      "run",
      "./cmd/slophammer-go",
      "check",
      fixturePath(fixture),
      "--format",
      "json",
    ],
    cwd: path.join(root, "go"),
  });
}

for (const fixture of typeScriptFixtures) {
  assertFixture({
    implementation: "typescript",
    fixture,
    command: "node",
    args: [
      "dist/src/cli/main.js",
      "check",
      fixturePath(fixture),
      "--format",
      "json",
    ],
    cwd: path.join(root, "typescript"),
  });
}

for (const fixture of rustFixtures) {
  assertFixture({
    implementation: "rust",
    fixture,
    command: "cargo",
    args: [
      "run",
      "-q",
      "-p",
      "slophammer-rs",
      "--",
      "check",
      fixturePath(fixture),
      "--format",
      "json",
    ],
    cwd: path.join(root, "rust"),
  });
}

for (const fixture of rustErrorFixtures) {
  run(
    "cargo",
    [
      "run",
      "-q",
      "-p",
      "slophammer-rs",
      "--",
      "check",
      fixturePath(fixture),
      "--format",
      "json",
    ],
    path.join(root, "rust"),
    [2],
  );
}

console.log(
  `Conformance passed: ${String(goFixtures.length)} Go fixtures, ${String(typeScriptFixtures.length)} TypeScript fixtures, ${String(rustFixtures.length)} Rust fixtures, ${String(rustErrorFixtures.length)} Rust error fixtures`,
);

function assertFixture({ implementation, fixture, command, args, cwd }) {
  const expected = expectedReport(fixture);
  const expectedCode = expected.ok ? 0 : 1;
  const result = run(command, args, cwd, [expectedCode]);
  const actual = normalizeReport(JSON.parse(result.stdout));
  const normalizedExpected = normalizeReport(expected);
  if (JSON.stringify(actual) !== JSON.stringify(normalizedExpected)) {
    throw new Error(
      `${implementation} fixture ${fixture} report mismatch\nexpected:\n${JSON.stringify(normalizedExpected, null, 2)}\nactual:\n${JSON.stringify(actual, null, 2)}`,
    );
  }
}

function expectedReport(fixture) {
  return JSON.parse(
    readFileSync(
      path.join(root, "fixtures", "expected", `${fixture}.json`),
      "utf8",
    ),
  );
}

function fixturePath(fixture) {
  return path.join(root, "fixtures", "repos", fixture);
}

function normalizeReport(report) {
  return {
    ok: Boolean(report.ok),
    findings: [...report.findings].sort((left, right) => {
      const leftKey = `${left.rule_id}\0${left.path}\0${left.message}`;
      const rightKey = `${right.rule_id}\0${right.path}\0${right.message}`;
      return leftKey.localeCompare(rightKey);
    }),
  };
}

function run(command, args, cwd, acceptedCodes) {
  const result = spawnSync(command, args, {
    cwd,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (!acceptedCodes.includes(result.status ?? -1)) {
    throw new Error(
      `${command} ${args.join(" ")} failed with ${String(result.status)} in ${cwd}\n${result.stdout}${result.stderr}`,
    );
  }
  return { stdout: result.stdout, stderr: result.stderr };
}
