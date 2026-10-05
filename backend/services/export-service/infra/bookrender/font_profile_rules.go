package bookrender

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// Flatten only rendering rules. Includes are never followed; directories and
// caches are supplied by the per-render profile, not user/system customization.
func extractFontRules(raw []byte, systemRoot bool) ([]byte, error) {
	d := xml.NewDecoder(bytes.NewReader(raw))
	var out bytes.Buffer
	e := xml.NewEncoder(&out)
	depth, copyDepth, roots := 0, 0, 0
	for {
		token, err := d.Token()
		if errors.Is(err, io.EOF) {
			if depth != 0 || roots != 1 {
				return nil, fmt.Errorf("invalid Fontconfig root")
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid Fontconfig XML: %w", err)
		}
		switch node := token.(type) {
		case xml.StartElement:
			depth++
			switch depth {
			case 1:
				roots++
				if node.Name.Local != "fontconfig" || node.Name.Space != "" {
					return nil, fmt.Errorf("invalid Fontconfig root")
				}
			case 2:
				switch node.Name.Local {
				case "match", "alias":
					copyDepth = depth
				case "description":
				case "dir", "include", "cachedir", "config":
					if !systemRoot {
						return nil, fmt.Errorf("external Fontconfig configuration forbidden")
					}
				default:
					return nil, fmt.Errorf("unsupported Fontconfig rule")
				}
			}
			if copyDepth > 0 {
				if node.Name.Space != "" || node.Name.Local == "include" || node.Name.Local == "dir" || node.Name.Local == "remap-dir" || node.Name.Local == "cachedir" {
					return nil, fmt.Errorf("font source mutation forbidden")
				}
				for _, attr := range node.Attr {
					if attr.Name.Local == "name" && (attr.Value == "file" || attr.Value == "fontwrapper") {
						return nil, fmt.Errorf("explicit font file rule forbidden")
					}
				}
			}
		case xml.EndElement:
			if copyDepth == depth {
				if err := e.EncodeToken(token); err != nil {
					return nil, err
				}
				copyDepth = 0
			}
			depth--
		}
		if copyDepth > 0 {
			if _, comment := token.(xml.Comment); !comment {
				if err := e.EncodeToken(token); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := e.Flush(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func fontProfileEnvironment(base []string, config string) []string {
	result := make([]string, 0, len(base)+3)
	for _, item := range base {
		if strings.HasPrefix(item, "FONTCONFIG_") || strings.HasPrefix(item, "FC_DEBUG=") {
			continue
		}
		result = append(result, item)
	}
	return append(result, "FONTCONFIG_FILE="+config, "FONTCONFIG_PATH="+filepath.Dir(config))
}

// Charset ranges come from Fontconfig's own font scanner. This is a coverage
// check, not shaping correctness or a claim that every glyph was rendered.
func requireFontCharacters(charsets, text string) error {
	type interval struct{ first, last rune }
	ranges := []interval{}
	for _, field := range strings.Fields(charsets) {
		first, last, paired := strings.Cut(field, "-")
		if !paired {
			last = first
		}
		a, errA := strconv.ParseUint(first, 16, 32)
		b, errB := strconv.ParseUint(last, 16, 32)
		if errA != nil || errB != nil || a > b || b > unicode.MaxRune {
			return fmt.Errorf("invalid font charset range %.40q", field)
		}
		ranges = append(ranges, interval{rune(a), rune(b)})
	}
	for _, char := range text {
		// Variation selectors are not standalone glyphs (Unicode ch. 23).
		// Checking the base character does not certify the selected variant.
		if unicode.IsSpace(char) || unicode.IsControl(char) || unicode.Is(unicode.Variation_Selector, char) {
			continue
		}
		covered := false
		for _, r := range ranges {
			if char >= r.first && char <= r.last {
				covered = true
				break
			}
		}
		if !covered {
			return fmt.Errorf("font profile lacks character U+%04X", char)
		}
	}
	return nil
}
