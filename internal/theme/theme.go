// Package theme holds the two colour palettes every card is drawn in.
//
// GitHub shows a README under four themes: Light, Dark, Dark dimmed and Dark
// high contrast. A card cannot tell them apart, because prefers-color-scheme
// only reports "light" or "dark". So cards paint no background of their own and
// let the page show through, and each one is generated twice -- once in each of
// the palettes below -- to be swapped by a <picture> block.
package theme

// A Theme is the full set of colours one variant of a card is drawn in.
type Theme struct {
	// Name is the file suffix for this variant, "dark" or "light".
	Name string

	// Border is the 1px card outline. Divider is the hairline between sections
	// inside a card, and Track is the unfilled part of a progress bar.
	Border  string
	Divider string
	Track   string

	// The text ramp, strongest to faintest: Strong for numbers and headings,
	// Body for prose, Muted for labels and secondary values, Faint for captions.
	Strong string
	Body   string
	Muted  string
	Faint  string

	// Accent is the profile's orange, used for bars, links and highlights.
	Accent string

	// Status colours, matching the verbs GitHub itself uses for pull requests.
	Green     string
	Purple    string
	Blue      string
	Red       string
	Attention string
	Social    string

	// WordmarkLight darkens the near-white greys in the header wordmark. The
	// artwork's tagline is drawn for a dark background and disappears on white,
	// so the light variant tones those greys down. Zero means leave it alone.
	WordmarkLight bool
}

// Dark is the palette for GitHub's dark themes, taken from the design.
var Dark = Theme{
	Name:      "dark",
	Border:    "#30363d",
	Divider:   "#21262d",
	Track:     "#21262d",
	Strong:    "#e6edf3",
	Body:      "#adbac7",
	Muted:     "#8b949e",
	Faint:     "#7d8590",
	Accent:    "#e08a3c",
	Green:     "#3fb950",
	Purple:    "#a371f7",
	Blue:      "#58a6ff",
	Red:       "#f85149",
	Attention: "#d29922",
	Social:    "#db61a2",
}

// Light is the palette for GitHub's light theme. The design fixes the border,
// text and accent values; the status colours are GitHub's own light tokens so
// the cards sit naturally next to the rest of the page.
var Light = Theme{
	Name:          "light",
	Border:        "#d1d9e0",
	Divider:       "#eaeef2",
	Track:         "#eaeef2",
	Strong:        "#1f2328",
	Body:          "#424a53",
	Muted:         "#59636e",
	Faint:         "#6e7781",
	Accent:        "#bc6c1f",
	Green:         "#1a7f37",
	Purple:        "#8250df",
	Blue:          "#0969da",
	Red:           "#cf222e",
	Attention:     "#9a6700",
	Social:        "#bf3989",
	WordmarkLight: true,
}

// Both is the pair every card is rendered in.
var Both = []Theme{Dark, Light}

// Verb returns the colour for a pull request or issue verb, matching the
// palette GitHub uses for the same states.
func (t Theme) Verb(verb string) string {
	switch verb {
	case "approved":
		return t.Green
	case "merged":
		return t.Purple
	case "opened":
		return t.Blue
	case "closed":
		return t.Red
	case "changes":
		return t.Accent
	case "released":
		return t.Attention
	case "joined":
		return t.Social
	default:
		return t.Muted
	}
}
