package gen

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Presentation keeps labels and artwork separate from provider verification evidence.
type Presentation struct {
	Label  string `json:"label"`
	Icon   string `json:"icon"`
	Fill   string `json:"fill"`
	Accent string `json:"accent"`
}

// textWidth conservatively estimates the shared 11px badge font without platform font dependencies.
func textWidth(text string) int {
	width := 0
	for _, character := range text {
		switch {
		case strings.ContainsRune("ilI.,:;!|' ", character):
			width += 3
		case strings.ContainsRune("mwMW@%", character):
			width += 9
		case unicode.IsUpper(character) || unicode.IsDigit(character):
			width += 7
		default:
			width += 6
		}
	}
	return width
}

// StatusBadge renders compact status segments from measured repository evidence.
func StatusBadge(label, value, color string) []byte {
	title := html.EscapeString(label + ": " + value)
	left, right := textWidth(label)+12, textWidth(value)+12
	width := left + right
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s"><title>%s</title><clipPath id="badge"><rect width="%d" height="20" rx="3"/></clipPath><g clip-path="url(#badge)"><path fill="#555" d="M0 0h%dv20H0z"/><path fill="%s" d="M%d 0h%dv20H%dz"/></g><g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11"><text x="%g" y="14">%s</text><text x="%g" y="14">%s</text></g></svg>
`, width, title, title, width, left, color, left, right, left, float64(left)/2, html.EscapeString(label), float64(left)+float64(right)/2, html.EscapeString(value)))
}

// tint mixes a brand accent with white for a legible outline or background.
func tint(accent string, fraction float64) (string, error) {
	if len(accent) != 7 || accent[0] != '#' {
		return "", fmt.Errorf("invalid badge accent %q", accent)
	}
	var result strings.Builder
	result.WriteByte('#')
	for index := 1; index < 7; index += 2 {
		channel, err := strconv.ParseUint(accent[index:index+2], 16, 8)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&result, "%02x", int(float64(channel)*fraction+255*(1-fraction)+0.5))
	}
	return result.String(), nil
}

// illustratorMetadata removes empty export metadata while preserving visible provider artwork.
var illustratorMetadata = regexp.MustCompile(`(?s)<foreignObject\b[^>]*(?:/>|>.*?</foreignObject>)`)

// svgArtwork preserves vector paths, gradients and styling inside an explicitly sized viewport.
func svgArtwork(data []byte, fill string, x int) (string, error) {
	var source struct {
		XMLName    xml.Name
		Attributes []xml.Attr `xml:",any,attr"`
		Inner      string     `xml:",innerxml"`
	}
	if err := xml.Unmarshal(data, &source); err != nil {
		return "", err
	}
	if source.XMLName.Local != "svg" {
		return "", fmt.Errorf("expected SVG artwork")
	}
	var out strings.Builder
	fmt.Fprintf(&out, `<svg x="%d" y="3" width="16" height="16"`, x)
	for _, attribute := range source.Attributes {
		name := attribute.Name.Local
		if name == "x" || name == "y" || name == "width" || name == "height" || name == "xmlns" || (name == "fill" && fill != "") {
			continue
		}
		if attribute.Name.Space == "http://www.w3.org/2000/xmlns/" || attribute.Name.Space == "xmlns" {
			name = "xmlns:" + name
		} else if attribute.Name.Space == "http://www.w3.org/XML/1998/namespace" {
			name = "xml:" + name
		}
		fmt.Fprintf(&out, ` %s="%s"`, name, html.EscapeString(attribute.Value))
	}
	if fill != "" {
		fmt.Fprintf(&out, ` fill="%s"`, html.EscapeString(fill))
	}
	out.WriteByte('>')
	out.WriteString(illustratorMetadata.ReplaceAllString(source.Inner, ""))
	out.WriteString("</svg>")
	return out.String(), nil
}

// ArtworkBadge embeds local artwork; labelled driver badges stay aligned beside searchable names.
func ArtworkBadge(root, directory string, presentation Presentation, labelled bool) ([]byte, error) {
	accent := presentation.Accent
	if accent == "" {
		accent = "#576574"
	}
	background, err := tint(accent, 0.08)
	if err != nil {
		return nil, err
	}
	border, err := tint(accent, 0.28)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, "docs/assets", directory, presentation.Icon)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	width, x := 22, 3
	label := html.EscapeString(presentation.Label)
	text := ""
	if labelled {
		width = labelledWidth(presentation.Label)
		x = 7
		text = fmt.Sprintf(`<text x="30" y="15" fill="#24292f" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11">%s</text>`, label)
	}
	var artwork string
	switch filepath.Ext(path) {
	case ".svg":
		artwork, err = svgArtwork(data, presentation.Fill, x)
		if err != nil {
			return nil, err
		}
	case ".png":
		artwork = fmt.Sprintf(`<image x="%d" y="3" width="16" height="16" href="data:image/png;base64,%s"/>`, x, base64.StdEncoding.EncodeToString(data))
	default:
		return nil, fmt.Errorf("unsupported badge artwork %s", path)
	}
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="22" role="img" aria-label="%s"><title>%s</title><rect x=".5" y=".5" width="%d" height="21" rx="4" fill="%s" stroke="%s"/>%s%s</svg>
`, width, label, label, width-1, background, border, artwork, text)), nil
}

// labelledWidth fixes the rendered badge size so GitHub tables cannot shrink its typography.
func labelledWidth(label string) int { return textWidth(label) + 46 }
