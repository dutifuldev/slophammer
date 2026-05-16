import { rangesOverlap } from "./report.js";
import { tokenSequences, type TokenEntry, type TokenSequence } from "./tokenize.js";
import type { DryFinding, SourceFile, SourceRange } from "./types.js";

export function findCopiedBlocks(
  files: readonly SourceFile[],
  windowSize: number
): readonly DryFinding[] {
  const matches = copiedBlockMatches(tokenSequences(files), windowSize);
  return collapseOverlapping(matches);
}

type Occurrence = {
  readonly sequence: number;
  readonly index: number;
};

function copiedBlockMatches(
  sequences: readonly TokenSequence[],
  windowSize: number
): readonly DryFinding[] {
  const windows = new Map<string, Occurrence[]>();
  sequences.forEach((sequence, sequenceIndex) => {
    for (let index = 0; index + windowSize <= sequence.tokens.length; index++) {
      const key = tokenWindowKey(sequence.tokens.slice(index, index + windowSize));
      windows.set(key, [...(windows.get(key) ?? []), { sequence: sequenceIndex, index }]);
    }
  });
  return [...windows.values()].flatMap((occurrences) =>
    matchesForOccurrences(sequences, occurrences, windowSize)
  );
}

function matchesForOccurrences(
  sequences: readonly TokenSequence[],
  occurrences: readonly Occurrence[],
  windowSize: number
): readonly DryFinding[] {
  const seen = new Set<string>();
  const matches: DryFinding[] = [];
  for (const pair of adjacentOccurrencePairs(
    compactOccurrences(sequences, occurrences, windowSize)
  )) {
    const match = copiedBlockMatch(sequences, pair.left, pair.right, windowSize);
    if (match !== undefined && !seen.has(findingKey(match))) {
      seen.add(findingKey(match));
      matches.push(match);
    }
  }
  return matches;
}

type OccurrencePair = {
  readonly left: Occurrence;
  readonly right: Occurrence;
};

function compactOccurrences(
  sequences: readonly TokenSequence[],
  occurrences: readonly Occurrence[],
  windowSize: number
): readonly Occurrence[] {
  const lastIndexBySequence = new Map<number, number>();
  const compacted: Occurrence[] = [];
  for (const occurrence of sortOccurrences(sequences, occurrences)) {
    const lastIndex = lastIndexBySequence.get(occurrence.sequence);
    if (lastIndex !== undefined && occurrence.index < lastIndex + windowSize) {
      continue;
    }
    compacted.push(occurrence);
    lastIndexBySequence.set(occurrence.sequence, occurrence.index);
  }
  return compacted;
}

function adjacentOccurrencePairs(occurrences: readonly Occurrence[]): readonly OccurrencePair[] {
  const pairs: OccurrencePair[] = [];
  for (let index = 0; index + 1 < occurrences.length; index++) {
    const left = occurrences[index];
    const right = occurrences[index + 1];
    if (left !== undefined && right !== undefined) {
      pairs.push({ left, right });
    }
  }
  return pairs;
}

function sortOccurrences(
  sequences: readonly TokenSequence[],
  occurrences: readonly Occurrence[]
): readonly Occurrence[] {
  return [...occurrences].sort((left, right) => compareOccurrences(left, right, sequences));
}

function copiedBlockMatch(
  sequences: readonly TokenSequence[],
  leftInput: Occurrence | undefined,
  rightInput: Occurrence | undefined,
  windowSize: number
): DryFinding | undefined {
  if (leftInput === undefined || rightInput === undefined) {
    return undefined;
  }
  const [left, right] = occurrenceLess(rightInput, leftInput, sequences)
    ? [rightInput, leftInput]
    : [leftInput, rightInput];
  if (
    left.sequence === right.sequence &&
    tokenRangesOverlap(
      left.index,
      left.index + windowSize - 1,
      right.index,
      right.index + windowSize - 1
    )
  ) {
    return undefined;
  }
  const match = expandTokenMatch(sequences, left, right, windowSize);
  return match.tokens >= windowSize ? match : undefined;
}

function expandTokenMatch(
  sequences: readonly TokenSequence[],
  left: Occurrence,
  right: Occurrence,
  windowSize: number
): DryFinding {
  const leftTokens = sequences[left.sequence]?.tokens ?? [];
  const rightTokens = sequences[right.sequence]?.tokens ?? [];
  const starts = expandStart(leftTokens, rightTokens, left, right, windowSize);
  const ends = expandEnd(leftTokens, rightTokens, left, right, starts, windowSize);
  return {
    kind: "copied-block",
    left: tokenRange(leftTokens.slice(starts.left, ends.left)),
    right: tokenRange(rightTokens.slice(starts.right, ends.right)),
    tokens: ends.left - starts.left,
    engine: "token-window"
  };
}

