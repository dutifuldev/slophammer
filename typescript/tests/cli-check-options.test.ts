import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { run } from "../src/cli/cli.js";
import { fixturePath } from "./helpers.js";

describe("check CLI options", () => {
  it("runs one rule from CLI args", async () => {
    const result = await run([
      "check",
      fixturePath("missing-readme"),
      "--only",
      "repo.readme-required",
      "--format",
      "json"
    ]);

    expect(result.code).toBe(1);
    expect(asReport(result.stdout).findings).toEqual([
      expect.objectContaining({ rule_id: "repo.readme-required" })
    ]);
  });

  it("runs dependency boundaries from direct CLI args", async () => {
    const result = await run(["boundaries", fixturePath("clean"), "--format", "json"]);

    expect(result.code).toBe(0);
    expect(result.stdout).toContain('"ok": true');
  });

  it("rejects unknown rule filters", async () => {
    await expect(
      run(["check", fixturePath("clean"), "--only", "missing.rule"])
    ).resolves.toMatchObject({
      code: 2
    });
  });

  it("rejects combining --baseline with --baseline-write", async () => {
    const result = await run(["check", fixturePath("clean"), "--baseline", "--baseline-write"]);

    expect(result.code).toBe(2);
    expect(result.stderr).toContain("--baseline and --baseline-write are mutually exclusive");
  });

  it("reads the baseline only behind the --baseline flag", async () => {
    const root = await baselinedRepo();

    const plain = await run(["check", root]);
    const baselined = await run(["check", root, "--baseline"]);

    expect(plain.code).toBe(1);
    expect(baselined.code).toBe(0);
    expect(baselined.stdout).toContain("findings baselined");
  });

  it("writes a baseline from the CLI", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "slophammer-cli-baseline-"));

    const result = await run(["check", root, "--baseline-write"]);

    expect(result.code).toBe(0);
    expect(result.stdout).toContain("baseline written:");
  });

  it("rejects baseline flags on the boundaries command", async () => {
    const result = await run(["boundaries", fixturePath("clean"), "--baseline"]);

    expect(result.code).toBe(2);
    expect(result.stderr).toContain("usage: slophammer-ts boundaries");
  });
});

async function baselinedRepo(): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), "slophammer-cli-baseline-"));
  const findings = [
    { rule_id: "repo.agents-required", path: "AGENTS.md" },
    { rule_id: "repo.ci-required", path: ".github/workflows" },
    { rule_id: "repo.readme-required", path: "README.md" }
  ];
  await writeFile(
    path.join(root, "slophammer-baseline.json"),
    `${JSON.stringify({ version: 1, findings })}\n`
  );
  return root;
}

function asReport(content: string): { readonly findings: readonly unknown[] } {
  const parsed: unknown = JSON.parse(content);
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return { findings: [] };
  }
  const findings = (parsed as Readonly<Record<string, unknown>>)["findings"];
  return { findings: Array.isArray(findings) ? findings : [] };
}
