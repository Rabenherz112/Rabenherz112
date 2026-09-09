package cards

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// counterPHP is the deployed view counter, which draws the same bird as the
// header flock.
const counterPHP = "../../counter/counter.php"

// TestBirdMatchesTheCounter guards the one piece of artwork this repository
// keeps in two places.
//
// The counter is a PHP file deployed on its own web server, away from this
// repository, so it cannot import the path from here. Duplication is the only
// option, and duplication drifts: changing the bird in Go and forgetting the
// counter would leave the flock crossing the header in one shape and the birds
// beside the visitor count in another.
func TestBirdMatchesTheCounter(t *testing.T) {
	php, err := os.ReadFile(counterPHP)
	if err != nil {
		t.Fatalf("read the counter: %v", err)
	}
	body := string(php)

	for _, d := range BirdPaths {
		if !strings.Contains(body, d) {
			t.Errorf("the counter does not carry this bird path:\n  %s\n"+
				"Update BIRD_PATHS in %s to match internal/cards/flock.go.", d, counterPHP)
		}
	}

	// The box the path is drawn in has to match too, or the same outline comes
	// out at a different aspect ratio.
	for _, want := range []string{
		fmt.Sprintf("const BIRD_W = %.1f;", BirdWidth),
		fmt.Sprintf("const BIRD_H = %.1f;", BirdHeight),
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the counter is missing %q, so it would scale the bird differently", want)
		}
	}
}

// TestFlockFits checks the header flock stays inside the width reserved for it.
// It is placed by measuring back from the right edge, so a bird that overran
// would push into the wordmark rather than being clipped.
func TestFlockFits(t *testing.T) {
	var right float64
	for _, b := range headerFlock {
		right = max(right, b.X+b.Width)
	}
	if right > flockWidth {
		t.Errorf("the flock spans %.0f, wider than the %.0f reserved for it", right, flockWidth)
	}
	if len(headerFlock) < 2 {
		t.Fatal("a flock needs more than one bird")
	}

	// Each bird sits further back than the one before it. That falling size
	// and opacity is what makes the row read as depth rather than as a line of
	// identical stamps.
	for i := 1; i < len(headerFlock); i++ {
		prev, cur := headerFlock[i-1], headerFlock[i]
		if cur.Width >= prev.Width {
			t.Errorf("bird %d is not smaller than the one before it", i)
		}
		if cur.Opacity >= prev.Opacity {
			t.Errorf("bird %d is not fainter than the one before it", i)
		}
	}
}
