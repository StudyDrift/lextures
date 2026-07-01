#!/usr/bin/env python3
"""Generate iOS String Catalog and Android strings.xml from clients/mobile/locales/*.json."""

from __future__ import annotations

import json
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LOCALES_DIR = ROOT / "clients" / "mobile" / "locales"
IOS_CATALOG = ROOT / "clients" / "ios" / "Lextures" / "Resources" / "Localizable.xcstrings"
ANDROID_RES = ROOT / "clients" / "android" / "app" / "src" / "main" / "res"

ANDROID_LOCALE_DIRS = {
    "en": "values",
    "es": "values-es",
    "fr": "values-fr",
    "ar": "values-ar",
    "en-XA": "values-en-rXA",
}

ANDROID_ARABIC_PLURAL_QUANTITIES = ("zero", "one", "two", "few", "many", "other")

IOS_LOCALE_CODES = {
    "en": "en",
    "es": "es",
    "fr": "fr",
    "ar": "ar",
    "en-XA": "en-XA",
}


def android_name(key: str) -> str:
    return key.replace(".", "_").replace("-", "_")


def android_format(value: str) -> str:
    out = value.replace("%lld", "%d")
    arg = 1
    while "%@" in out:
        out = out.replace("%@", f"%{arg}$s", 1)
        arg += 1
    while "%d" in out:
        out = out.replace("%d", f"%{arg}$d", 1)
        arg += 1
    return out


def escape_android_string(value: str) -> str:
    """Escape characters Android treats specially in <string> resources."""
    out = value.replace("\\", "\\\\")
    out = out.replace("'", "\\'")
    if out.startswith("@"):
        out = "\\" + out
    return out


def complete_android_plural_forms(tag: str, forms: dict[str, str]) -> dict[str, str]:
    """Android lint requires all CLDR plural categories for Arabic."""
    if tag != "ar":
        return forms
    completed = dict(forms)
    fallback = (
        completed.get("other")
        or completed.get("many")
        or completed.get("few")
        or completed.get("two")
        or completed.get("one")
        or completed.get("zero")
        or ""
    )
    for quantity in ANDROID_ARABIC_PLURAL_QUANTITIES:
        completed.setdefault(quantity, fallback)
    return completed


def load_locale(tag: str) -> dict:
    path = LOCALES_DIR / f"{tag}.json"
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


def all_locale_tags() -> list[str]:
    return sorted(path.stem for path in LOCALES_DIR.glob("*.json"))


def validate_key_parity(tags: list[str]) -> None:
    reference = load_locale("en")
    ref_strings = set(reference.get("strings", {}))
    ref_plurals = set(reference.get("plurals", {}))

    for tag in tags:
        if tag == "en":
            continue
        data = load_locale(tag)
        strings = set(data.get("strings", {}))
        plurals = set(data.get("plurals", {}))
        missing_strings = ref_strings - strings
        missing_plurals = ref_plurals - plurals
        extra_strings = strings - ref_strings
        extra_plurals = plurals - ref_plurals
        if missing_strings or missing_plurals or extra_strings or extra_plurals:
            raise SystemExit(
                f"Locale {tag} key mismatch.\n"
                f"  missing strings: {sorted(missing_strings)}\n"
                f"  missing plurals: {sorted(missing_plurals)}\n"
                f"  extra strings: {sorted(extra_strings)}\n"
                f"  extra plurals: {sorted(extra_plurals)}"
            )


def indent_xml(elem: ET.Element) -> None:
    ET.indent(elem, space="    ")


def write_android_strings(tag: str, data: dict) -> None:
    folder = ANDROID_LOCALE_DIRS.get(tag)
    if folder is None:
        raise SystemExit(f"No Android resource folder mapping for locale {tag}")

    resources = ET.Element("resources")
    for key, value in sorted(data.get("strings", {}).items()):
        item = ET.SubElement(resources, "string", name=android_name(key))
        item.text = escape_android_string(android_format(value))

    for key, forms in sorted(data.get("plurals", {}).items()):
        plural = ET.SubElement(resources, "plurals", name=android_name(key))
        for quantity in ("zero", "one", "two", "few", "many", "other"):
            localized = complete_android_plural_forms(tag, forms)
            if quantity in localized:
                entry = ET.SubElement(plural, "item", quantity=quantity)
                entry.text = escape_android_string(android_format(localized[quantity]))

    indent_xml(resources)
    out_dir = ANDROID_RES / folder
    out_dir.mkdir(parents=True, exist_ok=True)
    out_path = out_dir / "strings.xml"
    # Use ElementTree only — xml.dom.minidom escapes quotes differently on macOS vs Linux.
    xml_body = ET.tostring(resources, encoding="unicode")
    out_path.write_text('<?xml version="1.0" encoding="utf-8"?>\n' + xml_body + "\n", encoding="utf-8")


def ios_string_unit(value: str) -> dict:
    return {"stringUnit": {"state": "translated", "value": value}}


def ios_plural_variations(forms: dict[str, str]) -> dict:
    plural: dict[str, dict] = {}
    for quantity, value in forms.items():
        plural[quantity] = ios_string_unit(value)
    return {"variations": {"plural": plural}}


def write_ios_catalog(tags: list[str]) -> None:
    locale_data = {tag: load_locale(tag) for tag in tags}
    en = locale_data["en"]
    strings: dict[str, dict] = {}

    for key in sorted(en.get("strings", {})):
        entry: dict = {"localizations": {}}
        for tag in tags:
            ios_code = IOS_LOCALE_CODES[tag]
            value = locale_data[tag]["strings"][key]
            entry["localizations"][ios_code] = ios_string_unit(value)
        strings[key] = entry

    for key in sorted(en.get("plurals", {})):
        entry = {"localizations": {}}
        for tag in tags:
            ios_code = IOS_LOCALE_CODES[tag]
            forms = locale_data[tag]["plurals"][key]
            entry["localizations"][ios_code] = ios_plural_variations(forms)
        strings[key] = entry

    catalog = {
        "sourceLanguage": "en",
        "strings": strings,
        "version": "1.0",
    }
    IOS_CATALOG.parent.mkdir(parents=True, exist_ok=True)
    with IOS_CATALOG.open("w", encoding="utf-8") as handle:
        json.dump(catalog, handle, ensure_ascii=False, indent=2)
        handle.write("\n")


def main() -> int:
    tags = all_locale_tags()
    if "en" not in tags:
        print("FAIL: missing en.json locale source", file=sys.stderr)
        return 1

    validate_key_parity(tags)

    for tag in tags:
        write_android_strings(tag, load_locale(tag))

    write_ios_catalog(tags)
    print(f"Synced {len(tags)} mobile locales to iOS and Android resources.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
