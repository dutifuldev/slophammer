import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import {
  executeTypeScriptChecks,
  type CommandResult,
  type Runner
} from "../src/toolchecks/toolchecks.js";

describe("executeTypeScriptChecks package-script expansion", () => {
  it("does not treat npm ci as a package script named ci", async () => {
    const root = await packageFixture({
      bootstrap: "npm ci",
      ci: "eslint ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(args.includes("ci") ? failed("lint failed") : ok());
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(calls).toEqual(["npm run ci"]);
    expect(findings).toEqual([
      expect.objectContaining({
        rule_id: "ts.lint-required",
        message: "ESLint failed: lint failed"
      })
    ]);
  });

  it("still expands npm run-script wrappers", async () => {
    const root = await packageFixture({
      wrapper: "npm run-script ci",
      ci: "eslint ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    await executeTypeScriptChecks(root, runner);

    expect(calls).toEqual(["npm run wrapper"]);
  });
});

async function packageFixture(scripts: Readonly<Record<string, string>>): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), "slophammer-script-expansion-"));
  await mkdir(root, { recursive: true });
  await writeFile(path.join(root, "package.json"), JSON.stringify({ scripts }));
  return root;
}

function ok(): CommandResult {
  return { code: 0, stdout: "", stderr: "" };
}

function failed(message: string): CommandResult {
  return { code: 1, stdout: "", stderr: message };
}
