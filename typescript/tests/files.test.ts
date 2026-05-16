import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { loadSourceFiles } from "../src/dry/files.js";

describe("loadSourceFiles", () => {
  it("loads source files from directories and explicit file targets", async () => {
    const root = await sourceFixture();

    const files = await loadSourceFiles(root, [".", "src/a.ts"], ["**/*.test.ts"]);

    expect(files.map((file) => file.path)).toEqual([
      "src/a.cjs",
      "src/a.cts",
      "src/a.mjs",
      "src/a.mts",
      "src/a.ts",
      "src/helper.jsx"
    ]);
  });

  it("supports directory, suffix, wildcard, and exact excludes", async () => {
    const root = await sourceFixture();

    await expect(loadSourceFiles(root, ["."], ["src/**"])).resolves.toEqual([]);
    await expect(loadSourceFiles(root, ["."], ["**/*.test.ts"])).resolves.not.toContainEqual(
      expect.objectContaining({ path: "src/a.test.ts" })
    );
    await expect(loadSourceFiles(root, ["."], ["src/*helper*"])).resolves.not.toContainEqual(
      expect.objectContaining({ path: "src/helper.jsx" })
    );
    await expect(loadSourceFiles(root, ["."], ["src/a.ts"])).resolves.not.toContainEqual(
      expect.objectContaining({ path: "src/a.ts" })
    );
  });

  it("keeps directory glob excludes segment bounded", async () => {
    const root = await sourceFixture();
    await mkdir(path.join(root, "test"), { recursive: true });
    await mkdir(path.join(root, "testing"), { recursive: true });
    await mkdir(path.join(root, "dist-utils"), { recursive: true });
    await writeFile(path.join(root, "test", "helper.ts"), "export const helper = 1;\n");
    await writeFile(path.join(root, "testing", "parser.ts"), "export const parser = 1;\n");
    await writeFile(path.join(root, "dist-utils", "build.ts"), "export const build = 1;\n");

    const files = await loadSourceFiles(root, ["."], ["test/**", "dist/**"]);

    expect(files).toContainEqual(expect.objectContaining({ path: "testing/parser.ts" }));
    expect(files).toContainEqual(expect.objectContaining({ path: "dist-utils/build.ts" }));
    expect(files).not.toContainEqual(expect.objectContaining({ path: "test/helper.ts" }));
  });
});

async function sourceFixture(): Promise<string> {
  const root = await mkdtemp(path.join(tmpdir(), "slophammer-files-"));
  await mkdir(path.join(root, "src"), { recursive: true });
  await mkdir(path.join(root, "node_modules", "ignored"), { recursive: true });
  await mkdir(path.join(root, "dist"), { recursive: true });
  await writeFile(path.join(root, "src", "a.ts"), "export const a = 1;\n");
  await writeFile(path.join(root, "src", "a.mts"), "export const a = 1;\n");
  await writeFile(path.join(root, "src", "a.cts"), "export const a = 1;\n");
  await writeFile(path.join(root, "src", "a.mjs"), "export const a = 1;\n");
  await writeFile(path.join(root, "src", "a.cjs"), "exports.a = 1;\n");
  await writeFile(path.join(root, "src", "a.test.ts"), "export const test = 1;\n");
  await writeFile(path.join(root, "src", "helper.jsx"), "export const helper = 1;\n");
  await writeFile(path.join(root, "src", "notes.txt"), "not source\n");
  await writeFile(path.join(root, "node_modules", "ignored", "b.ts"), "export const b = 1;\n");
  await writeFile(path.join(root, "dist", "c.ts"), "export const c = 1;\n");
  return root;
}
