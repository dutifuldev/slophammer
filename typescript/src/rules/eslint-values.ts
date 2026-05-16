import ts from "typescript";

export function enforcedRuleValues(
  content: string,
  ruleName: string,
  warningsFail: boolean,
  scope: "core" | "typescript-eslint"
): readonly string[] {
  const effectiveValue = lastValue(eslintRuleValues(content, ruleName, scope));
  if (effectiveValue === undefined || !enforcingRuleValue(effectiveValue, warningsFail)) {
    return [];
  }
  return [effectiveValue];
}

export function stripJavaScriptComments(content: string): string {
  return javascriptTokens(content)
    .map((token) => token.text)
    .join(" ");
}

export function complexityLimit(value: string, maximum: number): boolean {
  const numbers = (value.match(/\b\d+\b/g) ?? []).map((item) => Number.parseInt(item, 10));
  const limitCandidates = numericSeverityValue(value) ? numbers.slice(1) : numbers;
  return limitCandidates.some((parsed) => {
    return Number.isInteger(parsed) && parsed > 0 && parsed <= maximum;
  });
}

function eslintRuleValues(
  content: string,
  ruleName: string,
  scope: "core" | "typescript-eslint"
): readonly string[] {
  const name = scope === "typescript-eslint" ? `@typescript-eslint/${ruleName}` : ruleName;
  const values: string[] = [];
  for (const source of eslintSources(content)) {
    collectRuleValues(source, name, values);
  }
  return values;
}

function eslintSources(content: string): readonly ts.SourceFile[] {
  return [sourceFile(content), sourceFile(`const slophammerConfig = ${content};`)];
}

function sourceFile(content: string): ts.SourceFile {
  return ts.createSourceFile("eslint.config.js", content, ts.ScriptTarget.Latest, true);
}

function collectRuleValues(source: ts.SourceFile, name: string, values: string[]): void {
  visit(source);

  function visit(node: ts.Node): void {
    if (ts.isPropertyAssignment(node) && propertyNameText(node.name) === "rules") {
      collectRulesObjectValues(source, node.initializer, name, values);
    }
    ts.forEachChild(node, visit);
  }
}

function collectRulesObjectValues(
  source: ts.SourceFile,
  node: ts.Expression,
  name: string,
  values: string[]
): void {
  if (!ts.isObjectLiteralExpression(node)) {
    return;
  }
  for (const property of node.properties) {
    if (!ts.isPropertyAssignment(property) || propertyNameText(property.name) !== name) {
      continue;
    }
    values.push(property.initializer.getText(source));
  }
}

function propertyNameText(name: ts.PropertyName): string {
  if (ts.isIdentifier(name) || ts.isStringLiteral(name) || ts.isNumericLiteral(name)) {
    return name.text;
  }
  return "";
}

function lastValue(values: readonly string[]): string | undefined {
  return values.length === 0 ? undefined : values[values.length - 1];
}

type JavaScriptToken = {
  readonly kind: ts.SyntaxKind;
  readonly text: string;
};

function javascriptTokens(content: string): readonly JavaScriptToken[] {
  const scanner = ts.createScanner(
    ts.ScriptTarget.Latest,
    false,
    ts.LanguageVariant.Standard,
    content
  );
  const tokens: JavaScriptToken[] = [];
  let token = scanner.scan();
  while (token !== ts.SyntaxKind.EndOfFileToken) {
    if (!commentToken(token)) {
      tokens.push({ kind: token, text: scanner.getTokenText() });
    }
    token = scanner.scan();
  }
  return tokens;
}

function commentToken(token: ts.SyntaxKind): boolean {
  return (
    token === ts.SyntaxKind.SingleLineCommentTrivia ||
    token === ts.SyntaxKind.MultiLineCommentTrivia
  );
}

function enforcingRuleValue(value: string, warningsFail: boolean): boolean {
  const normalized = value.trim().toLowerCase();
  const compact = normalized.replace(/\s+/gu, "");
  return (
    hasAnyPrefix(compact, enforcingRulePrefixes("error", "2")) ||
    (warningsFail && hasAnyPrefix(compact, enforcingRulePrefixes("warn", "1")))
  );
}

function enforcingRulePrefixes(
  level: "error" | "warn",
  numericLevel: "1" | "2"
): readonly string[] {
  return [
    numericLevel,
    `[${numericLevel}`,
    level,
    `[${level}`,
    `"${level}"`,
    `'${level}'`,
    `["${level}"`,
    `['${level}'`
  ];
}

function hasAnyPrefix(value: string, prefixes: readonly string[]): boolean {
  return prefixes.some((prefix) => value.startsWith(prefix));
}

function numericSeverityValue(value: string): boolean {
  const normalized = value.trim().replace(/\s+/gu, "");
  return ["0", "1", "2"].some((severity) => {
    return normalized.startsWith(severity) || normalized.startsWith(`[${severity}`);
  });
}
