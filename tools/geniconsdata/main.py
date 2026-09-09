"""Regenerate internal/icons/icons.go from vendored SVG sources.

Card SVGs are rendered by GitHub inside an <img> context, which blocks every
external resource, so any logo drawn on a card has to ship as path data rather
than a URL. This script flattens the marks we use into a Go source file.

Sources:
  - Simple Icons (CC0), fetched from the simple-icons npm package
  - Twemoji (CC-BY 4.0), for the waving hand in the header
  - Material Symbols (Apache 2.0), for the globe on the website link
  - assets/debian-logo.svg and assets/windows-logo.svg, already in the repo

Usage (from the repo root):
    python tools/geniconsdata/main.py
"""

import os
import re
import subprocess
import sys
import urllib.request

# Marks on the link row under the header. Each becomes a clickable tile.
LINK_ICONS = {
    "gmail": "Mail",
    "discord": "Discord",
    "lemmy": "Lemmy",
    "anilist": "AniList",
    "steam": "Steam",
}

# The toolbox strip on the fleet card.
#
# This is the "Technologies & Tools" table from the previous profile README and
# nothing else. Everything here is something the profile owner actually listed;
# tools that merely appear in the infrastructure are covered by the prose above
# the strip instead, which is where they belong.
#
# Five of the listed tools have no mark to draw. Simple Icons dropped the
# Microsoft ones over trademark policy, which takes out PowerShell, Command
# Prompt, VS Code, Azure and Hyper-V. There is no substitute for any of them.
TOOLBOX_ICONS = {
    # Languages and shells
    "javascript": "JavaScript",
    "gnubash": "GnuBash",
    # Databases
    "mariadb": "MariaDB",
    "mysql": "MySQL",
    "mongodb": "MongoDB",
    # Cloud providers
    "netcup": "Netcup",
    "digitalocean": "DigitalOcean",
    # Code management
    "git": "Git",
    "github": "GitHub",
    "gitlab": "GitLab",
    # Editors
    "nano": "Nano",
    "vim": "Vim",
    # Other
    "docker": "Docker",
    "vmware": "VMware",
}

CDN = "https://cdn.jsdelivr.net/npm/simple-icons@15/icons/{}.svg"
TWEMOJI = "https://cdn.jsdelivr.net/npm/@discordapp/twemoji@15/dist/svg/{}.svg"

# Material Symbols "language", Apache 2.0. Simple Icons carries brand marks
# only, and a personal website needs a generic globe.
GLOBE_PATH = (
    "M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zm6.93 6h-2.95c-.32-1.25-.78-2.45-1.38-3.56 1.84.63 3.37 1.91 4.33 3.56zM12 4.04c.83 1.2 1.48 2.53 1.91 3.96h-3.82c.43-1.43 1.08-2.76 1.91-3.96zM4.26 14C4.1 13.36 4 12.69 4 12s.1-1.36.26-2h3.38c-.08.66-.14 1.32-.14 2 0 .68.06 1.34.14 2H4.26zm.82 2h2.95c.32 1.25.78 2.45 1.38 3.56-1.84-.63-3.37-1.9-4.33-3.56zm2.95-8H5.08c.96-1.66 2.49-2.93 4.33-3.56C8.81 5.55 8.35 6.75 8.03 8zM12 19.96c-.83-1.2-1.48-2.53-1.91-3.96h3.82c-.43 1.43-1.08 2.76-1.91 3.96zM14.34 14H9.66c-.09-.66-.16-1.32-.16-2 0-.68.07-1.35.16-2h4.68c.09.65.16 1.32.16 2 0 .68-.07 1.34-.16 2zm.25 5.56c.6-1.11 1.06-2.31 1.38-3.56h2.95c-.96 1.65-2.49 2.93-4.33 3.56zM16.36 14c.08-.66.14-1.32.14-2 0-.68-.06-1.34-.14-2h3.38c.16.64.26 1.31.26 2s-.1 1.36-.26 2h-3.38z"
)

Q = chr(34)
NL = chr(10)


def collapse(d):
    return re.sub(r"\s+", " ", d).strip()


def paths_of(svg):
    return [collapse(d) for d in re.findall(r'\sd="(.*?)"', svg, re.S)]


def colour_paths_of(svg):
    """Return (fill, d) pairs, for artwork drawn in more than one colour."""
    out = []
    for tag in re.findall(r"<path\b[^>]*/?>", svg, re.S):
        d = re.search(r'\sd="(.*?)"', tag, re.S)
        fill = re.search(r'\sfill="(.*?)"', tag, re.S)
        if not d:
            continue
        out.append(((fill.group(1) if fill else "#000000"), collapse(d.group(1))))
    return out


def viewbox_of(svg):
    m = re.search(r'viewBox="0 0 ([\d.]+) ([\d.]+)"', svg)
    if not m:
        raise SystemExit("no viewBox found")
    return float(m.group(1)), float(m.group(2))


def quote(s):
    # Path data is digits, letters, commas, spaces, dots and minus signs. It
    # never contains a quote or a backslash, so a bare Go string literal is safe.
    if Q in s or chr(92) in s:
        raise SystemExit("path data needs escaping, which this generator does not do")
    return Q + s + Q


def fetch(url):
    req = urllib.request.Request(url, headers={"User-Agent": "geniconsdata"})
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.read().decode("utf-8")


