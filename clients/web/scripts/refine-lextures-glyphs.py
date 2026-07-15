"""Apply Lextures' signature t and sail-shaped x glyphs to the web fonts."""

from pathlib import Path

from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont


FONT_DIR = Path(__file__).resolve().parents[1] / "public" / "fonts"


def draw_t(font: TTFont, weight: int) -> None:
    glyph_set = font.getGlyphSet()
    pen = TTGlyphPen(glyph_set)
    stem = 76 + round((weight - 400) * 0.09)
    left = 137 - round((weight - 400) * 0.025)
    right = left + stem
    shoulder = 392 - round((weight - 400) * 0.02)

    # One continuous outline prevents seams where the stem meets the rising crossbar.
    pen.moveTo((left, 693))
    pen.lineTo((right, 693))
    pen.lineTo((right, shoulder + 86))
    pen.lineTo((344, shoulder + 91))
    pen.lineTo((344, shoulder + 12))
    pen.lineTo((right, shoulder + 7))
    pen.lineTo((right, 118))
    pen.qCurveTo((right, 50), (278, 50))
    pen.qCurveTo((312, 50), (332, 62))
    pen.lineTo((344, -3))
    pen.qCurveTo((305, -18), (265, -18))
    pen.qCurveTo((left, -18), (left, 112))
    pen.lineTo((left, shoulder + 4))
    pen.lineTo((24, shoulder))
    pen.lineTo((24, shoulder + 79))
    pen.lineTo((left, shoulder + 83))
    pen.closePath()
    glyph = pen.glyph()
    font["glyf"]["t"] = glyph


def draw_sail_x(font: TTFont, weight: int) -> None:
    glyph_set = font.getGlyphSet()
    pen = TTGlyphPen(glyph_set)
    mast = 24 + round((weight - 400) * 0.035)
    centre = 255

    # A leaning mast replaces the expected crossing strokes.
    pen.moveTo((centre - mast, 0))
    pen.lineTo((centre + 26, 493))
    pen.lineTo((centre + 26 + mast, 493))
    pen.lineTo((centre + mast, 0))
    pen.closePath()

    # Broad leeward sail: deliberately symbol-like and only faintly recognisable as x.
    pen.moveTo((centre + 30, 459))
    pen.qCurveTo((406, 351), (493, 60))
    pen.qCurveTo((376, 118), (centre + 7, 91))
    pen.closePath()

    # Small windward sail creates the opposing diagonal and the Lextures signature.
    pen.moveTo((centre - 26, 405))
    pen.qCurveTo((126, 286), (39, 61))
    pen.qCurveTo((151, 116), (centre - 4, 132))
    pen.closePath()
    glyph = pen.glyph()
    glyph.flags[0] |= 0x40
    font["glyf"]["x"] = glyph


def refine(path: Path) -> None:
    font = TTFont(path)
    weight = font["OS/2"].usWeightClass
    draw_t(font, weight)
    draw_sail_x(font, weight)
    font.flavor = "woff2"
    font.save(path)


for font_path in sorted(FONT_DIR.glob("lextures-*.woff2")):
    refine(font_path)
    print(f"refined {font_path.name}")
