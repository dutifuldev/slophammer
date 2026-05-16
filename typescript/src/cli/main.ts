#!/usr/bin/env node
import { run } from "./cli.js";

try {
  const result = await run(process.argv.slice(2));
  process.stdout.write(result.stdout);
  process.stderr.write(result.stderr);
  process.exitCode = result.code;
} catch (error) {
  process.stderr.write(error instanceof Error ? `${error.message}\n` : `${String(error)}\n`);
  process.exitCode = 2;
}
