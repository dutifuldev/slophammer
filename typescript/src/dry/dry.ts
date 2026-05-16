import { findCopiedBlocks } from "./copied-blocks.js";
import { loadSourceFiles } from "./files.js";
import { formatText, groupFindings, writeJSON } from "./report.js";
import type { DryOptions, DryReport } from "./types.js";

export const defaultCopiedBlockTokens = 100;
export const defaultMaxFindings = 0;

export async function findDry(options: DryOptions): Promise<DryReport> {
  const withDefaults = applyDefaults(options);
  const files = await loadSourceFiles(withDefaults.root, withDefaults.paths, withDefaults.exclude);
  const findings = withDefaults.copiedBlockEnabled
    ? findCopiedBlocks(files, withDefaults.copiedBlockTokens)
    : [];
  return { findings, groups: groupFindings(findings) };
}

export async function checkDry(
  options: DryOptions
): Promise<{ readonly code: number; readonly output: string }> {
  const withDefaults = applyDefaults(options);
  const report = await findDry(withDefaults);
  const reportOutput = renderReport(report, withDefaults);
  const countOutput =
    withDefaults.format === "json"
      ? ""
      : `DRY candidates: ${String(report.findings.length)}; maximum: ${String(withDefaults.maxFindings)}\n`;
  return {
    code: report.findings.length > withDefaults.maxFindings ? 1 : 0,
    output: `${reportOutput}${countOutput}`
  };
}

export function applyDefaults(options: DryOptions): DryOptions {
  const copiedBlockEnabled =
    options.copiedBlockSet || options.copiedBlockEnabled ? options.copiedBlockEnabled : true;
  return {
    ...options,
    root: options.root === "" ? "." : options.root,
    paths: options.paths.length === 0 ? ["."] : options.paths,
    maxFindings:
      options.maxFindingsSet || options.maxFindings !== 0
        ? options.maxFindings
        : defaultMaxFindings,
    copiedBlockEnabled,
    copiedBlockTokens:
      options.copiedBlockTokens === 0 ? defaultCopiedBlockTokens : options.copiedBlockTokens
  };
}

function renderReport(report: DryReport, options: DryOptions): string {
  if (options.format === "json" || options.showReport) {
    return writeJSON(report);
  }
  if (options.format === "text") {
    return formatText(report);
  }
  return "";
}
