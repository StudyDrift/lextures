import unittest

from research.lib.suppression import Cell, assert_publishable, suppress


class SuppressionTest(unittest.TestCase):
    def test_thresholds_and_complementary_suppression(self):
        cells = [Cell(("K12", "used"), 10, 49, 12), Cell(("K12", "unused"), 20, 100, 20), Cell(("HE", "used"), 30, 90, 11)]
        result = suppress(cells, margins=((0,),))
        self.assertEqual([c.reason for c in result], ["threshold", "complementary", None])
        assert_publishable(result, margins=((0,),))

    def test_rejects_unsafe_unsuppressed_cell(self):
        with self.assertRaises(ValueError): assert_publishable([Cell(("HS",), 1, 100, 2)])


if __name__ == "__main__": unittest.main()

