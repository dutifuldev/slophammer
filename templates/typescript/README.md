# TypeScript Template

Strict TypeScript baseline for small services, CLIs, libraries, and agent-generated modules.

## Commands

```sh
npm install
npm run check
npm test
```

## Guardrails

- `strict` compiler mode is enabled.
- Explicit `any` is rejected by ESLint.
- Unsafe assignments, calls, member access, and returns are rejected.
- Tests use Vitest and should stay fast.
