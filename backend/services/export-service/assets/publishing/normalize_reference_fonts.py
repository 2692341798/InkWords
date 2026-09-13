"""Normalize the Chinese book template's font declarations, without changing content/layout.

Usage: python normalize_reference_fonts.py INPUT.docx OUTPUT.docx
No font files are installed or embedded. Clients must have the declared fonts.
"""
import argparse
from zipfile import ZipFile, ZIP_DEFLATED
from lxml import etree as ET

W = 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'
Q = '{' + W + '}'
BODY = 'Noto Sans CJK SC'
MONO = 'Noto Sans Mono CJK SC'
CODE = 'FreeMono'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('input')
    parser.add_argument('output')
    args = parser.parse_args()
    with ZipFile(args.input) as archive:
        entries = [(item, archive.read(item.filename)) for item in archive.infolist()]
    output = []
    for item, data in entries:
        if item.filename.startswith('word/') and item.filename.endswith('.xml'):
            root = ET.fromstring(data)
            changed = False
            for fonts in root.iter(Q + 'rFonts'):
                # Theme attributes can win over explicit families in Word/WPS.
                # Replace inherited Office defaults with the reviewed Chinese profile.
                before = dict(fonts.attrib)
                for name in list(fonts.attrib):
                    if 'theme' in ET.QName(name).localname.lower():
                        del fonts.attrib[name]
                for name in ('ascii', 'hAnsi', 'eastAsia', 'cs'):
                    value = fonts.get(Q + name, BODY)
                    if value not in (BODY, MONO, CODE):
                        value = CODE if value in ('Courier', 'Courier New', 'Consolas') and name != 'eastAsia' else BODY
                    fonts.set(Q + name, value)
                changed |= before != dict(fonts.attrib)
            if item.filename == 'word/fontTable.xml':
                # An unused font table still triggers missing-font notices in WPS.
                for child in list(root):
                    root.remove(child)
                for name in (BODY, MONO, CODE):
                    ET.SubElement(root, Q + 'font', {Q + 'name': name})
                changed = True
            if changed:
                data = ET.tostring(root, encoding='UTF-8', xml_declaration=True, standalone=True)
        output.append((item, data))
    with ZipFile(args.output, 'w', ZIP_DEFLATED) as archive:
        for item, data in output:
            archive.writestr(item, data)


if __name__ == '__main__':
    main()
