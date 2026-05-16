import { readFile } from "node:fs/promises";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { defaultDefinitions } from "../src/rules/definitions.js";
import { repoRoot } from "./helpers.js";

type RuleSpec = {
  readonly id: string;
  readonly title: string;
  readonly category: string;
  readonly severity: string;
  readonly path: string;
  readonly message: string;
  readonly description: string;
  readonly tool?: string;
  readonly status: string;
};

describe("TypeScript rule metadata", () => {
  it("matches the shared rule spec", async () => {
    const spec = await sharedRuleSpec();

    expect(defaultDefinitions).toEqual(
      spec.filter((rule) => rule.category === "repo" || rule.category === "typescript")
    );
  });
});

async function sharedRuleSpec(): Promise<readonly RuleSpec[]> {
  const content = await readFile(path.join(repoRoot, "specs", "rules.json"), "utf8");
  const parsed: unknown = JSON.parse(content);
  const rules = asRecord(parsed)["rules"];
  if (!Array.isArray(rules)) {
    throw new Error("specs/rules.json must contain rules");
  }
  return rules.map(asRuleSpec);
}

function asRuleSpec(value: unknown): RuleSpec {
  const root = asRecord(value);
  return {
    id: asString(root["id"]),
    title: asString(root["title"]),
    category: asString(root["category"]),
    severity: asString(root["severity"]),
    path: asString(root["path"]),
    message: asString(root["message"]),
    description: asString(root["description"]),
    ...(typeof root["tool"] === "string" ? { tool: root["tool"] } : {}),
    status: asString(root["status"])
  };
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Readonly<Record<string, unknown>>;
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}
