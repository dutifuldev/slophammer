# Slophammer TypeScript

Standalone TypeScript implementation of the Slophammer repository quality
checker. The user-facing product name is `slophammer-ts`.

The TypeScript implementation is native-first. It can also carry selected
checks for other ecosystems when those checks are covered by the shared specs
and fixtures.

## Commands

```sh
npm install -g slophammer-ts
slophammer-ts check .
slophammer-ts rules
slophammer-ts rules --format json
slophammer-ts dry .
```

Source-tree development uses the local package scripts:

```sh
npm install
npm run check
slophammer-ts check ..
slophammer-ts rules
slophammer-ts rules --format json
slophammer-ts dry ..
npm run mutate
```

The npm package is released as `slophammer-ts`. The packed artifact contains
runtime `dist/src/**` files and package metadata, and exposes the public
`slophammer-ts` bin.

The `slophammer` npm package name is reserved for a future umbrella package or
default installer. The TypeScript implementation should not claim the
`slophammer` bin in its published package.

Source-tree development can also run the built CLI directly:

```sh
node dist/src/cli/main.js check ..
node dist/src/cli/main.js dry ..
```

The implementation is intentionally strict. Source uses `unknown` at dynamic
boundaries, rejects `any`, keeps filesystem/process work outside the rule
engine, and declares StrykerJS as the mutation testing gate.
