export type Severity = "error" | "warn";

export type Finding = {
  readonly rule_id: string;
  readonly severity: Severity;
  readonly path: string;
  readonly message: string;
  readonly baselined?: true;
};

export type Report = {
  readonly ok: boolean;
  readonly findings: readonly Finding[];
  readonly scope?: ScopeCoverage;
};

// Coverage of configured scope over the production TypeScript files,
// reported so a narrowed scope is visible instead of silent.
export type ScopeCoverage = {
  readonly scanned: number;
  readonly production_files: number;
};

export type Definition = {
  readonly id: string;
  readonly title: string;
  readonly category: string;
  readonly severity: Severity;
  readonly path: string;
  readonly message: string;
  readonly description: string;
  readonly tool?: string;
  readonly status: "implemented";
};

export type Metadata = {
  readonly id: string;
  readonly severity: Severity;
  readonly description: string;
};
