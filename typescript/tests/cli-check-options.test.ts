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
});

function asReport(content: string): { readonly findings: readonly unknown[] } {
  const parsed: unknown = JSON.parse(content);
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return { findings: [] };
  }
  const findings = (parsed as Readonly<Record<string, unknown>>)["findings"];
  return { findings: Array.isArray(findings) ? findings : [] };
}
