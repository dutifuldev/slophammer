# Slophammer TypeScript

Standalone TypeScript implementation of the Slophammer repository quality
checker.

## Commands

```sh
npm install
npm run check
node dist/src/cli/main.js check ..
node dist/src/cli/main.js typescript dry ..
npm run mutate
```

The implementation is intentionally strict. Source uses `unknown` at dynamic
boundaries, rejects `any`, keeps filesystem/process work outside the rule
engine, and declares StrykerJS as the mutation testing gate.
