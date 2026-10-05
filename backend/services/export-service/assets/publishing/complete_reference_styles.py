"""Add missing Pandoc styles without replacing the book's existing styles.

Usage: python complete_reference_styles.py INPUT.docx OUTPUT.docx
Uses the installed Pandoc reference document; never downloads a template.
"""

import argparse
import io
import subprocess
from zipfile import ZipFile, ZIP_DEFLATED

from lxml import etree as ET


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input")
    parser.add_argument("output")
    args = parser.parse_args()
    namespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
    q = "{" + namespace + "}"
    default_docx = subprocess.check_output(["pandoc", "--print-default-data-file", "reference.docx"])
    with ZipFile(io.BytesIO(default_docx)) as archive:
        defaults = ET.fromstring(archive.read("word/styles.xml"))
    with ZipFile(args.input) as archive:
        entries = [(item, archive.read(item.filename)) for item in archive.infolist()]
    styles = ET.fromstring(dict((item.filename, data) for item, data in entries)["word/styles.xml"])
    existing = {style.get(q + "styleId") for style in styles.findall(q + "style")}
    added = []
    for style in defaults.findall(q + "style"):
        style_id = style.get(q + "styleId")
        if style_id not in existing:
            if style_id == "VerbatimChar":
                # Consolas is absent from the local Linux renderer. FreeMono
                # is already shipped in its image; keep CJK fallback explicit.
                fonts = style.find(q + "rPr/" + q + "rFonts")
                fonts.set(q + "ascii", "FreeMono")
                fonts.set(q + "hAnsi", "FreeMono")
                fonts.set(q + "eastAsia", "Noto Sans CJK SC")
                style.find(q + "rPr/" + q + "sz").set(q + "val", "18")
            styles.append(style)
            added.append(style_id)
    if "SourceCode" not in existing:
        style = ET.SubElement(styles, q + "style", {q + "type": "paragraph", q + "customStyle": "1", q + "styleId": "SourceCode"})
        ET.SubElement(style, q + "name", {q + "val": "Source Code"})
        ET.SubElement(style, q + "basedOn", {q + "val": "Normal"})
        ET.SubElement(style, q + "link", {q + "val": "VerbatimChar"})
        paragraph = ET.SubElement(style, q + "pPr")
        ET.SubElement(paragraph, q + "wordWrap", {q + "val": "on"})
        ET.SubElement(paragraph, q + "spacing", {q + "before": "0", q + "after": "0", q + "line": "240", q + "lineRule": "auto"})
        run = ET.SubElement(style, q + "rPr")
        ET.SubElement(run, q + "rFonts", {q + "ascii": "FreeMono", q + "hAnsi": "FreeMono", q + "eastAsia": "Noto Sans CJK SC"})
        ET.SubElement(run, q + "sz", {q + "val": "18"})
        added.append("SourceCode")
    updated = ET.tostring(styles, encoding="utf-8", xml_declaration=True)
    with ZipFile(args.output, "w", ZIP_DEFLATED) as archive:
        for item, data in entries:
            archive.writestr(item, updated if item.filename == "word/styles.xml" else data)
    print("Added styles:", ", ".join(added) or "none")


if __name__ == "__main__":
    main()
