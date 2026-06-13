uv run ty check src
uv run ruff check .
uv run ruff format --check .
uv run pytest --cov=src --cov-fail-under=85
uv run slophammer-py dry .
uv run mutmut run
uv run python scripts/check-mutation.py --min-kill-rate 74
uv run pip-audit
uvx slophammer-py check .
