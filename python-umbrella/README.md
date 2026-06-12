# slophammer

The bare `slophammer` command for the Slophammer repository quality checker.

This package intentionally does not provide an `import slophammer` module. The
Python implementation owns that import namespace through `slophammer-py`; this
package only installs the `slophammer` command and delegates to the pinned
Python checker release.

```sh
uvx slophammer check .
uvx slophammer dry .
```

See the repository root for the rule reference and configuration docs.
