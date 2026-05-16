import path from "node:path";
import ts from "typescript";

export function compilerConfig(
  filePath: string,
  content: string
): Readonly<Record<string, unknown>> {
  const parsed = ts.parseConfigFileTextToJson(filePath, content);
  return asRecord(parsed.config as unknown);
}

export function extendedConfigPaths(
  filePath: string,
  config: Readonly<Record<string, unknown>>
): readonly string[] {
  const value = config["extends"];
  const entries =
    typeof value === "string"
      ? [value]
      : Array.isArray(value)
        ? value.filter((item): item is string => typeof item === "string")
        : [];
  return entries.flatMap((entry) => resolveExtendedConfigPath(filePath, entry));
}

function resolveExtendedConfigPath(filePath: string, entry: string): readonly string[] {
  if (!entry.startsWith(".")) {
    return [];
  }
  const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(filePath), entry));
  return path.posix.extname(resolved) === "" ? [`${resolved}.json`] : [resolved];
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}
