import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["tests/**/*.test.ts"],
    exclude: ["**/.stryker-tmp/**", "dist/**", "node_modules/**"],
    coverage: {
      provider: "v8",
      reporter: ["text"],
      exclude: [
        ".stryker-tmp/**",
        "dist/**",
        "coverage/**",
        "tests/**",
        "eslint.config.mjs",
        "vitest.config.ts",
        "vitest.stryker.config.ts",
        "src/cli/main.ts",
        "src/**/types.ts"
      ],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85
      }
    }
  }
});
