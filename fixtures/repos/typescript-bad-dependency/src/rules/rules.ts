import { describeApp } from "../app/app.js";
import { describeRepo } from "../repo/repo.js";

export function describeRules(): string {
  return `${describeApp()} ${describeRepo()}`;
}
