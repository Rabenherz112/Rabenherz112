// Package fonts carries the typeface that ships inside every card.
//
// A card is loaded by GitHub through an <img> tag and proxied by camo, so an
// @font-face pointing at fonts.gstatic.com never resolves: the request is
// blocked and the card falls back to whatever monospace the viewer happens to
// have. Since the layout is computed from JetBrains Mono's own metrics, that
// fallback would misplace every run. So the font travels with the file.
//
// The embedded file is the Latin subset Google Fonts serves (U+0000-00FF plus
// general punctuation and the common symbols), which keeps it to ~31KB while
// covering any Latin text the APIs return. It is a variable font with a weight
// axis from 400 to 800, so one file covers every weight the design uses.
package fonts

import (
	_ "embed"
	"encoding/base64"
	"sync"
)

//go:embed JetBrainsMono-latin.woff2
var jetBrainsMono []byte

var (
	once    sync.Once
	encoded string
)

// JetBrainsMonoBase64 returns the font encoded for a data URI. Every card
// embeds the same bytes, so the encoding is done once and shared.
func JetBrainsMonoBase64() string {
	once.Do(func() {
		encoded = base64.StdEncoding.EncodeToString(jetBrainsMono)
	})
	return encoded
}
