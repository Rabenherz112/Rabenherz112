"""Build the small font subset the view counter embeds.

The counter is served live by counter/counter.php on every profile view, and
unlike the cards it is deliberately uncacheable, so its payload is paid every
time. The full Latin subset the cards carry is 31KB, which is far too much for
one short line of text, so this cuts it down to the characters the counter can
actually print.

Requires fonttools and brotli:
    pip install fonttools brotli

Usage (from the repo root):
    python tools/gensubset/main.py
"""

import os
import sys

# Everything the counter might render: its label, a formatted number, and
# enough of the alphabet that the label can be reworded without regenerating.
CHARS = (
    "abcdefghijklmnopqrstuvwxyz"
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    "0123456789"
    " $.,:;!?()[]{}<>/\\|-_+=@#%&*'\"~^"
    "·"  # the middle dot used as a separator elsewhere in the design
    "—"  # the em dash that stands in for a figure with no data
    "…"  # the ellipsis used when a run is truncated
)

SRC = os.path.join("internal", "fonts", "JetBrainsMono-latin.woff2")
DST = os.path.join("counter", "jetbrains-mono-subset.woff2")


def main():
    root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    os.chdir(root)

    try:
        from fontTools import subset
        from fontTools.ttLib import TTFont
    except ImportError:
        raise SystemExit("fonttools is required: pip install fonttools brotli")

    font = TTFont(SRC)

    options = subset.Options()
    options.flavor = "woff2"
    options.desubroutinize = True
    # The counter uses one weight, so the variable axis is pinned rather than
    # carried along.
    options.layout_features = ["kern"]

    subsetter = subset.Subsetter(options=options)
    subsetter.populate(text=CHARS)
    subsetter.subset(font)

    os.makedirs(os.path.dirname(DST), exist_ok=True)
    font.flavor = "woff2"
    font.save(DST)

    before = os.path.getsize(SRC)
    after = os.path.getsize(DST)
    print("wrote " + DST)
    print("  %d bytes, down from %d (%.0f%% smaller)" % (after, before, 100 * (1 - after / before)))
    print("  %d characters" % len(set(CHARS)))


if __name__ == "__main__":
    sys.exit(main())
