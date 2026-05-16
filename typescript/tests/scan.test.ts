import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { scanRepo } from "../src/scan/scan.js";

describe("scanRepo", () => {
  it("skips generated Stryker temp directories", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "slophammer-scan-"));
    await mkdir(path.join(root, ".stryker-tmp", "src"), { recursive: true });
    await mkdir(path.join(root, "src"), { recursive: true });
    await writeFile(path.join(root, "src", "index.ts"), "export const value = 1;\n");
    await writeFile(
      path.join(root, ".stryker-tmp", "src", "mutant.ts"),
      "export const mutant = 1;\n"
    );

    const snapshot = await scanRepo(root);

    expect([...snapshot.files.keys()]).toEqual(["src/index.ts"]);
  });
});
