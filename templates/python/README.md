# Python Template

Strict Python baseline for small services, CLIs, libraries, and agent-generated modules.

## Commands

```sh
python -m pip install -e ".[dev]"
ruff format --check .
ruff check .
mypy src tests
pytest
```

## Guardrails

- Ruff handles formatting and linting.
- mypy runs in strict mode and rejects explicit `Any`.
- Tests use pytest and should stay fast.

