import path from "node:path";

// typeScriptSourcePath reports whether a path is production TypeScript
// source: a TypeScript extension that is not a declaration file, a test
// file, a tooling config, or ignored project data.
export function typeScriptSourcePath(filePath: string): boolean {
  return (
    !ignoredProjectDataPath(filePath) &&
    !testSourcePath(filePath) &&
    !typeScriptDeclarationPath(filePath) &&
    !typeScriptToolingConfigPath(filePath) &&
    typeScriptSourceExtension(filePath)
  );
}

export function typeScriptSourceExtension(filePath: string): boolean {
  return (
    filePath.endsWith(".ts") ||
    filePath.endsWith(".tsx") ||
    filePath.endsWith(".mts") ||
    filePath.endsWith(".cts")
  );
}

export function typeScriptDeclarationPath(filePath: string): boolean {
  return filePath.endsWith(".d.ts") || filePath.endsWith(".d.mts") || filePath.endsWith(".d.cts");
}

export function testSourcePath(filePath: string): boolean {
  const baseName = path.posix.basename(filePath);
  return (
    filePath.startsWith("test/") ||
    filePath.startsWith("tests/") ||
    filePath.includes("/test/") ||
    filePath.includes("/tests/") ||
    baseName.includes(".test.") ||
    baseName.includes(".spec.")
  );
}

export function typeScriptToolingConfigPath(filePath: string): boolean {
  return path.posix.basename(filePath).includes(".config.");
}

export function ignoredProjectDataPath(filePath: string): boolean {
  return filePath.startsWith("fixtures/") || filePath.startsWith("templates/");
}
