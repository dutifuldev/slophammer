import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { check } from "../src/app/app.js";
import { fixturePath, normalizeReport, parseReport, repoRoot } from "./helpers.js";

let goBinary = "";
let tempDirectory = "";

describe("shared fixture equivalence", () => {
  beforeAll(() => {
    tempDirectory = mkdtempSync(path.join(tmpdir(), "slophammer-go-equivalence-"));
    goBinary = path.join(tempDirectory, "slophammer");
    const build = spawnSync("go", ["build", "-o", goBinary, "./cmd/slophammer"], {
      cwd: path.join(repoRoot, "go"),
      encoding: "utf8"
    });

    if (build.status !== 0) {
      throw new Error(`failed to build Go checker: ${build.stderr}`);
    }
  }, 30_000);

  afterAll(() => {
    if (tempDirectory !== "") {
      rmSync(tempDirectory, { recursive: true, force: true });
    }
  });

  for (const name of ["clean", "missing-readme", "missing-agents", "missing-ci"]) {
    it(`matches Go implementation for ${name}`, async () => {
      const fixture = fixturePath(name);
      const typescript = await check({ root: fixture, format: "json", execute: false });
      const go = spawnSync(goBinary, ["check", fixture, "--format", "json"], {
        cwd: path.join(repoRoot, "go"),
        encoding: "utf8"
      });

      expect(normalizeReport(parseReport(typescript.stdout))).toEqual(
        normalizeReport(parseReport(go.stdout))
      );
    });
  }
});
