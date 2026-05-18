import { describe, expect, it } from "vitest";

import { loadConfig } from "../src/config/config.js";
import { newSnapshot } from "../src/repo/repo.js";

describe("loadConfig TypeScript coverage", () => {
  it("loads scoped TypeScript coverage config", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "typescript:",
          "  coverage:",
          "    threshold: 90",
          "    paths:",
          "      - src/runtime",
          "    exclude:",
          "      - dist/**",
          ""
        ].join("\n")
      }
    ]);

    expect(loadConfig(snapshot).typescript.coverage).toEqual({
      threshold: 90,
      paths: ["src/runtime"],
      exclude: ["dist/**"]
    });
  });

  it("rejects weak scoped TypeScript coverage targets", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "typescript:\n  coverage:\n    threshold: 84\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("coverage.threshold");
  });
});
