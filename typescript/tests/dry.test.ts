import { describe, expect, it } from "vitest";

import { findCopiedBlocks } from "../src/dry/copied-blocks.js";
import { applyDefaults, checkDry, findDry } from "../src/dry/dry.js";
import { tokenSequences } from "../src/dry/tokenize.js";
import { fixturePath } from "./helpers.js";

describe("findDry", () => {
  it("reports copied TypeScript blocks", async () => {
    const report = await findDry({
      root: fixturePath("typescript-duplicate-blocks"),
      paths: ["src"],
      exclude: [],
      maxFindings: 0,
      maxFindingsSet: true,
      copiedBlockEnabled: true,
      copiedBlockSet: true,
      copiedBlockTokens: 40,
      showReport: false,
      format: ""
    });

    expect(report.findings.length).toBeGreaterThan(0);
    expect(report.findings[0]?.kind).toBe("copied-block");
  });

  it("renders JSON reports and honors max findings", async () => {
    const result = await checkDry({
      root: fixturePath("typescript-duplicate-blocks"),
      paths: ["src"],
      exclude: [],
      maxFindings: 999,
      maxFindingsSet: true,
      copiedBlockEnabled: true,
      copiedBlockSet: true,
      copiedBlockTokens: 40,
      showReport: false,
      format: "json"
    });

    expect(result.code).toBe(0);
    expect(JSON.parse(result.output)).toHaveProperty("findings");
  });

  it("renders text reports on request", async () => {
    const result = await checkDry({
      root: fixturePath("typescript-duplicate-blocks"),
      paths: ["src"],
      exclude: [],
      maxFindings: 999,
      maxFindingsSet: true,
      copiedBlockEnabled: true,
      copiedBlockSet: true,
      copiedBlockTokens: 40,
      showReport: false,
      format: "text"
    });

    expect(result.output).toContain("DRY findings:");
  });

  it("applies production defaults", () => {
    expect(
      applyDefaults({
        root: "",
        paths: [],
        exclude: [],
        maxFindings: 0,
        maxFindingsSet: false,
        copiedBlockEnabled: false,
        copiedBlockSet: false,
        copiedBlockTokens: 0,
        showReport: false,
        format: ""
      })
    ).toMatchObject({
      root: ".",
      paths: ["."],
      maxFindings: 0,
      copiedBlockEnabled: true,
      copiedBlockTokens: 100
    });
  });

  it("does not report shifted overlapping windows from one repeated run", () => {
    const report = findCopiedBlocks(
      [
        {
          path: "src/repeated.ts",
          content: Array.from({ length: 26 }, () => "doThing();").join("\n")
        }
      ],
      100
    );

    expect(report).toEqual([]);
  });

  it("reports copied repeated runs without comparing every shifted window", () => {
    const content = Array.from({ length: 400 }, () => "doThing();").join("\n");
    const report = findCopiedBlocks(
      [
        { path: "src/a.ts", content },
        { path: "src/b.ts", content }
      ],
      100
    );

    expect(
      report.some(
        (finding) => finding.left.path === "src/a.ts" && finding.right.path === "src/b.ts"
      )
    ).toBe(true);
  });

  it("preserves template literal text in tokens", () => {
    const sequences = tokenSequences([
      { path: "src/a.ts", content: "const sql = `select ${name}`;\n" },
      { path: "src/b.ts", content: "const sql = `delete ${name}`;\n" }
    ]);

    expect(sequences[0]?.tokens.map((token) => token.tag)).not.toEqual(
      sequences[1]?.tokens.map((token) => token.tag)
    );
  });
});
