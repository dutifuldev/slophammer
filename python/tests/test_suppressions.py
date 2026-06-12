"""Suppression discipline tests."""

from test_rules import clean_python_repo, report_for, rule_ids

ONLY = ["py.suppressions-justified"]


def offenses(source: str) -> list[str]:
    files = clean_python_repo({"src/demo/main.py": source})
    report = report_for(files, only=ONLY)
    return [finding.message for finding in report.findings]


class TestBareSuppressions:
    def test_bare_noqa_is_a_finding(self):
        assert offenses("value = compute()  # noqa\n")

    def test_coded_noqa_without_reason_is_a_finding(self):
        assert offenses("value = compute()  # noqa: ANN401\n")

    def test_coded_noqa_with_reason_passes(self):
        assert not offenses("value = compute()  # noqa: ANN401 -- boundary genuinely returns any\n")

    def test_bare_type_ignore_is_always_a_finding(self):
        assert offenses("value = compute()  # type: ignore -- even with a reason\n")

    def test_coded_type_ignore_needs_a_reason(self):
        assert offenses("value = compute()  # type: ignore[arg-type]\n")
        assert not offenses("value = compute()  # type: ignore[arg-type] -- upstream stub gap\n")

    def test_bare_ty_ignore_is_a_finding(self):
        assert offenses("value = compute()  # ty: ignore -- needs a rule code\n")

    def test_coded_ty_ignore_needs_a_reason(self):
        assert offenses("value = compute()  # ty: ignore[invalid-argument-type]\n")
        assert not offenses(
            "value = compute()  # ty: ignore[invalid-argument-type] -- exercised in tests\n"
        )

    def test_preceding_comment_justifies(self):
        assert not offenses(
            "# the stub for this api lies about the return type\n"
            "value = compute()  # noqa: ANN401\n"
        )

    def test_first_offense_line_is_reported(self):
        messages = offenses("ok = 1\nvalue = compute()  # noqa\nother = c()  # noqa\n")
        assert len(messages) == 1
        assert "(line 2)" in messages[0]


class TestScopeExemptions:
    def test_directives_in_strings_are_invisible(self):
        assert not offenses("TEXT = \"use # noqa to silence\"\nQUOTED = '# type: ignore'\n")

    def test_tests_are_exempt(self):
        files = clean_python_repo({"tests/test_demo.py": "x = 1  # noqa\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_test_prefixed_files_are_exempt(self):
        files = clean_python_repo({"src/demo/test_helpers.py": "x = 1  # noqa\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_conftest_is_exempt(self):
        files = clean_python_repo({"conftest.py": "x = 1  # noqa\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_generated_files_are_exempt(self):
        files = clean_python_repo({"src/generated_client.py": "x = 1  # noqa\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_build_output_is_exempt(self):
        files = clean_python_repo({"out/build/foo.py": "x = 1  # noqa\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_migrations_are_exempt(self):
        files = clean_python_repo({"app/migrations/0001_initial.py": "x = 1  # noqa\n"})
        assert rule_ids(report_for(files, only=ONLY)) == []
