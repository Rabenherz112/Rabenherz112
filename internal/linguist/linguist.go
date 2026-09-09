// Package linguist maps a language name to the bar colour used on the cards.
//
// The colours are GitHub's own, from Linguist, so a language reads the same
// here as it does on the language strip of any repository.
//
// A handful differ deliberately. Linguist picks colours to sit on GitHub's
// white repository page, and a few of them are near-black: PowerShell's
// #012456 and JSON's #292929 are all but invisible on a dark README. Where a
// Linguist colour fails on a dark background, the entry below uses the
// language's own brand colour instead, which is legible on both. Those are
// marked.
package linguist

import "strings"

// Unknown is the neutral grey used for a language with no entry. It is
// readable against both a light and a dark page.
const Unknown = "#8b949e"

var colors = map[string]string{
	// Brand colours substituted for near-black Linguist values.
	"powershell": "#5391fe", // Linguist #012456
	"json":       "#7a869a", // Linguist #292929
	"lua":        "#5b7fff", // Linguist #000080
	"dockerfile": "#2496ed", // Linguist #384d54
	"batchfile":  "#c1f12e",

	// Straight from Linguist.
	"go":           "#00add8",
	"javascript":   "#f1e05a",
	"typescript":   "#3178c6",
	"python":       "#3572a5",
	"shell":        "#89e051",
	"yaml":         "#cb171e",
	"html":         "#e34c26",
	"css":          "#563d7c",
	"scss":         "#c6538c",
	"less":         "#1d365d",
	"vue":          "#41b883",
	"svelte":       "#ff3e00",
	"rust":         "#dea584",
	"c":            "#555555",
	"c++":          "#f34b7d",
	"c#":           "#178600",
	"java":         "#b07219",
	"kotlin":       "#a97bff",
	"swift":        "#f05138",
	"objective-c":  "#438eff",
	"php":          "#4f5d95",
	"ruby":         "#701516",
	"perl":         "#0298c3",
	"r":            "#198ce7",
	"scala":        "#c22d40",
	"haskell":      "#5e5086",
	"elixir":       "#6e4a7e",
	"erlang":       "#b83998",
	"clojure":      "#db5855",
	"dart":         "#00b4ab",
	"groovy":       "#4298b8",
	"makefile":     "#427819",
	"cmake":        "#da3434",
	"nix":          "#7e7eff",
	"hcl":          "#844fba",
	"terraform":    "#844fba",
	"vim script":   "#199f4b",
	"viml":         "#199f4b",
	"markdown":     "#083fa1",
	"tex":          "#3d6117",
	"toml":         "#9c4221",
	"ini":          "#d1dbe0",
	"xml":          "#0060ac",
	"sql":          "#e38c00",
	"plpgsql":      "#336790",
	"assembly":     "#6e4c13",
	"zig":          "#ec915c",
	"solidity":     "#aa6746",
	"astro":        "#ff5a03",
	"handlebars":   "#f7931e",
	"twig":         "#c1d026",
	"blade":        "#f7523f",
	"jsonc":        "#7a869a",
	"text":         Unknown,
	"other":        Unknown,
	"gitignore":    Unknown,
	"editorconfig": Unknown,
}

// Color returns the bar colour for a language, or Unknown if it has no entry.
func Color(name string) string {
	if c, ok := colors[strings.ToLower(strings.TrimSpace(name))]; ok {
		return c
	}
	return Unknown
}
