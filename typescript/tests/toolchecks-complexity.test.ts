import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import {
  executeTypeScriptChecks,
  type CommandResult,
  type Runner
} from "../src/toolchecks/toolchecks.js";

describe("executeTypeScriptChecks complexity scripts", () => {
  it("runs ESLint complexity scripts with whitespace max-warnings syntax", async () => {
    const root = await packageFixture({
      ...requiredScripts(),
      complexity: "eslint . --max-warnings 0"
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(
          args.includes("complexity") ? { code: 1, stdout: "", stderr: "too complex" } : ok()
        );
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(calls).toContain("npm run complexity");
    expect(findings).toEqual([
      expect.objectContaining({
        rule_id: "ts.complexity-required",
        message: "complexity check failed: too complex"
      })
    ]);
  });
});

async function packageFixture(scripts: Readonly<Record<string, string>>): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), "slophammer-typescript-"));
  await mkdir(root, { recursive: true });
  await writeFile(path.join(root, "package.json"), JSON.stringify({ scripts }));
  return root;
}

function requiredScripts(): Readonly<Record<string, string>> {
  return {
    format: "prettier --check .",
    lint: "eslint .",
    typecheck: "tsc --noEmit",
    test: "vitest run",
    coverage: "vitest run --coverage",
    dry: "slophammer typescript dry ."
  };
}

function ok(): CommandResult {
  return { code: 0, stdout: "", stderr: "" };
}
