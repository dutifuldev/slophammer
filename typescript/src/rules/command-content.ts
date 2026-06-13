export function commandSegments(content: string): readonly string[] {
  return content
    .replaceAll("\\\n", " ")
    .split(/\n|&&|;/u)
    .map((segment) => normalizeCommandContent(segment).trim())
    .filter((segment) => !segment.includes("||"))
    .filter((segment) => segment.length > 0);
}

export function normalizeCommandContent(content: string): string {
  return content.replaceAll("\\\n", " ").replace(/\s+/g, " ").toLowerCase();
}

export function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}
