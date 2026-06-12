"""The ty strictness contract and the mypy/pyright strictness checks."""

from test_rules import STRICT_PYPROJECT, STRICT_TY_TOML, clean_python_repo, report_for, rule_ids

ONLY = ["py.types-strict-required"]


def messages(files: dict[str, str]) -> str:
    report = report_for(files, only=ONLY)
    return "\n".join(finding.message for finding in report.findings)


class TestTyContract:
    def test_full_contract_passes(self):
        assert rule_ids(report_for(clean_python_repo(), only=ONLY)) == []

    def test_demoting_a_default_error_rule_is_a_finding(self):
        files = clean_python_repo(
            {"ty.toml": STRICT_TY_TOML + '\nunresolved-attribute = "ignore"\n'}
        )
        assert "default-error ty rules demoted" in messages(files)
        assert "unresolved-attribute" in messages(files)

    def test_reasoned_demotion_in_slophammer_yml_is_allowed(self):
        files = clean_python_repo(
            {
                "ty.toml": STRICT_TY_TOML + '\nunresolved-attribute = "ignore"\n',
                "slophammer.yml": (
                    "python:\n"
                    "  typecheck:\n"
                    "    demotions:\n"
                    "      - rule: unresolved-attribute\n"
                    "        reason: upstream false positive on sqlalchemy descriptors\n"
                ),
            }
        )
        assert "demoted" not in messages(files)

    def test_cli_flag_demotion_is_a_finding(self):
        files = clean_python_repo()
        files[".github/workflows/ci.yml"] = files[".github/workflows/ci.yml"].replace(
            "uv run ty check src --error-on-warning",
            "uv run ty check src --error-on-warning --ignore unresolved-attribute",
        )
        assert "unresolved-attribute" in messages(files)

    def test_missing_error_on_warning_is_a_finding(self):
        ty_toml = STRICT_TY_TOML.replace("error-on-warning = true", "error-on-warning = false")
        files = clean_python_repo({"ty.toml": ty_toml})
        files[".github/workflows/ci.yml"] = files[".github/workflows/ci.yml"].replace(
            " --error-on-warning", ""
        )
        assert "error-on-warning" in messages(files)

    def test_missing_promotions_are_findings(self):
        ty_toml = STRICT_TY_TOML.replace('missing-type-argument = "error"\n', "")
        files = clean_python_repo({"ty.toml": ty_toml})
        assert "missing-type-argument" in messages(files)

    def test_respecting_type_ignore_comments_is_a_finding(self):
        ty_toml = STRICT_TY_TOML.replace(
            "respect-type-ignore-comments = false", "respect-type-ignore-comments = true"
        )
        files = clean_python_repo({"ty.toml": ty_toml})
        assert "respect-type-ignore-comments" in messages(files)

    def test_unknown_rules_are_tolerated(self):
        files = clean_python_repo(
            {"ty.toml": STRICT_TY_TOML + '\nrule-from-the-future = "ignore"\n'}
        )
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_tool_ty_section_in_pyproject_counts(self):
        files = clean_python_repo(
            {
                "ty.toml": "",
                "pyproject.toml": STRICT_PYPROJECT
                + (
                    "\n[tool.ty.terminal]\nerror-on-warning = true\n"
                    "\n[tool.ty.analysis]\nrespect-type-ignore-comments = false\n"
                    "\n[tool.ty.rules]\n"
                    'missing-type-argument = "error"\n'
                    'possibly-missing-attribute = "error"\n'
                    'possibly-unresolved-reference = "error"\n'
                    'possibly-missing-import = "error"\n'
                ),
            }
        )
        del files["ty.toml"]
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_extend_select_ann_counts(self):
        files = clean_python_repo(
            {
                "pyproject.toml": STRICT_PYPROJECT.replace(
                    'select = ["E", "F", "ANN", "C90"]',
                    'select = ["E", "F"]\nextend-select = ["ANN", "C90"]',
                )
            }
        )
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_ignored_ann_family_is_a_finding(self):
        files = clean_python_repo(
            {
                "pyproject.toml": STRICT_PYPROJECT.replace(
                    'select = ["E", "F", "ANN", "C90"]',
                    'select = ["ALL"]\nignore = ["ANN401"]',
                )
            }
        )
        assert "ANN" in messages(files)

    def test_missing_ann_selection_is_a_finding(self):
        files = clean_python_repo({"pyproject.toml": STRICT_PYPROJECT.replace('"ANN", ', "")})
        assert "ANN" in messages(files)


def mypy_repo(extra_pyproject: str = "", strict_section: str = "[tool.mypy]\nstrict = true\n"):
    files = clean_python_repo(
        {
            "pyproject.toml": STRICT_PYPROJECT + "\n" + strict_section + extra_pyproject,
        }
    )
    del files["ty.toml"]
    files[".github/workflows/ci.yml"] = files[".github/workflows/ci.yml"].replace(
        "uv run ty check src --error-on-warning", "uv run mypy src"
    )
    return files


class TestMypy:
    def test_strict_mypy_passes(self):
        assert rule_ids(report_for(mypy_repo(), only=ONLY)) == []

    def test_unstrict_mypy_is_a_finding(self):
        files = mypy_repo(strict_section="[tool.mypy]\npretty = true\n")
        assert "strict" in messages(files)

    def test_disallow_untyped_defs_flags_count_as_strict(self):
        files = mypy_repo(
            strict_section=(
                "[tool.mypy]\n"
                "disallow_untyped_defs = true\n"
                "disallow_incomplete_defs = true\n"
                "check_untyped_defs = true\n"
            )
        )
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_pydantic_without_plugin_is_a_finding(self):
        files = mypy_repo(extra_pyproject="")
        files["pyproject.toml"] = files["pyproject.toml"].replace(
            'version = "0.0.0"', 'version = "0.0.0"\ndependencies = ["pydantic>=2"]'
        )
        assert "pydantic plugin" in messages(files)

    def test_pydantic_with_plugin_passes(self):
        files = mypy_repo(
            strict_section='[tool.mypy]\nstrict = true\nplugins = ["pydantic.mypy"]\n'
        )
        files["pyproject.toml"] = files["pyproject.toml"].replace(
            'version = "0.0.0"', 'version = "0.0.0"\ndependencies = ["pydantic>=2"]'
        )
        assert rule_ids(report_for(files, only=ONLY)) == []


class TestPyright:
    def test_strict_pyright_passes(self):
        files = mypy_repo(strict_section="")
        files[".github/workflows/ci.yml"] = files[".github/workflows/ci.yml"].replace(
            "uv run mypy src", "uv run pyright src"
        )
        files["pyrightconfig.json"] = '{"typeCheckingMode": "strict"}'
        assert rule_ids(report_for(files, only=ONLY)) == []

    def test_basic_pyright_is_a_finding(self):
        files = mypy_repo(strict_section="")
        files[".github/workflows/ci.yml"] = files[".github/workflows/ci.yml"].replace(
            "uv run mypy src", "uv run pyright src"
        )
        files["pyrightconfig.json"] = '{"typeCheckingMode": "basic"}'
        assert "typeCheckingMode" in messages(files)


def test_no_typechecker_leaves_strict_rule_silent():
    files = clean_python_repo()
    files[".github/workflows/ci.yml"] = files[".github/workflows/ci.yml"].replace(
        "      - run: uv run ty check src --error-on-warning\n", ""
    )
    assert rule_ids(report_for(files, only=ONLY)) == []
