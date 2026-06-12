from __future__ import annotations

from _slophammer_dispatcher import cli


def test_version_uses_bare_distribution(monkeypatch, capsys):
    monkeypatch.setattr(cli.metadata, "version", lambda name: f"{name}-version")

    assert cli.main(["--version"]) == 0

    assert capsys.readouterr().out == "slophammer-version\n"


def test_placeholder_command_points_to_current_checker(capsys):
    assert cli.main(["check", "."]) == 2

    captured = capsys.readouterr()
    assert captured.out == ""
    assert "Use slophammer-py" in captured.err
