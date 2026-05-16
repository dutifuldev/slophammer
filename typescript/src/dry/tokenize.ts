import ts from "typescript";

import type { SourceFile } from "./types.js";

export type TokenEntry = {
  readonly tag: string;
  readonly file: string;
  readonly line: number;
};

export type TokenSequence = {
  readonly file: SourceFile;
  readonly tokens: readonly TokenEntry[];
};

export function tokenSequences(files: readonly SourceFile[]): readonly TokenSequence[] {
  return files.map((file) => ({ file, tokens: tokenize(file) }));
}

function tokenize(file: SourceFile): readonly TokenEntry[] {
  const scanner = ts.createScanner(
    ts.ScriptTarget.Latest,
    false,
    ts.LanguageVariant.Standard,
    file.content
  );
  const lineStarts = lineStartOffsets(file.content);
  const tokens: TokenEntry[] = [];
  let token = scanner.scan();
  while (token !== ts.SyntaxKind.EndOfFileToken) {
    if (keepToken(token)) {
      tokens.push({
        tag: tokenTag(token, scanner.getTokenText()),
        file: file.path,
        line: lineForPosition(lineStarts, scanner.getTokenStart())
      });
    }
    token = scanner.scan();
  }
  return tokens;
}

function keepToken(token: ts.SyntaxKind): boolean {
  return (
    token !== ts.SyntaxKind.WhitespaceTrivia &&
    token !== ts.SyntaxKind.NewLineTrivia &&
    token !== ts.SyntaxKind.SingleLineCommentTrivia &&
    token !== ts.SyntaxKind.MultiLineCommentTrivia &&
    token !== ts.SyntaxKind.ShebangTrivia &&
    token !== ts.SyntaxKind.ConflictMarkerTrivia
  );
}

function tokenTag(token: ts.SyntaxKind, text: string): string {
  if (token === ts.SyntaxKind.Identifier) {
    return `ident/${text}`;
  }
  if (literalToken(token)) {
    return `literal/${ts.SyntaxKind[token]}/${text}`;
  }
  return ts.SyntaxKind[token];
}

function literalToken(token: ts.SyntaxKind): boolean {
  return (
    token === ts.SyntaxKind.NumericLiteral ||
    token === ts.SyntaxKind.StringLiteral ||
    token === ts.SyntaxKind.BigIntLiteral ||
    token === ts.SyntaxKind.RegularExpressionLiteral ||
    token === ts.SyntaxKind.NoSubstitutionTemplateLiteral ||
    token === ts.SyntaxKind.TemplateHead ||
    token === ts.SyntaxKind.TemplateMiddle ||
    token === ts.SyntaxKind.TemplateTail
  );
}

function lineStartOffsets(content: string): readonly number[] {
  const starts = [0];
  for (let index = 0; index < content.length; index++) {
    if (content[index] === "\n") {
      starts.push(index + 1);
    }
  }
  return starts;
}

function lineForPosition(starts: readonly number[], position: number): number {
  let low = 0;
  let high = starts.length - 1;
  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    const start = starts[mid] ?? 0;
    if (start <= position) {
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }
  return high + 1;
}
