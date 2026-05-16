import { describe, expect, it } from "vitest";

import { loadConfig } from "../src/config/config.js";
import { newSnapshot } from "../src/repo/repo.js";

describe("loadConfig", () => {
  it("rejects weak TypeScript coverage targets", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "typescript:\n  coverage_threshold: 84\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("coverage_threshold");
  });

  it("rejects weak TypeScript complexity targets", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "typescript:\n  complexity_max: 9\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("complexity_max");
  });

  it("rejects non-numeric TypeScript policy values", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "typescript:",
          '  coverage_threshold: "90"',
          '  complexity_max: "6"',
          "  dry:",
          '    max_findings: "0"',
          "    copied_blocks:",
          '      min_tokens: "100"',
          ""
        ].join("\n")
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("coverage_threshold must be a number");
  });

  it("rejects non-boolean TypeScript copied-block toggles", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: ["typescript:", "  dry:", "    copied_blocks:", '      enabled: "true"', ""].join(
          "\n"
        )
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("copied_blocks.enabled must be a boolean");
  });

  it("rejects non-string TypeScript config array entries", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "typescript:",
          "  dry:",
          "    paths:",
          "      - src",
          "      - 123",
          "  dependency_boundaries:",
          "    - from: src/app",
          "      allow:",
          "        - src/core",
          "        - false",
          ""
        ].join("\n")
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("typescript.dry.paths[1] must be a string");
  });

  it("rejects malformed TypeScript dependency boundaries", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: ["typescript:", "  dependency_boundaries:", "    - 123", ""].join("\n")
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow(
      "typescript.dependency_boundaries[0] must be an object"
    );
  });

  it("rejects non-string TypeScript dependency boundary allow entries", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "typescript:",
          "  dependency_boundaries:",
          "    - from: src/app",
          "      allow:",
          "        - src/core",
          "        - false",
          ""
        ].join("\n")
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow(
      "typescript.dependency_boundaries[0].allow[1] must be a string"
    );
  });

  it("rejects fractional TypeScript DRY count values", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "typescript:",
          "  dry:",
          "    max_findings: 0.5",
          "    copied_blocks:",
          "      min_tokens: 10.5",
          ""
        ].join("\n")
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("max_findings must be a non-negative integer");
  });
});

describe("loadConfig config discovery", () => {
  it("ignores nested configs when the root config is missing", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "fixtures/repos/example/slophammer.yml",
        content: "typescript:\n  coverage_threshold: 84\n"
      }
    ]);

    expect(loadConfig(snapshot).typescript.coverageThreshold).toBe(0);
  });
});

describe("loadConfig shared rules", () => {
  it("rejects invalid rule severity values", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "rules:\n  repo.readme-required:\n    severity: notice\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("severity");
  });

  it("rejects disabled rules without a reason", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "rules:\n  repo.readme-required:\n    disabled: true\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("reason");
  });

  it("prefers the root config over fixture configs", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "fixtures/repos/example/slophammer.yml",
        content: "typescript:\n  coverage_threshold: 84\n"
      },
      {
        path: "slophammer.yml",
        content: "typescript:\n  coverage_threshold: 85\n"
      }
    ]);

    expect(loadConfig(snapshot).typescript.coverageThreshold).toBe(85);
  });
});
