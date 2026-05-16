export type Severity = "error" | "warn";

export type Finding = {
  readonly rule_id: string;
  readonly severity: Severity;
  readonly path: string;
  readonly message: string;
};

export type Report = {
  readonly ok: boolean;
  readonly findings: readonly Finding[];
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
