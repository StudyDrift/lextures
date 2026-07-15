"""Apply Lextures' signature glyph refinements to the web and marketing fonts.

Lextures is Hanken Grotesk with three letters redrawn to give the family a
quiet, deliberate voice without ever fighting the reader:

  * t  a clean-topped stem with a crossbar at the x-height and a foot that
       sweeps into the wind
  * l  a plain ascender that ends in the same sweeping foot, so it can never be
       mistaken for a capital I or a figure 1
  * x  the brand letter, kept as an even grotesque cross with a hair of taper

The shapes are derived from the font itself at draw time -- the stem weight is
read from `dotlessi` and the crossbar depth from `f` -- so every weight stays in
step automatically and the script is safe to re-run.
"""

import shutil
from pathlib import Path

from fontTools.pens.recordingPen import RecordingPen
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont


ROOT = Path(__file__).resolve().parents[3]
FONT_DIR = ROOT / "clients" / "web" / "public" / "fonts"
MIRROR_DIR = ROOT / "www" / "public" / "fonts"  # marketing site, kept in sync

ASCENDER = 697  # height shared by h, k, b, d
X_HEIGHT = 493


def measure(font: TTFont) -> tuple[int, int]:
    """Read the stem weight and crossbar depth from untouched reference glyphs."""
    glyph_set = font.getGlyphSet()

    stem_rec = RecordingPen()
    glyph_set["dotlessi"].draw(stem_rec)
    stem_x = [pt[0] for _, points in stem_rec.value for pt in points]
    stem = max(stem_x) - min(stem_x)

    # The f crossbar's left wing sits at x == 10; its lower edge is the depth we
    # want the t crossbar to share.
    f_rec = RecordingPen()
    glyph_set["f"].draw(f_rec)
    crossbar_bottom = min(
        pt[1] for _, points in f_rec.value for pt in points if pt[0] == 10
    )
    return stem, crossbar_bottom


# The shared "sweep" foot: the stroke leaves the stem high, arcs out on a broad
# two-control curve, and tapers into a soft terminal so it never snaps to the
# baseline. t and l use the exact same foot, which is what pairs them.
FOOT_REACH = 84         # how far the foot sweeps past the stem
FOOT_TAIL_START = 215   # where the stem gives way to the curve (high = gradual)
FOOT_TERM_TOP = 54      # the terminal's widest reach sits above the baseline
FOOT_TERM_CUT = 18      # terminal tucks back to the baseline
FOOT_INNER = 150        # where the underside rejoins the stem


def sweep_foot(pen: TTGlyphPen, left: int, right: int) -> int:
    """Draw the foot from the stem's right edge round to its left edge."""
    terminal = right + FOOT_REACH
    pen.lineTo((right, FOOT_TAIL_START))
    pen.qCurveTo((right, 120), (right + 34, 34), (terminal, FOOT_TERM_TOP))
    pen.lineTo((terminal - FOOT_TERM_CUT, 0))
    pen.qCurveTo((right + 6, 0), (left, FOOT_INNER))
    return terminal


def draw_l(font: TTFont, stem: int) -> None:
    pen = TTGlyphPen(font.getGlyphSet())
    left = 96
    right = left + stem

    pen.moveTo((left, ASCENDER))
    pen.lineTo((right, ASCENDER))
    terminal = sweep_foot(pen, left, right)
    pen.closePath()

    font["glyf"]["l"] = pen.glyph()
    font["hmtx"]["l"] = (terminal + 34, left)


def draw_t(font: TTFont, stem: int, crossbar_bottom: int) -> None:
    pen = TTGlyphPen(font.getGlyphSet())
    left = 48
    right = left + stem
    top = 636
    crossbar_left = 10
    crossbar_right = right + 92

    pen.moveTo((left, top))
    pen.lineTo((right, top))                        # clean flat top, no spike
    pen.lineTo((right, X_HEIGHT))
    pen.lineTo((crossbar_right, X_HEIGHT))          # right crossbar wing
    pen.lineTo((crossbar_right, crossbar_bottom))
    pen.lineTo((right, crossbar_bottom))
    terminal = sweep_foot(pen, left, right)         # same foot as l
    pen.lineTo((left, crossbar_bottom))
    pen.lineTo((crossbar_left, crossbar_bottom))    # left crossbar wing
    pen.lineTo((crossbar_left, X_HEIGHT))
    pen.lineTo((left, X_HEIGHT))
    pen.closePath()

    font["glyf"]["t"] = pen.glyph()
    font["hmtx"]["t"] = (terminal + 30, crossbar_left)


def draw_x(font: TTFont, stem: int) -> None:
    pen = TTGlyphPen(font.getGlyphSet())
    inset = round(stem * 0.10)  # a hair of taper toward the crossing

    pen.moveTo((36, 0))
    pen.lineTo((214 - inset, 246))
    pen.lineTo((44, X_HEIGHT))
    pen.lineTo((150 + inset, X_HEIGHT))
    pen.lineTo((257, 300 - inset))
    pen.lineTo((364 - inset, X_HEIGHT))
    pen.lineTo((470, X_HEIGHT))
    pen.lineTo((300 + inset, 244))
    pen.lineTo((478, 0))
    pen.lineTo((372 - inset, 0))
    pen.lineTo((257, 190 + inset))
    pen.lineTo((144 + inset, 0))
    pen.closePath()

    font["glyf"]["x"] = pen.glyph()


def refine(path: Path) -> None:
    font = TTFont(path)
    stem, crossbar_bottom = measure(font)
    draw_l(font, stem)
    draw_t(font, stem, crossbar_bottom)
    draw_x(font, stem)
    font.flavor = "woff2"
    font.save(path)


for font_path in sorted(FONT_DIR.glob("lextures-*.woff2")):
    refine(font_path)
    print(f"refined {font_path.relative_to(ROOT)}")
    mirror = MIRROR_DIR / font_path.name
    if mirror.exists():
        shutil.copyfile(font_path, mirror)
        print(f"  mirrored to {mirror.relative_to(ROOT)}")
