# Slophammer TypeScript

Standalone TypeScript implementation of the Slophammer repository quality
checker. The user-facing product name is `slophammer-ts`.

The TypeScript implementation is native-first. It can also carry selected
checks for other ecosystems when those checks are covered by the shared specs
and fixtures.

## Commands

```sh
npm install
npm run check
slophammer-ts check ..
slophammer-ts rules
slophammer-ts dry ..
npm run mutate
```

The npm package is publishable as `@dutifuldev/slophammer-ts`. The packed
artifact contains runtime `dist/src/**` files and package metadata, exposes the
public `slophammer-ts` bin, and keeps the legacy `slophammer` bin alias during
the transition.

Source-tree development can also run the built CLI directly:

```sh
node dist/src/cli/main.js check ..
node dist/src/cli/main.js dry ..
```

The implementation is intentionally strict. Source uses `unknown` at dynamic
boundaries, rejects `any`, keeps filesystem/process work outside the rule
engine, and declares StrykerJS as the mutation testing gate.