type Pair = {
  readonly left: number;
  readonly right: number;
};

function expandStart(
  leftTokens: readonly TokenEntry[],
  rightTokens: readonly TokenEntry[],
  left: Occurrence,
  right: Occurrence,
  windowSize: number
): Pair {
  let leftStart = left.index;
  let rightStart = right.index;
  while (
    leftStart > 0 &&
    rightStart > 0 &&
    leftTokens[leftStart - 1]?.tag === rightTokens[rightStart - 1]?.tag
  ) {
    if (
      left.sequence === right.sequence &&
      tokenRangesOverlap(
        leftStart - 1,
        left.index + windowSize - 1,
        rightStart - 1,
        right.index + windowSize - 1
      )
    ) {
      break;
    }
    leftStart--;
    rightStart--;
  }
  return { left: leftStart, right: rightStart };
}

function expandEnd(
  leftTokens: readonly TokenEntry[],
  rightTokens: readonly TokenEntry[],
  left: Occurrence,
  right: Occurrence,
  starts: Pair,
  windowSize: number
): Pair {
  let leftEnd = left.index + windowSize;
  let rightEnd = right.index + windowSize;
  while (
    leftEnd < leftTokens.length &&
    rightEnd < rightTokens.length &&
    leftTokens[leftEnd]?.tag === rightTokens[rightEnd]?.tag
  ) {
    if (
      left.sequence === right.sequence &&
      tokenRangesOverlap(starts.left, leftEnd, starts.right, rightEnd)
    ) {
      break;
    }
    leftEnd++;
    rightEnd++;
  }
  return { left: leftEnd, right: rightEnd };
}

function collapseOverlapping(matches: readonly DryFinding[]): readonly DryFinding[] {
  const sorted = [...matches].sort(
    (left, right) => right.tokens - left.tokens || findingKey(left).localeCompare(findingKey(right))
  );
  const kept: DryFinding[] = [];
  for (const match of sorted) {
    if (!kept.some((existing) => overlapPair(match, existing))) {
      kept.push(match);
    }
  }
  return kept.sort((left, right) => findingKey(left).localeCompare(findingKey(right)));
}

function overlapPair(left: DryFinding, right: DryFinding): boolean {
  return (
    (rangesOverlap(left.left, right.left) && rangesOverlap(left.right, right.right)) ||
    (rangesOverlap(left.left, right.right) && rangesOverlap(left.right, right.left))
  );
}

function occurrenceLess(
  left: Occurrence,
  right: Occurrence,
  sequences: readonly TokenSequence[]
): boolean {
  return compareOccurrences(left, right, sequences) < 0;
}

function compareOccurrences(
  left: Occurrence,
  right: Occurrence,
  sequences: readonly TokenSequence[]
): number {
  const leftFile = sequences[left.sequence]?.file.path ?? "";
  const rightFile = sequences[right.sequence]?.file.path ?? "";
  const byFile = leftFile.localeCompare(rightFile);
  if (byFile !== 0) {
    return byFile;
  }
  return left.sequence === right.sequence
    ? left.index - right.index
    : left.sequence - right.sequence;
}

function tokenWindowKey(tokens: readonly TokenEntry[]): string {
  return tokens.map((token) => token.tag).join("\0");
}

function tokenRange(tokens: readonly TokenEntry[]): SourceRange {
  const first = tokens[0];
  const last = tokens[tokens.length - 1];
  if (first === undefined || last === undefined) {
    return { path: "", start_line: 0, end_line: 0 };
  }
  return { path: first.file, start_line: first.line, end_line: last.line };
}

function tokenRangesOverlap(
  leftStart: number,
  leftEnd: number,
  rightStart: number,
  rightEnd: number
): boolean {
  return Math.max(leftStart, rightStart) <= Math.min(leftEnd, rightEnd);
}

function findingKey(finding: DryFinding): string {
  return [
    finding.left.path,
    finding.left.start_line,
    finding.left.end_line,
    finding.right.path,
    finding.right.start_line,
    finding.right.end_line,
    finding.tokens
  ].join("|");
}
