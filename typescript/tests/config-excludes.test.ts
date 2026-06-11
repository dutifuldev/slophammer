import { describe, expect, it } from "vitest";

import { conventionalExcludePattern, loadConfig } from "../src/config/config.js";
import { newSnapshot, type Snapshot } from "../src/repo/repo.js";

describe("loadConfig exclude entries", () => {
  it("accepts conventional string excludes", () => {
    const cfg = loadConfig(
      configSnapshot([
        "typescript:",
        "  dry:",
        "    exclude:",
        '      - "**/*.test.ts"',
        '      - "fixtures/**"',
        "  coverage:",
        "    exclude:",
        '      - "dist/**"',
        ""
      ])
    );

    expect(cfg.typescript.dry.exclude).toEqual(["**/*.test.ts", "fixtures/**"]);
    expect(cfg.typescript.coverage.exclude).toEqual(["dist/**"]);
  });

  it("rejects production string excludes without a reason", () => {
    const snapshot = configSnapshot([
      "typescript:",
      "  dry:",
      "    exclude:",
      '      - "src/vendored/**"',
      ""
    ]);

    expect(() => loadConfig(snapshot)).toThrow(
      "typescript.dry.exclude requires a reason for production paths"
    );
  });

  it("rejects production coverage excludes without a reason", () => {
    const snapshot = configSnapshot([
      "typescript:",
      "  coverage:",
      "    exclude:",
      '      - "src/vendored/**"',
      ""
    ]);

    expect(() => loadConfig(snapshot)).toThrow(
      "typescript.coverage.exclude requires a reason for production paths"
    );
  });

  it("accepts reasoned excludes and keeps their patterns flowing", () => {
    const cfg = loadConfig(
      configSnapshot([
        "typescript:",
        "  dry:",
        "    exclude:",
        '      - pattern: "src/vendored/**"',
        "        reason: vendored upstream code, synced verbatim",
        ""
      ])
    );

    expect(cfg.typescript.dry.exclude).toEqual(["src/vendored/**"]);
  });

  it("rejects reasoned excludes with empty reasons", () => {
    const snapshot = configSnapshot([
      "typescript:",
      "  dry:",
      "    exclude:",
      '      - pattern: "src/vendored/**"',
      '        reason: "  "',
      ""
    ]);

    expect(() => loadConfig(snapshot)).toThrow("typescript.dry.exclude reasons must not be empty");
  });

  it("rejects unknown keys and missing patterns on the object form", () => {
    expect(() =>
      loadConfig(
        configSnapshot([
          "typescript:",
          "  dry:",
          "    exclude:",
          '      - pattern: "src/vendored/**"',
          "        reason: vendored",
          "        why: extra",
          ""
        ])
      )
    ).toThrow("typescript.dry.exclude[0].why is not supported");
    expect(() =>
      loadConfig(
        configSnapshot(["typescript:", "  dry:", "    exclude:", "      - reason: vendored", ""])
      )
    ).toThrow("typescript.dry.exclude[0].pattern must be a string");
  });

  it("rejects non-list excludes and non-entry items", () => {
    expect(() =>
      loadConfig(configSnapshot(["typescript:", "  dry:", "    exclude: src", ""]))
    ).toThrow("typescript.dry.exclude must be a list");
    expect(() =>
      loadConfig(configSnapshot(["typescript:", "  dry:", "    exclude:", "      - 5", ""]))
    ).toThrow("typescript.dry.exclude[0] must be an object");
  });

  it("classifies conventional exclude patterns", () => {
    expect(conventionalExcludePattern("**/*_test.go")).toBe(true);
    expect(conventionalExcludePattern("node_modules/**")).toBe(true);
    expect(conventionalExcludePattern("src/generated_parser.ts")).toBe(true);
    expect(conventionalExcludePattern("src/core/**")).toBe(false);
  });
});

describe("loadConfig cross-language exclude entries", () => {
  it("accepts the object form in cross-language sections", () => {
    const snapshot = configSnapshot([
      "go:",
      "  dry:",
      "    exclude:",
      '      - pattern: "go/internal/vendored/**"',
      "        reason: vendored",
      "rust:",
      "  coverage:",
      "    threshold: 85",
      "    exclude:",
      '      - pattern: "rust/legacy/**"',
      "        reason: scheduled for deletion",
      "  mutation:",
      "    exclude:",
      '      - "rust/fixtures/**"',
      ""
    ]);

    expect(() => loadConfig(snapshot)).not.toThrow();
  });

  it("rejects malformed exclude entries in cross-language sections", () => {
    expect(() =>
      loadConfig(
        configSnapshot([
          "go:",
          "  exclude:",
          '      - pattern: "go/vendored/**"',
          "        why: extra",
          ""
        ])
      )
    ).toThrow("go.exclude[0].why is not supported");
    expect(() => loadConfig(configSnapshot(["rust:", "  dry:", "    exclude: nope", ""]))).toThrow(
      "rust.dry.exclude must be a list"
    );
  });
});

function configSnapshot(lines: readonly string[]): Snapshot {
  return newSnapshot("/repo", [{ path: "slophammer.yml", content: lines.join("\n") }]);
}
