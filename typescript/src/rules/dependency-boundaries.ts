import path from "node:path";
import ts from "typescript";

import type { Config } from "../config/config.js";
import type { Snapshot } from "../repo/repo.js";
import type { Definition, Finding } from "./types.js";

export function dependencyBoundaryFindings(
  definition: Definition,
  snapshot: Snapshot,
  cfg: Config
): readonly Finding[] {
  if (cfg.typescript.dependencyBoundaries.length === 0) {
    return [finding(definition)];
  }
  return cfg.typescript.dependencyBoundaries.flatMap((boundary) =>
    importsUnder(snapshot, boundary.from)
      .filter((edge) => !boundaryAllows(edge.to, [boundary.from, ...boundary.allow]))
      .map((edge) =>
        finding(
          definition,
          edge.from,
          `Import ${edge.to} is outside allowed dependencies for ${boundary.from}`
        )
      )
  );
}

type ImportEdge = {
  readonly from: string;
  readonly to: string;
};

function importsUnder(snapshot: Snapshot, root: string): readonly ImportEdge[] {
  return [...snapshot.files.values()]
    .filter((file) => file.path.startsWith(`${root}/`) && sourceExtension(file.path))
    .flatMap((file) =>
      importSpecifiers(file.content).map((specifier) => ({
        from: file.path,
        to: resolveImport(file.path, specifier)
      }))
    );
}

function importSpecifiers(content: string): readonly string[] {
  const specifiers: string[] = [];
  const source = ts.createSourceFile("input.ts", content, ts.ScriptTarget.Latest, true);
  const visit = (node: ts.Node): void => {
    const specifier = importSpecifierFromNode(node);
    if (specifier?.startsWith(".")) {
      specifiers.push(specifier);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
  return specifiers;
}

function importSpecifierFromNode(node: ts.Node): string | undefined {
  if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) {
    return stringLiteralText(node.moduleSpecifier);
  }
  if (ts.isCallExpression(node)) {
    return callImportSpecifier(node);
  }
  if (ts.isImportEqualsDeclaration(node)) {
    return importEqualsSpecifier(node);
  }
  if (ts.isImportTypeNode(node)) {
    return importTypeSpecifier(node);
  }
  return undefined;
}

function callImportSpecifier(node: ts.CallExpression): string | undefined {
  if (node.expression.kind === ts.SyntaxKind.ImportKeyword || requireCall(node)) {
    return stringLiteralText(node.arguments[0]);
  }
  return undefined;
}

function importTypeSpecifier(node: ts.ImportTypeNode): string | undefined {
  const argument = node.argument;
  if (!ts.isLiteralTypeNode(argument)) {
    return undefined;
  }
  return stringLiteralText(argument.literal);
}

function importEqualsSpecifier(node: ts.ImportEqualsDeclaration): string | undefined {
  if (!ts.isExternalModuleReference(node.moduleReference)) {
    return undefined;
  }
  return stringLiteralText(node.moduleReference.expression);
}

function requireCall(node: ts.CallExpression): boolean {
  return ts.isIdentifier(node.expression) && node.expression.text === "require";
}

function stringLiteralText(node: ts.Node | undefined): string | undefined {
  return node !== undefined && ts.isStringLiteral(node) ? node.text : undefined;
}

function resolveImport(from: string, specifier: string): string {
  return path.posix.normalize(path.posix.join(path.posix.dirname(from), specifier));
}

function boundaryAllows(target: string, allowed: readonly string[]): boolean {
  const normalizedTarget = boundaryPath(target);
  return allowed.some((root) => {
    const normalizedRoot = boundaryPath(root);
    return normalizedTarget === normalizedRoot || normalizedTarget.startsWith(`${normalizedRoot}/`);
  });
}

function boundaryPath(filePath: string): string {
  return filePath.replace(/\.(?:[cm]?js|jsx|[cm]?ts|tsx)$/u, "");
}

function finding(definition: Definition, pathOverride?: string, messageOverride?: string): Finding {
  return {
    rule_id: definition.id,
    severity: definition.severity,
    path: pathOverride ?? definition.path,
    message: messageOverride ?? definition.message
  };
}

function sourceExtension(filePath: string): boolean {
  return (
    filePath.endsWith(".ts") ||
    filePath.endsWith(".tsx") ||
    filePath.endsWith(".mts") ||
    filePath.endsWith(".cts") ||
    filePath.endsWith(".js") ||
    filePath.endsWith(".jsx") ||
    filePath.endsWith(".mjs") ||
    filePath.endsWith(".cjs")
  );
}
