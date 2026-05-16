export type DryOptions = {
  readonly root: string;
  readonly paths: readonly string[];
  readonly exclude: readonly string[];
  readonly maxFindings: number;
  readonly maxFindingsSet: boolean;
  readonly copiedBlockEnabled: boolean;
  readonly copiedBlockSet: boolean;
  readonly copiedBlockTokens: number;
  readonly showReport: boolean;
  readonly format: "text" | "json" | "";
};

export type SourceRange = {
  readonly path: string;
  readonly start_line: number;
  readonly end_line: number;
};

export type DryFinding = {
  readonly kind: "copied-block";
  readonly left: SourceRange;
  readonly right: SourceRange;
  readonly tokens: number;
  readonly engine: "token-window";
};

export type DryGroup = {
  readonly id: string;
  readonly findings: readonly number[];
  readonly kinds: readonly string[];
  readonly left: SourceRange;
  readonly right: SourceRange;
};

export type DryReport = {
  readonly findings: readonly DryFinding[];
  readonly groups: readonly DryGroup[];
};

export type SourceFile = {
  readonly path: string;
  readonly content: string;
};
