import io
import os
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest import mock

import risk_tier

RULES = risk_tier.load_rules(
    (Path(__file__).resolve().parent / "risk-paths.txt").read_text(encoding="utf-8")
)


def tier(*paths):
    return risk_tier.classify(list(paths), RULES)[0]


class RiskTierTest(unittest.TestCase):
    def test_unmatched_is_medium(self):
        self.assertEqual(tier("some/new/file.txt"), "medium")

    def test_new_go_file_with_no_rule_is_medium(self):
        self.assertEqual(tier("internal/provider/helpers.go"), "medium")

    def test_examples_and_readme_are_low(self):
        self.assertEqual(tier("examples/resources/x.tf"), "low")
        self.assertEqual(tier("README.md"), "low")

    def test_provider_tests_are_medium(self):
        self.assertEqual(tier("internal/provider/group_resource_test.go"), "medium")

    def test_resource_behaviour_is_medium(self):
        self.assertEqual(tier("internal/provider/group_resource.go"), "medium")

    def test_provider_config_is_high(self):
        self.assertEqual(tier("internal/provider/provider.go"), "high")

    def test_workflow_and_release_files_are_high(self):
        self.assertEqual(tier(".github/workflows/provider.yml"), "high")
        self.assertEqual(tier('.github/workflows/a"b.yml'), "high")
        self.assertEqual(tier("goreleaser.yml"), "high")
        self.assertEqual(tier("go.mod"), "high")

    def test_extra_release_and_dependency_paths_are_high(self):
        self.assertEqual(tier(".goreleaser.yml"), "high")
        self.assertEqual(tier("goreleaser.yaml"), "high")
        self.assertEqual(tier(".config/settings.yml"), "high")
        self.assertEqual(tier("go.work"), "high")
        self.assertEqual(tier("go.work.sum"), "high")
        self.assertEqual(tier("vendor/modules.txt"), "high")

    def test_agents_and_cursor_are_medium(self):
        self.assertEqual(tier(".agents/skills/scope-ticket/SKILL.md"), "medium")
        self.assertEqual(tier(".cursor/rules/provider-layout.mdc"), "medium")

    def test_security_review_skill_is_high(self):
        self.assertEqual(tier(".agents/skills/security-review/SKILL.md"), "high")
        low_rules = risk_tier.load_rules("low .agents/**\n")
        forced = risk_tier.best_rule(
            ".agents/skills/security-review/SKILL.md", low_rules
        )
        self.assertIsNotNone(forced)
        self.assertEqual(forced[0], "high")

    def test_highest_tier_wins(self):
        self.assertEqual(
            tier("README.md", "internal/provider/group_resource.go", "go.sum"), "high"
        )

    def test_checker_files_are_forced_high(self):
        self.assertEqual(tier(".factory/risk_tier.py"), "high")
        self.assertEqual(tier(".factory/state-compatibility.md"), "high")

    def test_nul_name_status_keeps_quotes_and_both_rename_paths(self):
        text = 'M\0.github/workflows/a"b.yml\0R100\0README.md\0new.yml\0'
        paths = risk_tier.parse_name_status(text)
        self.assertEqual(
            paths,
            ['.github/workflows/a"b.yml', "README.md", "new.yml"],
        )
        self.assertEqual(tier(*paths), "high")
        copied = risk_tier.parse_name_status("C050\0docs/a.md\0docs/b.md\0")
        self.assertEqual(copied, ["docs/a.md", "docs/b.md"])

    def test_changed_files_uses_nul_diff(self):
        completed = mock.Mock(returncode=0, stdout='A\0.github/workflows/a"b.yml\0', stderr="")
        base = "a" * 40
        head = "b" * 40
        with mock.patch.object(risk_tier.subprocess, "run", return_value=completed) as run:
            found = risk_tier.changed_files(base, head)
        self.assertEqual(found, ['.github/workflows/a"b.yml'])
        self.assertEqual(
            run.call_args.args[0],
            [
                "git",
                "diff",
                "--name-status",
                "-z",
                "--find-renames",
                "--end-of-options",
                base,
                head,
            ],
        )

    def test_sha_must_be_40_hex(self):
        risk_tier.validate_sha("BASE_SHA", "a" * 40)
        risk_tier.validate_sha("HEAD_SHA", "0123456789abcdefABCDEF0123456789abcdefAB")
        for bad in ("", "a" * 39, "a" * 41, "g" * 40, "HEAD", "--help"):
            with self.assertRaises(SystemExit):
                risk_tier.validate_sha("BASE_SHA", bad)

    def test_main_rejects_sha_before_git(self):
        env = {"BASE_SHA": "not-a-sha", "HEAD_SHA": "b" * 40}
        with mock.patch.dict(os.environ, env):
            with mock.patch.object(risk_tier, "changed_files") as changed:
                with self.assertRaises(SystemExit):
                    risk_tier.main([])
                changed.assert_not_called()

    def test_main_accepts_40_hex_shas(self):
        env = {"BASE_SHA": "a" * 40, "HEAD_SHA": "b" * 40}
        with mock.patch.dict(os.environ, env):
            with mock.patch.object(
                risk_tier, "changed_files", return_value=["README.md"]
            ) as changed:
                with redirect_stdout(io.StringIO()):
                    result = risk_tier.main([])
                self.assertEqual(result, 0)
                changed.assert_called_once_with("a" * 40, "b" * 40)

    def test_files_reject_absolute_and_dotdot(self):
        risk_tier.validate_file_arg('.github/workflows/a"b.yml')
        risk_tier.validate_file_arg("internal/provider/helpers.go")
        for bad in ("/etc/passwd", "../secret", "foo/../../etc/passwd", "foo/../bar"):
            with self.assertRaises(SystemExit):
                risk_tier.validate_file_arg(bad)
            with self.assertRaises(SystemExit):
                risk_tier.main(["--files", bad])


if __name__ == "__main__":
    unittest.main()
