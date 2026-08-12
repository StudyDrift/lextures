"""Deterministic primary and complementary suppression for aggregate tables."""

from __future__ import annotations

from collections import defaultdict
from dataclasses import dataclass, replace
from typing import Iterable, Sequence

MIN_LEARNERS = 50
MIN_INSTITUTIONS = 10


@dataclass(frozen=True)
class Cell:
    key: tuple[str, ...]
    value: float
    learners: int
    institutions: int
    suppressed: bool = False
    reason: str | None = None


def suppress(cells: Iterable[Cell], margins: Sequence[Sequence[int]] = ()) -> list[Cell]:
    """Apply thresholds, then hide a second cell where a margin has one hidden cell.

    Each margin is a tuple of key indexes defining published totals. Repetition reaches a fixed point,
    which handles overlapping row and column margins without leaking a primary-suppressed value.
    """
    result = [replace(c, suppressed=True, reason="threshold") if c.learners < MIN_LEARNERS or c.institutions < MIN_INSTITUTIONS else c for c in cells]
    changed = True
    while changed:
        changed = False
        for indexes in margins:
            groups: dict[tuple[str, ...], list[int]] = defaultdict(list)
            for i, cell in enumerate(result):
                groups[tuple(cell.key[index] for index in indexes)].append(i)
            for members in groups.values():
                hidden = [i for i in members if result[i].suppressed]
                visible = [i for i in members if not result[i].suppressed]
                if len(hidden) == 1 and visible:
                    candidate = min(visible, key=lambda i: (result[i].value, result[i].key))
                    result[candidate] = replace(result[candidate], suppressed=True, reason="complementary")
                    changed = True
    return result


def assert_publishable(cells: Iterable[Cell], margins: Sequence[Sequence[int]] = ()) -> None:
    rows = list(cells)
    for cell in rows:
        if not cell.suppressed and (cell.learners < MIN_LEARNERS or cell.institutions < MIN_INSTITUTIONS):
            raise ValueError(f"unsafe cell {cell.key!r}")
    for indexes in margins:
        groups: dict[tuple[str, ...], list[Cell]] = defaultdict(list)
        for cell in rows:
            groups[tuple(cell.key[index] for index in indexes)].append(cell)
        if any(sum(c.suppressed for c in group) == 1 for group in groups.values()):
            raise ValueError("a published margin can reveal a suppressed cell")


def public_rows(cells: Iterable[Cell]) -> list[dict[str, object]]:
    return [{"key": list(c.key), "value": None if c.suppressed else c.value, "suppressed": c.suppressed, "suppression_reason": c.reason} for c in cells]

