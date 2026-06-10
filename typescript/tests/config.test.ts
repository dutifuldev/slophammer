import { describe, expect, it } from "vitest";

import { loadConfig } from "../src/config/config.js";
import { newSnapshot } from "../src/repo/repo.js";

describe("loadConfig", () => {
  it("rejects weak TypeScript coverage targets", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "typescript:\n  coverage:\n    threshold: 84\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("coverage.threshold");
  });

  it("rejects weak TypeScript complexity targets", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: "typescript:\n  complexity:\n    max: 9\n"
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("complexity.max");
  });

  it("rejects non-numeric TypeScript policy values", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "typescript:",
          "  coverage:",
          '    threshold: "90"',
          "  complexity:",
          '    max: "6"',
          "  dry:",
          '    max_findings: "0"',
          "    copied_blocks:",
          '      min_tokens: "100"',
          ""
        ].join("\n")
      }
    ]);

    expect(() => loadConfig(snapshot)).toThrow("coverage.threshold must be a number");
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

describe("loadConfig strict keys", () => {
  it("rejects unknown config keys", () => {
    const cases = [
      { name: "root", content: "made_up: true\n", want: "root.made_up" },
      {
        name: "rules",
        content: "rules:\n  repo.readme-required:\n    made_up: true\n",
        want: "rules.repo.readme-required.made_up"
      },
      { name: "typescript", content: "typescript:\n  made_up: true\n", want: "typescript.made_up" },
      {
        name: "typescript coverage",
        content: "typescript:\n  coverage:\n    made_up: true\n",
        want: "typescript.coverage.made_up"
      },
      {
        name: "typescript dry",
        content: "typescript:\n  dry:\n    made_up: true\n",
        want: "typescript.dry.made_up"
      },
      {
        name: "typescript copied blocks",
        content: "typescript:\n  dry:\n    copied_blocks:\n      made_up: true\n",
        want: "typescript.dry.copied_blocks.made_up"
      },
      {
        name: "typescript boundary",
        content:
          "typescript:\n  dependency_boundaries:\n    - from: src/app\n      made_up: true\n",
        want: "typescript.dependency_boundaries[0].made_up"
      },
      { name: "ignored go", content: "go:\n  made_up: true\n", want: "go.made_up" },
      {
        name: "ignored go mutation",
        content: "go:\n  mutation:\n    made_up: true\n",
        want: "go.mutation.made_up"
      },
      {
        name: "removed go mutation targets",
        content: "go:\n  mutation_targets:\n    - main.go\n",
        want: "go.mutation_targets"
      },
      {
        name: "removed go coverage_threshold",
        content: "go:\n  coverage_threshold: 85\n",
        want: "go.coverage_threshold"
      },
      {
        name: "removed typescript coverage_threshold",
        content: "typescript:\n  coverage_threshold: 85\n",
        want: "typescript.coverage_threshold"
      },
      {
        name: "removed typescript complexity_max",
        content: "typescript:\n  complexity_max: 8\n",
        want: "typescript.complexity_max"
      },
      {
        name: "removed typescript mutation_targets",
        content: "typescript:\n  mutation_targets:\n    - src/rules.ts\n",
        want: "typescript.mutation_targets"
      },
      {
        name: "removed rust coverage_threshold",
        content: "rust:\n  coverage_threshold: 85\n",
        want: "rust.coverage_threshold"
      },
      { name: "ignored rust", content: "rust:\n  made_up: true\n", want: "rust.made_up" },
      {
        name: "ignored rust dry",
        content: "rust:\n  dry:\n    made_up: true\n",
        want: "rust.dry.made_up"
      },
      {
        name: "ignored rust unsafe allow",
        content: "rust:\n  unsafe:\n    allow:\n      - path: src/lib.rs\n        made_up: true\n",
        want: "rust.unsafe.allow[0].made_up"
      }
    ];

    for (const testCase of cases) {
      const snapshot = newSnapshot("/repo", [
        {
          path: "slophammer.yml",
          content: testCase.content
        }
      ]);

      expect(() => loadConfig(snapshot), testCase.name).toThrow(testCase.want);
    }
  });
});

describe("loadConfig shared language sections", () => {
  it("allows shared Go, TypeScript, and Rust config", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "slophammer.yml",
        content: [
          "go:",
          "  coverage:",
          "    threshold: 85",
          "    profile: coverage.out",
          "  targets:",
          "    - go",
          "  exclude:",
          "    - fixtures/**",
          "  dry:",
          "    structural:",
          "      enabled: true",
          "      threshold: 0.82",
          "  crap:",
          "    max_score: 8",
          "  mutation:",
          "    targets:",
          "      - go/internal/rules",
          "    exclude:",
          "      - go/internal/rules/generated/**",
          "typescript:",
          "  coverage:",
          "    threshold: 85",
          "  complexity:",
          "    max: 8",
          "  dry:",
          "    copied_blocks:",
          "      enabled: true",
          "      min_tokens: 100",
          "rust:",
          "  coverage:",
          "    threshold: 85",
          "    paths:",
          "      - rust/crates",
          "  complexity:",
          "    cognitive_max: 8",
          "  targets:",
          "    - rust/crates",
          "  exclude:",
          "    - rust/target/**",
          "  dry:",
          "    max_findings: 0",
          "    paths:",
          "      - rust/crates",
          "    copied_blocks:",
          "      enabled: true",
          "      min_tokens: 100",
          "  unsafe:",
          "    policy: forbid",
          "    allow:",
          "      - path: src/lib.rs",
          "        reason: reviewed",
          "  mutation:",
          "    targets:",
          "      - rust/crates/slophammer-cli/src/rust_rules",
          "  dependency_boundaries:",
          "    - from: rust/crates/slophammer-cli",
          "      allow: []",
          ""
        ].join("\n")
      }
    ]);

    expect(loadConfig(snapshot).typescript.coverage.threshold).toBe(85);
  });
});

describe("loadConfig config discovery", () => {
  it("ignores nested configs when the root config is missing", () => {
    const snapshot = newSnapshot("/repo", [
      {
        path: "fixtures/repos/example/slophammer.yml",
        content: "typescript:\n  coverage:\n    threshold: 84\n"
      }
    ]);

    expect(loadConfig(snapshot).typescript.coverage.threshold).toBe(0);
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
        content: "typescript:\n  coverage:\n    threshold: 84\n"
      },
      {
        path: "slophammer.yml",
        content: "typescript:\n  coverage:\n    threshold: 85\n"
      }
    ]);

    expect(loadConfig(snapshot).typescript.coverage.threshold).toBe(85);
  });
});
