import type { Definition } from "./types.js";

export const ruleIDs = {
  readmeRequired: "repo.readme-required",
  agentsRequired: "repo.agents-required",
  ciRequired: "repo.ci-required",
  tsPackageRequired: "ts.package-required",
  tsTypecheckRequired: "ts.typecheck-required",
  tsStrictRequired: "ts.strict-required",
  tsNoExplicitAny: "ts.no-explicit-any",
  tsNoUnsafeTypes: "ts.no-unsafe-types",
  tsLintRequired: "ts.lint-required",
  tsFormatRequired: "ts.format-required",
  tsTestRequired: "ts.test-required",
  tsCoverageRequired: "ts.coverage-required",
  tsComplexityRequired: "ts.complexity-required",
  tsDryRequired: "ts.dry-required",
  tsMutationRequired: "ts.mutation-required",
  tsDependencyBoundariesRequired: "ts.dependency-boundaries-required"
} as const;

export const defaultDefinitions: readonly Definition[] = [
  {
    id: ruleIDs.readmeRequired,
    title: "README required",
    category: "repo",
    severity: "error",
    path: "README.md",
    message: "README.md is required",
    description: "The target repo should have a README.md.",
    status: "implemented"
  },
  {
    id: ruleIDs.agentsRequired,
    title: "Agent instructions required",
    category: "repo",
    severity: "error",
    path: "AGENTS.md",
    message: "AGENTS.md is required",
    description: "The target repo should have an AGENTS.md.",
    status: "implemented"
  },
  {
    id: ruleIDs.ciRequired,
    title: "CI workflow required",
    category: "repo",
    severity: "error",
    path: ".github/workflows",
    message: ".github/workflows must contain at least one .yml or .yaml workflow",
    description: "The target repo should have a CI workflow under .github/workflows.",
    status: "implemented"
  },
  {
    id: ruleIDs.tsPackageRequired,
    title: "TypeScript package required",
    category: "typescript",
    severity: "error",
    path: "package.json",
    message: "TypeScript projects must include package.json",
    description: "TypeScript projects should include package.json.",
    status: "implemented"
  },
  {
    id: ruleIDs.tsTypecheckRequired,
    title: "TypeScript typecheck required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare tsc --noEmit in CI or scripts",
    description: "TypeScript projects should declare tsc --noEmit in CI or scripts.",
    tool: "tsc --noEmit",
    status: "implemented"
  },
  {
    id: ruleIDs.tsStrictRequired,
    title: "TypeScript strict mode required",
    category: "typescript",
    severity: "error",
    path: "tsconfig.json",
    message: "TypeScript projects must enable strict mode",
    description: "TypeScript projects should enable strict compiler settings.",
    tool: "tsc",
    status: "implemented"
  },
  {
    id: ruleIDs.tsNoExplicitAny,
    title: "No explicit any",
    category: "typescript",
    severity: "error",
    path: "eslint.config.mjs",
    message: "TypeScript projects must reject explicit any",
    description: "TypeScript projects should configure ESLint to reject explicit any.",
    tool: "eslint",
    status: "implemented"
  },
  {
    id: ruleIDs.tsNoUnsafeTypes,
    title: "No unsafe type operations",
    category: "typescript",
    severity: "error",
    path: "eslint.config.mjs",
    message: "TypeScript projects must reject unsafe type operations",
    description:
      "TypeScript projects should configure ESLint to reject unsafe assignments, calls, member access, and returns.",
    tool: "eslint",
    status: "implemented"
  },
  {
    id: ruleIDs.tsLintRequired,
    title: "TypeScript lint required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare ESLint in CI or scripts",
    description: "TypeScript projects should declare ESLint in CI, scripts, or package.json.",
    tool: "eslint",
    status: "implemented"
  },
  {
    id: ruleIDs.tsFormatRequired,
    title: "TypeScript formatter required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare a formatter check",
    description: "TypeScript projects should declare a formatter check, normally Prettier.",
    tool: "prettier",
    status: "implemented"
  },
  {
    id: ruleIDs.tsTestRequired,
    title: "TypeScript tests required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare tests in CI or scripts",
    description: "TypeScript projects should declare a test command.",
    tool: "vitest",
    status: "implemented"
  },
  {
    id: ruleIDs.tsCoverageRequired,
    title: "TypeScript coverage gate required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare a coverage gate",
    description: "TypeScript projects should declare a coverage gate with a target of at least 85.",
    tool: "vitest coverage",
    status: "implemented"
  },
  {
    id: ruleIDs.tsComplexityRequired,
    title: "TypeScript complexity required",
    category: "typescript",
    severity: "error",
    path: "eslint.config.mjs",
    message: "TypeScript projects must enforce complexity limits",
    description: "TypeScript projects should enforce complexity limits through ESLint.",
    tool: "eslint",
    status: "implemented"
  },
  {
    id: ruleIDs.tsDryRequired,
    title: "TypeScript DRY check required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare a DRY check",
    description:
      "TypeScript projects should declare Slophammer's native copied-block duplicate detector.",
    tool: "slophammer-ts dry",
    status: "implemented"
  },
  {
    id: ruleIDs.tsMutationRequired,
    title: "TypeScript mutation check required",
    category: "typescript",
    severity: "error",
    path: ".github/workflows",
    message: "TypeScript projects must declare mutation testing",
    description:
      "TypeScript projects should declare TypeScript mutation testing, normally through StrykerJS.",
    tool: "stryker",
    status: "implemented"
  },
  {
    id: ruleIDs.tsDependencyBoundariesRequired,
    title: "TypeScript dependency boundaries required",
    category: "typescript",
    severity: "error",
    path: "slophammer.yml",
    message: "TypeScript projects must respect configured dependency boundaries",
    description:
      "TypeScript projects should declare dependency boundaries in slophammer.yml and keep imports inside them.",
    status: "implemented"
  }
];
