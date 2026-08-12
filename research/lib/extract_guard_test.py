import unittest

from research.lib.extract_guard import assert_deidentified, assert_no_excluded_organizations


class ExtractGuardTest(unittest.TestCase):
    def test_opt_out_is_fail_closed(self):
        with self.assertRaises(ValueError): assert_no_excluded_organizations([{"org_id": "blocked"}], {"blocked"})

    def test_identifier_columns_are_rejected(self):
        with self.assertRaises(ValueError): assert_deidentified([{"segment": "K12", "email": "x@example.test"}])


if __name__ == "__main__": unittest.main()

