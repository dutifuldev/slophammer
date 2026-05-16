import { describe, expect, it } from "vitest";

import { newReport, writeJSON, writeSARIF, writeText } from "../src/report/report.js";
import type { Finding } from "../src/rules/types.js";

describe("report writers", () => {
  it("sorts findings and writes JSON", () => {
    const report = newReport([
      finding("repo.ci-required", "z.yml"),
      finding("repo.agents-required", "a.md")
    ]);

    expect(report.ok).toBe(false);
    expect(report.findings.map((item) => item.rule_id)).toEqual([
      "repo.agents-required",
      "repo.ci-required"
    ]);
    expect(JSON.parse(writeJSON(report))).toEqual(report);
  });

  it("writes compact text reports", () => {
    expect(writeText(newReport([]))).toBe("OK: no findings\n");
    expect(writeText(newReport([finding("repo.readme-required", "README.md")]))).toContain(
      "1 finding(s)"
    );
  });

  it("writes SARIF for findings", () => {
    const content = writeSARIF(newReport([finding("repo.readme-required", "README.md", "warn")]));
    const parsed = JSON.parse(content) as {
      runs: readonly [
        {
          results: readonly [{ level: string; locations: readonly unknown[] }];
          tool: { driver: { rules: readonly [{ id: string }] } };
        }
      ];
    };

    expect(parsed.runs[0].tool.driver.rules[0].id).toBe("repo.readme-required");
    expect(parsed.runs[0].results[0].level).toBe("warning");
    expect(parsed.runs[0].results[0].locations).toHaveLength(1);
  });
});

function finding(
  ruleID: string,
  filePath: string,
  severity: Finding["severity"] = "error"
): Finding {
  return {
    rule_id: ruleID,
    severity,
    path: filePath,
    message: `${ruleID} message`
  };
}