def emit_simple_icons(add, mapping, heading):
    add("// " + heading)
    add("var (")
    for slug, ident in mapping.items():
        svg = fetch(CDN.format(slug))
        ds = paths_of(svg)
        if len(ds) != 1:
            raise SystemExit("expected a single path for " + slug)
        w, h = viewbox_of(svg)
        if (w, h) != (24.0, 24.0):
            raise SystemExit("expected a 24x24 viewBox for " + slug)
        add("// " + ident + " is the " + slug + " mark.")
        add(ident + " = Icon{ViewBox: 24, Paths: []string{")
        add(quote(ds[0]) + ",")
        add("}}")
    add(")")
    add("")


def main():
    root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    os.chdir(root)

    out = []
    add = out.append

    add("// Code generated by tools/geniconsdata. DO NOT EDIT.")
    add("")
    add("// Package icons holds vector path data for the logos drawn onto generated cards.")
    add("//")
    add("// GitHub renders a README card through an <img> tag, which blocks every external")
    add("// resource the SVG might reference, so a logo has to be inlined as path data")
    add("// rather than linked. Brand marks come from Simple Icons (CC0) on a 24x24 grid;")
    add("// the waving hand is Twemoji (CC-BY 4.0); the globe is Material Symbols")
    add("// (Apache 2.0); Debian and Windows come from assets/ and keep their own")
    add("// coordinates.")
    add("package icons")
    add("")
    add("// An Icon is a single-colour mark. Paths are filled with whatever colour the")
    add("// caller passes, in a square coordinate space ViewBox units on a side.")
    add("type Icon struct {")
    add("ViewBox float64")
    add("Paths []string")
    add("}")
    add("")
    add("// A ColorPath is one path of a mark that carries its own colour.")
    add("type ColorPath struct {")
    add("Fill string")
    add("D string")
    add("}")
    add("")
    add("// A ColorIcon is a mark drawn in its own colours rather than the caller's.")
    add("type ColorIcon struct {")
    add("ViewBox float64")
    add("Paths []ColorPath")
    add("}")
    add("")

    emit_simple_icons(add, LINK_ICONS, "Marks for the link row. Simple Icons, CC0. https://simpleicons.org")
    emit_simple_icons(add, TOOLBOX_ICONS, "Marks for the toolbox strip. Simple Icons, CC0.")

    add("// Toolbox is the strip drawn on the fleet card, in display order.")
    add("var Toolbox = []Icon{")
    for ident in TOOLBOX_ICONS.values():
        add(ident + ",")
    add("}")
    add("")

    add("// Globe is the mark on the personal website link. Material Symbols")
    add('// "language", Apache 2.0, because Simple Icons carries brand marks only.')
    add("var Globe = Icon{ViewBox: 24, Paths: []string{")
    add(quote(GLOBE_PATH) + ",")
    add("}}")
    add("")

    # --- Twemoji waving hand ---------------------------------------------
    svg = fetch(TWEMOJI.format("1f44b"))
    w, h = viewbox_of(svg)
    hand = colour_paths_of(svg)
    if not hand:
        raise SystemExit("no paths in the waving hand")
    add("// WavingHand is the waving hand from Twemoji, CC-BY 4.0, by Twitter Inc and")
    add("// other contributors. https://github.com/jdecked/twemoji")
    add("var WavingHand = ColorIcon{ViewBox: " + repr(h) + ", Paths: []ColorPath{")
    for fill, d in hand:
        add("{Fill: " + quote(fill) + ", D: " + quote(d) + "},")
    add("}}")
    add("")

    # --- Debian ----------------------------------------------------------
    with open("assets/debian-logo.svg", encoding="utf-8") as f:
        svg = f.read()
    dw, dh = viewbox_of(svg)
    dpaths = paths_of(svg)
    add("// The Debian swirl, from assets/debian-logo.svg. Trademark of Software in the")
    add("// Public Interest, Inc. The artwork is taller than it is wide, so it is padded")
    add("// out to a square and shifted right by DebianXOffset to sit centred in it.")
    add("const (")
    add("DebianColor = " + quote("#A80030"))
    add("debianWidth = " + repr(dw))
    add("debianHeight = " + repr(dh))
    add(")")
    add("")
    add("// DebianXOffset centres the swirl within its padded square viewBox.")
    add("const DebianXOffset = (debianHeight - debianWidth) / 2")
    add("")
    add("// Debian is the Debian swirl, padded to a square viewBox.")
    add("var Debian = Icon{ViewBox: debianHeight, Paths: []string{")
    for d in dpaths:
        add(quote(d) + ",")
    add("}}")
    add("")

    # --- Windows ---------------------------------------------------------
    with open("assets/windows-logo.svg", encoding="utf-8") as f:
        svg = f.read()
    ww, wh = viewbox_of(svg)
    wpaths = paths_of(svg)
    if len(wpaths) != 1 or ww != wh:
        raise SystemExit("expected a single path on a square viewBox for windows")
    add("// WindowsColor is the Microsoft Windows brand blue.")
    add("const WindowsColor = " + quote("#0078d4"))
    add("")
    add("// Windows is the four-pane Windows logo. Trademark of Microsoft.")
    add("var Windows = Icon{ViewBox: " + repr(wh) + ", Paths: []string{")
    add(quote(wpaths[0]) + ",")
    add("}}")
    add("")

    dst = os.path.join("internal", "icons", "icons.go")
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    with open(dst, "w", encoding="utf-8", newline=NL) as f:
        f.write(NL.join(out) + NL)

    subprocess.run(["gofmt", "-w", dst], check=True)
    print("wrote " + dst + " (" + str(os.path.getsize(dst)) + " bytes)")
    print("link icons: " + str(len(LINK_ICONS)) + ", toolbox: " + str(len(TOOLBOX_ICONS)))
    print("waving hand: " + str(len(hand)) + " paths, viewBox " + str(w) + "x" + str(h))


if __name__ == "__main__":
    sys.exit(main())
