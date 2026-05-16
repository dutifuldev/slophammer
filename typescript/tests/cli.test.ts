import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { run } from "../src/cli/cli.js";
import { fixturePath } from "./helpers.js";

describe("run", () => {
  it("prints usage when no command is provided", async () => {
    const result = await run([]);

    expect(result.code).toBe(0);
    expect(result.stdout).toContain("usage:");
  });

  it("runs check from CLI args", async () => {
    const result = await run(["check", fixturePath("clean"), "--format", "json"]);

    expect(result.code).toBe(0);
    expect(result.stdout).toContain('"ok": true');
  });

  it("rejects unknown output formats", async () => {
    const result = await run(["check", fixturePath("clean"), "--format", "xml"]);

    expect(result.code).toBe(2);
    expect(result.stderr).toContain("unsupported format");
  });

  it("rejects malformed check args", async () => {
    await expect(run(["check"])).resolves.toMatchObject({ code: 2 });
    await expect(run(["check", fixturePath("clean"), "--bogus"])).resolves.toMatchObject({
      code: 2
    });
    await expect(
      run(["check", fixturePath("clean"), fixturePath("missing-readme")])
    ).resolves.toMatchObject({
      code: 2
    });
    await expect(run(["check", fixturePath("clean"), "--format"])).resolves.toMatchObject({
      code: 2
    });
  });

  it("runs explain from CLI args", async () => {
    const result = await run(["explain", "repo.readme-required"]);

    expect(result.code).toBe(0);
    expect(result.stdout).toContain("repo.readme-required");
  });

  it("rejects malformed explain and unknown commands", async () => {
    await expect(run(["explain"])).resolves.toMatchObject({ code: 2 });
    await expect(run(["unknown"])).resolves.toMatchObject({ code: 2 });
  });

  it("runs TypeScript dry from CLI args", async () => {
    const result = await run([
      "typescript",
      "dry",
      fixturePath("typescript-duplicate-blocks"),
      "--show-report"
    ]);

    expect(result.code).toBe(1);
    expect(result.stdout).toContain("copied-block");
  });

  it("excludes test files from TypeScript dry defaults", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "slophammer-ts-dry-tests-"));
    await mkdir(path.join(root, "tests"), { recursive: true });
    await mkdir(path.join(root, "src", "test"), { recursive: true });
    await mkdir(path.join(root, "src"), { recursive: true });
    const duplicate = Array.from({ length: 120 }, (_, index) => `expect(value${String(index)});`)
      .join("\n")
      .concat("\n");

    await writeFile(path.join(root, "tests", "helpers.ts"), duplicate);
    await writeFile(path.join(root, "src", "test", "helpers.ts"), duplicate);
    await writeFile(path.join(root, "src", "component.test.tsx"), duplicate);

    await expect(run(["typescript", "dry", root])).resolves.toMatchObject({ code: 0 });
  });

  it("parses TypeScript dry options", async () => {
    const root = fixturePath("typescript-duplicate-blocks");

    await expect(
      run(["typescript", "dry", root, "--format", "json", "--max-findings", "999"])
    ).resolves.toMatchObject({ code: 0 });
    await expect(run(["typescript", "dry", root, "--format", "xml"])).resolves.toMatchObject({
      code: 2
    });
    await expect(run(["typescript", "dry", root, "--max-findings", "-1"])).resolves.toMatchObject({
      code: 2
    });
    await expect(
      run(["typescript", "dry", root, "--max-findings", "999junk"])
    ).resolves.toMatchObject({
      code: 2
    });
    await expect(run(["typescript", "dry", root, "--max-findings", "1.5"])).resolves.toMatchObject({
      code: 2
    });
    await expect(run(["typescript", "dry", root, "--unknown"])).resolves.toMatchObject({
      code: 2
    });
    await expect(run(["typescript", "unknown"])).resolves.toMatchObject({ code: 2 });
  });
});
