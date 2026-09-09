package cards

import (
	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// KavitaCommand heads the card, and KavitaNote is its top-right caption.
const (
	KavitaCommand = "$ kavita stats"
	KavitaNote    = "self-hosted"
)

const kavitaStatSize = 22.0

// Kavita draws the size of the self-hosted manga library.
func Kavita(t theme.Theme, data model.Kavita) Built {
	c := svgx.NewCard(WidthHalf, t)

	y := labelWithNote(c, PadTop, KavitaCommand, KavitaNote, LabelGap)

	known := data.Known()
	y = statRow(c, y, []stat{
		{Value: figure(known, comma(data.Series)), Label: "SERIES"},
		{Value: figure(known, binarySize(data.Bytes)), Label: "ON DISK"},
		{Value: figure(known, comma(data.Genres)), Label: "GENRES"},
	}, kavitaStatSize)

	return Built{Card: c, Height: y + PadBottom}
}
