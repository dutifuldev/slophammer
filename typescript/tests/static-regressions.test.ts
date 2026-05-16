import { describe, expect, it } from "vitest";

import type { Config } from "../src/config/config.js";
import { emptyConfig } from "../src/config/config.js";
import { newSnapshot } from "../src/repo/repo.js";
import { runRules } from "../src/rules/rules.js";

describe("TypeScript static rule regressions", () => {
  it("does not accept npm test wrappers without a real package test runner", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        {
          path: ".github/workflows/ci.yml",
          content: "name: ci\njobs:\n  test:\n    steps:\n      - run: npm test\n"
        },
        packageWithScripts({ test: "echo ok" }),
        enabledESLintConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.test-required");
  });

  it("does not apply Vitest config thresholds to nyc without check coverage", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ coverage: "nyc mocha" }),
        enabledESLintConfig(),
        coverageConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.coverage-required");
  });

  it("does not apply Vitest config thresholds to Jest coverage", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ coverage: "jest --coverage" }),
        enabledESLintConfig(),
        coverageConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.coverage-required");
  });

  it("accepts Jest coverage with Jest config thresholds", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ coverage: "jest --coverage" }),
        enabledESLintConfig(),
        jestCoverageConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).not.toContain("ts.coverage-required");
  });

  it("accepts Vitest coverage thresholds from Vite config", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ coverage: "vitest run --coverage" }),
        enabledESLintConfig(),
        { path: "vite.config.ts", content: coverageConfig().content }
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).not.toContain("ts.coverage-required");
  });

  it("accepts coverage wrappers that forward coverage args to the test script", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ test: "vitest run", coverage: "npm run test -- --coverage" }),
        enabledESLintConfig(),
        coverageConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).not.toContain("ts.coverage-required");
  });
});

describe("TypeScript command failure regressions", () => {
  it("does not accept lint commands whose failures are ignored", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ lint: "eslint . || true" }),
        enabledESLintConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.lint-required");
  });

  it("does not accept DRY commands whose failures are ignored", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ dry: "slophammer typescript dry . || true" }),
        enabledESLintConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.dry-required");
  });

  it("does not accept missing TypeScript mutation command placeholders", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({
          mutate: "slophammer typescript mutate . && echo typescript mutation"
        }),
        enabledESLintConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.mutation-required");
  });

  it("does not accept mutation commands whose failures are ignored", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts({ mutate: "stryker run || true" }),
        enabledESLintConfig()
      ]),
      emptyConfig()
    );

    expect(report.findings.map((finding) => finding.rule_id)).toContain("ts.mutation-required");
  });

  it("checks import type expressions against dependency boundaries", () => {
    const report = runRules(
      newSnapshot("/repo", [
        ...baseTypeScriptFiles(),
        packageWithScripts(),
        enabledESLintConfig(),
        { path: "src/app/index.ts", content: 'type Secret = import("../secret/secret").Secret;\n' },
        { path: "src/secret/secret.ts", content: "export type Secret = string;\n" }
      ]),
      configWithBoundary()
    );

    expect(report.findings).toEqual([
      {
        rule_id: "ts.dependency-boundaries-required",
        severity: "error",
        path: "src/app/index.ts",
        message: "Import src/secret/secret is outside allowed dependencies for src/app"
      }
    ]);
  });
});

function baseTypeScriptFiles(): readonly { readonly path: string; readonly content: string }[] {
  return [
    { path: "README.md", content: "# Repo\n" },
    { path: "AGENTS.md", content: "# Agents\n" },
    { path: ".github/workflows/ci.yml", content: "name: ci\n" },
    {
      path: "tsconfig.json",
      content:
        '{"compilerOptions":{"strict":true,"noImplicitAny":true,"noImplicitOverride":true,"noUncheckedIndexedAccess":true,"exactOptionalPropertyTypes":true,"noFallthroughCasesInSwitch":true,"noPropertyAccessFromIndexSignature":true,"useUnknownInCatchVariables":true,"noEmitOnError":true}}'
    }
  ];
}

function packageWithScripts(overrides: Readonly<Partial<Record<string, string>>> = {}): {
  readonly path: string;
  readonly content: string;
} {
  return {
    path: "package.json",
    content: JSON.stringify({
      scripts: { ...packageScripts(), ...overrides }
    })
  };
}

function packageScripts(): Readonly<Record<string, string>> {
  return {
    format: "prettier --check .",
    lint: "eslint .",
    typecheck: "tsc --noEmit",
    test: "vitest run",
    coverage:
      "vitest run --coverage --coverage.thresholds.lines=85 --coverage.thresholds.functions=85 --coverage.thresholds.branches=85 --coverage.thresholds.statements=85",
    dry: "slophammer typescript dry .",
    mutate: "stryker run"
  };
}

function enabledESLintConfig(): { readonly path: string; readonly content: string } {
  return {
    path: "eslint.config.mjs",
    content:
      'export default [{rules:{"@typescript-eslint/no-explicit-any":"error","@typescript-eslint/no-unsafe-assignment":"error","@typescript-eslint/no-unsafe-call":"error","@typescript-eslint/no-unsafe-member-access":"error","@typescript-eslint/no-unsafe-return":"error",complexity:["error",8]}}];'
  };
}

function coverageConfig(): { readonly path: string; readonly content: string } {
  return {
    path: "vitest.config.ts",
    content:
      "export default {test:{coverage:{thresholds:{lines:85,functions:85,branches:85,statements:85}}}};\n"
  };
}

function jestCoverageConfig(): { readonly path: string; readonly content: string } {
  return {
    path: "jest.config.js",
    content:
      "export default {coverageThreshold:{global:{lines:85,functions:85,branches:85,statements:85}}};\n"
  };
}

function configWithBoundary(): Config {
  const cfg = emptyConfig();
  return {
    ...cfg,
    typescript: {
      ...cfg.typescript,
      dependencyBoundaries: [{ from: "src/app", allow: [] }]
    }
  };
}
