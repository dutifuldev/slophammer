from __future__ import annotations

from _slophammer_dispatcher import cli


def test_version_uses_bare_distribution(monkeypatch, capsys):
    monkeypatch.setattr(cli.metadata, "version", lambda name: f"{name}-version")

    assert cli.main(["--version"]) == 0

    assert capsys.readouterr().out == "slophammer-version\n"


def test_delegates_to_python_checker(monkeypatch):
    calls: list[list[str]] = []

    def fake_run_python_checker(argv: list[str]) -> int:
        calls.append(argv)
        return 7

    monkeypatch.setattr(cli, "run_python_checker", fake_run_python_checker)

    assert cli.main(["check", "."]) == 7

    assert calls == [["check", "."]]
