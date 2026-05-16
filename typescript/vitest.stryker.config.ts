import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: [
      "tests/dependency-boundaries.test.ts",
      "tests/project-detection.test.ts",
      "tests/rules.test.ts",
      "tests/static-regressions.test.ts"
    ]
  }
});
