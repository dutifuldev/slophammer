import type { Finding, Report, Severity } from "../rules/types.js";

export function newReport(findings: readonly Finding[]): Report {
  const sorted = [...findings].sort((left, right) => {
    const byRule = left.rule_id.localeCompare(right.rule_id);
    if (byRule !== 0) {
      return byRule;
    }
    return left.path.localeCompare(right.path);
  });
  return { ok: sorted.length === 0, findings: sorted };
}

export function writeJSON(report: Report): string {
  return `${JSON.stringify(report, null, 2)}\n`;
}

export function writeText(report: Report): string {
  return `${reportBody(report)}${scopeLine(report)}`;
}

function reportBody(report: Report): string {
  if (report.ok) {
    return "OK: no findings\n";
  }
  const lines = report.findings.map(
    (finding) => `${finding.severity} ${finding.rule_id} ${finding.path}: ${finding.message}`
  );
  return `${lines.join("\n")}\n\n${String(report.findings.length)} finding(s)\n`;
}

function scopeLine(report: Report): string {
  if (report.scope === undefined) {
    return "";
  }
  const { scanned, production_files: productionFiles } = report.scope;
  return `scope: scanned ${String(scanned)} of ${String(productionFiles)} production files\n`;
}

export function writeSARIF(report: Report): string {
  return `${JSON.stringify(sarifReport(report), null, 2)}\n`;
}

type SarifLog = {
  readonly $schema: string;
  readonly version: "2.1.0";
  readonly runs: readonly SarifRun[];
};

type SarifRun = {
  readonly tool: { readonly driver: SarifDriver };
  readonly results: readonly SarifResult[];
};

type SarifDriver = {
  readonly name: "slophammer";
  readonly rules?: readonly SarifRule[];
};

type SarifRule = {
  readonly id: string;
  readonly shortDescription: { readonly text: string };
};

type SarifResult = {
  readonly ruleId: string;
  readonly level: string;
  readonly message: { readonly text: string };
  readonly locations?: readonly SarifLocation[] | undefined;
  readonly suppressions?: readonly SarifSuppression[];
};

type SarifSuppression = {
  readonly kind: "external";
};

type SarifLocation = {
  readonly physicalLocation: {
    readonly artifactLocation: { readonly uri: string };
    readonly region: { readonly startLine: 1 };
  };
};

function sarifReport(report: Report): SarifLog {
  return {
    $schema: "https://json.schemastore.org/sarif-2.1.0.json",
    version: "2.1.0",
    runs: [
      {
        tool: { driver: { name: "slophammer", rules: sarifRules(report.findings) } },
        results: sarifResults(report.findings)
      }
    ]
  };
}

function sarifRules(findings: readonly Finding[]): readonly SarifRule[] {
  const seen = new Set<string>();
  const rules: SarifRule[] = [];
  for (const finding of findings) {
    if (!seen.has(finding.rule_id)) {
      seen.add(finding.rule_id);
      rules.push({ id: finding.rule_id, shortDescription: { text: finding.message } });
    }
  }
  return rules;
}

function sarifResults(findings: readonly Finding[]): readonly SarifResult[] {
  return findings.map((finding) => ({
    ruleId: finding.rule_id,
    level: sarifLevel(finding.severity),
    message: { text: finding.message },
    locations: sarifLocations(finding.path),
    ...(finding.baselined === true ? { suppressions: [{ kind: "external" as const }] } : {})
  }));
}

function sarifLevel(severity: Severity): string {
  return severity === "warn" ? "warning" : "error";
}

function sarifLocations(filePath: string): readonly SarifLocation[] | undefined {
  if (filePath === "") {
    return undefined;
  }
  return [
    {
      physicalLocation: {
        artifactLocation: { uri: filePath },
        region: { startLine: 1 }
      }
    }
  ];
}
