import { check, explain, exitError, exitOK, typescriptDry, type CheckOptions } from "../app/app.js";
import type { DryOptions } from "../dry/types.js";

type Result = {
  readonly code: number;
  readonly stdout: string;
  readonly stderr: string;
};

export async function run(args: readonly string[]): Promise<Result> {
  try {
    return await dispatch(args);
  } catch (error) {
    return { code: exitError, stdout: "", stderr: `${errorMessage(error)}\n` };
  }
}

async function dispatch(args: readonly string[]): Promise<Result> {
  const command = args[0];
  if (command === undefined || command === "-h" || command === "--help" || command === "help") {
    return { code: exitOK, stdout: usage(), stderr: "" };
  }
  if (command === "check") {
    return await check(parseCheckArgs(args.slice(1)));
  }
  if (command === "explain") {
    return runExplain(args.slice(1));
  }
  if (command === "typescript") {
    return await runTypeScript(args.slice(1));
  }
  return { code: exitError, stdout: "", stderr: `unknown command: ${command}\n${usage()}` };
}

function runExplain(args: readonly string[]): Result {
  const ruleID = args[0];
  if (ruleID === undefined || args.length !== 1) {
    return { code: exitError, stdout: "", stderr: "usage: slophammer explain <rule-id>\n" };
  }
  return explain(ruleID);
}

function runTypeScript(args: readonly string[]): Promise<Result> | Result {
  const subcommand = args[0];
  if (subcommand === "dry") {
    return typescriptDry(parseDryArgs(args.slice(1)));
  }
  return {
    code: exitError,
    stdout: "",
    stderr: `unknown typescript command: ${subcommand ?? ""}\n${typescriptUsage()}`
  };
}

function parseCheckArgs(args: readonly string[]): CheckOptions {
  let options: CheckOptions = { root: "", format: "text", execute: false };
  for (let index = 0; index < args.length; index++) {
    const parsed = parseCheckArg(options, args, index);
    options = parsed.options;
    index += parsed.advance;
  }
  if (options.root === "") {
    throw new Error("usage: slophammer check <path> [--format text|json|sarif] [--execute]");
  }
  return options;
}

function parseCheckArg(
  options: CheckOptions,
  args: readonly string[],
  index: number
): { readonly options: CheckOptions; readonly advance: number } {
  const arg = args[index];
  if (arg === "--format") {
    return {
      options: { ...options, format: parseFormat(nextValue(args, index, "--format")) },
      advance: 1
    };
  }
  if (arg === "--json") {
    return { options: { ...options, format: "json" }, advance: 0 };
  }
  if (arg === "--execute") {
    return { options: { ...options, execute: true }, advance: 0 };
  }
  if (arg?.startsWith("-")) {
    throw new Error(`unknown check option: ${arg}`);
  }
  if (arg !== undefined) {
    if (options.root !== "") {
      throw new Error("check accepts exactly one path");
    }
    return { options: { ...options, root: arg }, advance: 0 };
  }
  return { options, advance: 0 };
}

function parseDryArgs(args: readonly string[]): DryOptions {
  let options = defaultDryOptions();
  for (let index = 0; index < args.length; index++) {
    const parsed = parseDryArg(options, args, index);
    options = parsed.options;
    index += parsed.advance;
  }
  return options;
}

function parseDryArg(
  options: DryOptions,
  args: readonly string[],
  index: number
): { readonly options: DryOptions; readonly advance: number } {
  const arg = args[index];
  if (arg === "--show-report") {
    return { options: { ...options, showReport: true }, advance: 0 };
  }
  if (arg === "--max-findings") {
    return {
      options: {
        ...options,
        maxFindings: parseNonNegativeInteger(nextValue(args, index, "--max-findings")),
        maxFindingsSet: true
      },
      advance: 1
    };
  }
  if (arg === "--format") {
    return {
      options: { ...options, format: parseDryFormat(nextValue(args, index, "--format")) },
      advance: 1
    };
  }
  if (arg?.startsWith("-")) {
    throw new Error(`unknown typescript dry option: ${arg}`);
  }
  return arg === undefined
    ? { options, advance: 0 }
    : { options: { ...options, root: arg }, advance: 0 };
}

function defaultDryOptions(): DryOptions {
  return {
    root: ".",
    paths: ["."],
    exclude: [
      "test/**",
      "tests/**",
      "**/test/**",
      "**/tests/**",
      "**/*.test.cts",
      "**/*.test.js",
      "**/*.test.jsx",
      "**/*.test.mts",
      "**/*.test.ts",
      "**/*.test.tsx",
      "**/*.spec.cts",
      "**/*.spec.js",
      "**/*.spec.jsx",
      "**/*.spec.mts",
      "**/*.spec.ts",
      "**/*.spec.tsx",
      ".stryker-tmp/**",
      "fixtures/**",
      "dist/**",
      "coverage/**"
    ],
    maxFindings: 0,
    maxFindingsSet: false,
    copiedBlockEnabled: false,
    copiedBlockSet: false,
    copiedBlockTokens: 0,
    showReport: false,
    format: ""
  };
}

function parseFormat(value: string): CheckOptions["format"] {
  if (value === "text" || value === "json" || value === "sarif") {
    return value;
  }
  throw new Error(`unsupported format: ${value}`);
}

function parseDryFormat(value: string): DryOptions["format"] {
  if (value === "text" || value === "json") {
    return value;
  }
  throw new Error(`unsupported typescript dry format: ${value}`);
}

function parseNonNegativeInteger(value: string): number {
  if (!/^(?:0|[1-9]\d*)$/u.test(value)) {
    throw new Error("--max-findings must be a non-negative integer");
  }
  const parsed = Number.parseInt(value, 10);
  if (!Number.isInteger(parsed) || parsed < 0) {
    throw new Error("--max-findings must be a non-negative integer");
  }
  return parsed;
}

function nextValue(args: readonly string[], index: number, flag: string): string {
  const value = args[index + 1];
  if (value === undefined) {
    throw new Error(`${flag} requires a value`);
  }
  return value;
}

function usage(): string {
  return [
    "usage:",
    "  slophammer check <path> [--format text|json|sarif] [--execute]",
    "  slophammer explain <rule-id>",
    "  slophammer typescript dry <path>"
  ].join("\n");
}

function typescriptUsage(): string {
  return "usage: slophammer typescript dry <path> [--max-findings n] [--show-report] [--format json|text]\n";
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
