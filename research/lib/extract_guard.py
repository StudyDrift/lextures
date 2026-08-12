"""Fail-closed assertions applied before a de-identified extract leaves production."""

from collections.abc import Iterable, Mapping


def assert_no_excluded_organizations(rows: Iterable[Mapping[str, object]], excluded_org_ids: set[str]) -> None:
    leaked = {str(row.get("org_id")) for row in rows if str(row.get("org_id")) in excluded_org_ids}
    if leaked:
        raise ValueError(f"extract contains {len(leaked)} excluded organization(s)")


def assert_deidentified(rows: Iterable[Mapping[str, object]]) -> None:
    forbidden = {"learner_id", "user_id", "email", "name", "course_id", "instructor_id", "school_id", "org_id"}
    for row in rows:
        found = forbidden.intersection(row)
        if found:
            raise ValueError(f"extract contains forbidden identifiers: {', '.join(sorted(found))}")

