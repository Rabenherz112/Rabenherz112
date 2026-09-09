package cards

import (
	"github.com/Rabenherz112/Rabenherz112/internal/icons"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// The link row is the only clickable graphic on the profile, so each entry is
// drawn as a bordered tile rather than a bare icon. A loose 20px glyph on the
// page reads as decoration; a tile reads as a button.
//
// Each tile has to be its own image, because a link wraps the whole image and
// there is no way to make one region of an SVG clickable on GitHub.
const (
	TileSize = 38.0
	tileR    = 8.0
	tileIcon = 17.0
)

// A Link is one tile: where it goes, and the mark on it.
type Link struct {
	// Name is the file stem, so the tile lands at cards/link-<name>-<theme>.svg.
	Name string
	URL  string
	Alt  string
	Icon icons.Icon
}

// Links is the row under the header, in display order.
var Links = []Link{
	{Name: "mail", URL: "mailto:rabenherz@theravenhub.com", Alt: "Mail", Icon: icons.Mail},
	{Name: "discord", URL: "https://discord.gg/ySk5eYrrjG", Alt: "Discord", Icon: icons.Discord},
	{Name: "anilist", URL: "https://anilist.co/user/Rabenherz", Alt: "AniList", Icon: icons.AniList},
	{Name: "steam", URL: "https://steamcommunity.com/id/ger-rabenherz", Alt: "Steam", Icon: icons.Steam},
	{Name: "web", URL: "https://theravenhub.com", Alt: "theravenhub.com", Icon: icons.Globe},
}

// Tile draws one link tile: a rounded outline with a centred mark.
//
// The tile carries no text, so it is rendered without the embedded font. That
// keeps each one near a kilobyte instead of the forty-odd the face would add,
// which matters when six of them sit at the top of the page.
func Tile(t theme.Theme, l Link) []byte {
	c := svgx.NewCard(TileSize, t)
	c.StrokeRoundRect(0, 0, TileSize, TileSize, tileR, t.Border)

	// The mark is a hair brighter than body text so the tiles carry some
	// weight of their own without shouting.
	off := (TileSize - tileIcon) / 2
	c.Mark(off, off, tileIcon, l.Icon.ViewBox, l.Icon.Paths, t.Muted)

	return c.RenderBare(TileSize)
}
