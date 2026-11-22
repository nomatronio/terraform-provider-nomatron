import unittest
from pathlib import Path

import risk_tier

RULES = risk_tier.load_rules(
    (Path(__file__).resolve().parent / "risk-paths.txt").read_text(encoding="utf-8")
)


def tier(*paths):
    return risk_tier.classify(list(paths), RULES)[0]


class RiskTierTest(unittest.TestCase):
    def test_unmatched_is_low(self):
        self.assertEqual(tier("some/new/file.txt"), "low")

    def test_tests_and_examples_are_low(self):
        self.assertEqual(
            tier("internal/provider/group_resource_test.go", "examples/resources/x.tf"),
            "low",
        )

    def test_resource_behaviour_is_medium(self):
        self.assertEqual(tier("internal/provider/group_resource.go"), "medium")

    def test_provider_config_is_high(self):
        self.assertEqual(tier("internal/provider/provider.go"), "high")

    def test_workflow_and_release_files_are_high(self):
        self.assertEqual(tier(".github/workflows/provider.yml"), "high")
        self.assertEqual(tier("goreleaser.yml"), "high")
        self.assertEqual(tier("go.mod"), "high")

    def test_highest_tier_wins(self):
        self.assertEqual(
            tier("README.md", "internal/provider/group_resource.go", "go.sum"), "high"
        )

    def test_checker_files_are_forced_high(self):
        self.assertEqual(tier(".factory/risk_tier.py"), "high")
        self.assertEqual(tier(".factory/state-compatibility.md"), "high")


if __name__ == "__main__":
    unittest.main()
