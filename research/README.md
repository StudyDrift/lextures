# Lextures research program

This directory is the review boundary for original research. A report may not enter analysis until its
committed `plan.md` and DPIA are approved. Analysts receive only a de-identified extract containing
explicitly participating tenants. Published cells must represent at least 50 learners and 10
institutions; complementary suppression is computed by `lib/suppression.py`, never by hand.

## Required sequence

1. Copy `templates/analysis-plan.md` and `templates/dpia-checklist.md` into a report directory.
2. Commit and approve both documents before creating an analysis commit.
3. Generate the extract in a controlled, access-logged environment. Treat missing participation as
   exclusion and assert the extract contains no opted-out organization IDs.
4. Run suppression and `assert_publishable`; independently review analysis code and exact claims.
5. Publish the web report, CSV, JSON, data dictionary, chart tables, press kit, version history, and
   analysis code where safe. All artifacts use CC BY 4.0.
6. Correct through a dated erratum and retained prior version. Never silently replace a result.

Null and unfavorable results publish with equal prominence. Suppression for any non-privacy reason
requires a recorded CEO decision. Marketing claims must reference stable dataset `figure_id` values.

Run the privacy unit tests with `python3 -m unittest discover research -p '*_test.py'`.

