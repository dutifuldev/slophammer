import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import {
  executeTypeScriptChecks,
  type CommandResult,
  type Runner
} from "../src/toolchecks/toolchecks.js";

describe("executeTypeScriptChecks", () => {
  it("runs required package scripts with npm by default", async () => {
    const root = await packageFixture(requiredScripts());
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    await expect(executeTypeScriptChecks(root, runner)).resolves.toEqual([]);
    expect(calls).toEqual([
      "npm run format",
      "npm run lint",
      "npm run typecheck",
      "npm run test",
      "npm run coverage",
      "npm run dry"
    ]);
  });

  it("returns no findings when package.json is absent", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "slophammer-no-package-"));

    await expect(executeTypeScriptChecks(root, silentRunner())).resolves.toEqual([]);
  });

  it("does not report missing scripts as execute failures", async () => {
    const root = await packageFixture({ lint: "eslint ." });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([]);
    expect(calls).toEqual(["npm run lint"]);
  });

  it("runs noncanonical scripts that declare accepted checks", async () => {
    const root = await packageFixture({
      check:
        "prettier --check . && eslint . && tsc --noEmit && vitest run && vitest run --coverage && slophammer typescript dry ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([]);
    expect(calls).toEqual(["npm run check"]);
  });

  it("reports one neutral finding when an aggregate script fails", async () => {
    const root = await packageFixture({
      check:
        "prettier --check . && eslint . && tsc --noEmit && vitest run && vitest run --coverage && slophammer typescript dry ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve({ code: 1, stdout: "", stderr: "lint failed\nmore" });
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(calls).toEqual(["npm run check"]);
    expect(findings).toEqual([
      {
        rule_id: "ts.lint-required",
        severity: "error",
        path: "package.json",
        message: "aggregate TypeScript check failed: lint failed"
      }
    ]);
  });

  it("prefers matching scripts over nonmatching canonical names", async () => {
    const root = await packageFixture({
      lint: "echo ok",
      check: "eslint ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([]);
    expect(calls).toEqual(["npm run check"]);
  });

  it("reports failing scripts with the first output line", async () => {
    const root = await packageFixture(requiredScripts());
    const runner: Runner = {
      run: (_cwd, _command, args) =>
        Promise.resolve(
          args.includes("lint") ? { code: 1, stdout: "", stderr: "lint failed\nmore" } : ok()
        )
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([
      {
        rule_id: "ts.lint-required",
        severity: "error",
        path: "package.json",
        message: "ESLint failed: lint failed"
      }
    ]);
  });
});

describe("executeTypeScriptChecks script matching", () => {
  it("does not run aggregate scripts that include full mutation testing", async () => {
    const root = await packageFixture({
      check:
        "prettier --check . && eslint . && tsc --noEmit && vitest run && vitest run --coverage && slophammer typescript dry . && stryker run"
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([]);
    expect(calls).toEqual([]);
  });

  it("runs coverage wrappers resolved through package scripts", async () => {
    const root = await packageFixture({
      ...requiredScripts(),
      coverage: "npm run test -- --coverage"
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(
          args.includes("coverage") ? { code: 1, stdout: "", stderr: "coverage failed" } : ok()
        );
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(calls).toContain("npm run coverage");
    expect(findings).toEqual([
      expect.objectContaining({
        rule_id: "ts.coverage-required",
        message: "coverage gate failed: coverage failed"
      })
    ]);
  });

  it("runs local TypeScript DRY invocations", async () => {
    const root = await packageFixture({
      ...requiredScripts(),
      dry: "node dist/src/cli/main.js typescript dry ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(
          args.includes("dry") ? { code: 1, stdout: "DRY failed", stderr: "" } : ok()
        );
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(calls).toContain("npm run dry");
    expect(findings).toEqual([
      expect.objectContaining({
        rule_id: "ts.dry-required",
        message: "DRY check failed: DRY failed"
      })
    ]);
  });
});

describe("executeTypeScriptChecks errors", () => {
  it("throws infrastructure errors from the runner", async () => {
    const root = await packageFixture(requiredScripts());
    const runner: Runner = {
      run: () =>
        Promise.resolve({
          code: 2,
          stdout: "",
          stderr: "spawn pnpm ENOENT",
          infrastructureError: true
        })
    };

    await expect(executeTypeScriptChecks(root, runner)).rejects.toThrow(
      "failed to run npm run format"
    );
  });
});

describe("executeTypeScriptChecks package managers", () => {
  it("uses pnpm when pnpm-lock.yaml is present", async () => {
    const root = await packageFixture(requiredScripts());
    await writeFile(path.join(root, "pnpm-lock.yaml"), "");
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    await executeTypeScriptChecks(root, runner);

    expect(calls[0]).toBe("pnpm run format");
  });

  it("uses a workspace lockfile for nested packages", async () => {
    const workspace = await mkdtemp(path.join(tmpdir(), "slophammer-workspace-"));
    const packageRoot = path.join(workspace, "packages", "app");
    await mkdir(packageRoot, { recursive: true });
    await writeFile(path.join(workspace, "pnpm-lock.yaml"), "");
    await writeFile(
      path.join(packageRoot, "package.json"),
      JSON.stringify({ scripts: requiredScripts() })
    );
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    await executeTypeScriptChecks(packageRoot, runner, workspace);

    expect(calls[0]).toBe("pnpm run format");
  });

  it("uses yarn when yarn.lock is present", async () => {
    const root = await packageFixture(requiredScripts());
    await writeFile(path.join(root, "yarn.lock"), "");
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    await executeTypeScriptChecks(root, runner);

    expect(calls[0]).toBe("yarn format");
  });

  it("runs optional mutation checks when configured", async () => {
    const root = await packageFixture({ ...requiredScripts(), mutate: "stryker run" });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(
          args.includes("mutate") ? { code: 1, stdout: "mutation failed", stderr: "" } : ok()
        );
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(calls).toContain("npm run mutate -- --dryRunOnly");
    expect(findings[0]?.rule_id).toBe("ts.mutation-required");
    expect(findings[0]?.message).toContain("mutation dry-run failed");
  });

  it("runs noncanonical pure mutation checks with dry-run arguments", async () => {
    const root = await packageFixture({ ...requiredScripts(), mutation: "stryker run" });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    await executeTypeScriptChecks(root, runner);

    expect(calls).toContain("npm run mutation -- --dryRunOnly");
  });

  it("does not run missing TypeScript mutation command placeholders", async () => {
    const root = await packageFixture({
      ...requiredScripts(),
      mutate: "slophammer typescript mutate ."
    });
    const calls: string[] = [];
    const runner: Runner = {
      run: (_cwd, command, args) => {
        calls.push([command, ...args].join(" "));
        return Promise.resolve(ok());
      }
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([]);
    expect(calls).not.toContain("npm run mutate -- --dryRunOnly");
  });

  it("runs optional complexity checks when configured", async () => {
    const root = await packageFixture({
      ...requiredScripts(),
      complexity: "eslint . --max-warnings=0"
    });
    const runner: Runner = {
      run: (_cwd, _command, args) =>
        Promise.resolve(
          args.includes("complexity") ? { code: 1, stdout: "", stderr: "complexity failed" } : ok()
        )
    };

    const findings = await executeTypeScriptChecks(root, runner);

    expect(findings).toEqual([
      expect.objectContaining({
        rule_id: "ts.complexity-required",
        message: "complexity check failed: complexity failed"
      })
    ]);
  });

  it("rejects multiple package lockfiles", async () => {
    const root = await packageFixture(requiredScripts());
    await writeFile(path.join(root, "pnpm-lock.yaml"), "");
    await writeFile(path.join(root, "package-lock.json"), "");

    await expect(executeTypeScriptChecks(root, silentRunner())).rejects.toThrow(
      "multiple package lockfiles"
    );
  });
});

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

async function packageFixture(scripts: Readonly<Record<string, string>>): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), "slophammer-ts-"));
  await writeFile(path.join(root, "package.json"), JSON.stringify({ scripts }));
  return root;
}

function ok(): CommandResult {
  return { code: 0, stdout: "", stderr: "" };
}

function silentRunner(): Runner {
  return { run: () => Promise.resolve(ok()) };
}
