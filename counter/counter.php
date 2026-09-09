<?php
declare(strict_types=1);

/**
 * Profile view counter for the GitHub profile README.
 *
 * Returns an SVG drawn in the same language as the generated cards: no
 * background, a hairline border, and the same type and palette. Unlike those
 * cards this one cannot be generated ahead of time, because the number has to
 * be current and the act of fetching it is what counts the view.
 *
 * Two things make this work on GitHub:
 *
 *   1. The no-cache headers. GitHub proxies README images through its camo
 *      cache, and a cacheable response would be served to everyone after the
 *      first fetch, so the count would stop moving. These headers are the whole
 *      reason the counter is a live endpoint rather than another static card.
 *
 *   2. One URL, not two. The palette switches inside the file on a
 *      prefers-color-scheme media query, which works in an SVG loaded through
 *      an <img> tag. A <picture> with a source per theme would make the browser
 *      speculatively fetch the fallback and abort it, so a single view could
 *      reach the server twice. One image, one request, one count.
 *
 * What is being counted is camo fetching the image, which is close to a page
 * view but not identical: GitHub may re-fetch on its own, and a viewer whose
 * browser has the image cached will not be counted again straight away. Every
 * GitHub view counter has this caveat.
 *
 * Requires PHP 7.0 or newer and nothing outside a default build: no Composer,
 * no extensions beyond PCRE, no database. It is one file plus a font.
 *
 * Deployment: drop this and jetbrains-mono-subset.woff2 in the same directory,
 * make sure the storage path below is writable by the web server, and point the
 * README at it. See counter/README.md.
 */

// ---------------------------------------------------------------- config ---

/**
 * Where the count lives. Put this outside the web root: it has no business
 * being fetchable, and a stray directory listing should not expose it.
 */
const STORE = __DIR__ . '/../var/profile-views.count';

/**
 * The face is embedded so the counter matches the cards exactly. It is about
 * 8KB, paid on every view because the response cannot be cached. Set this to
 * false to fall back to the viewer's own monospace, which costs nothing and
 * looks close enough at this size.
 */
const EMBED_FONT = true;

const FONT_FILE = __DIR__ . '/jetbrains-mono-subset.woff2';

/** The label to the left of the number. */
const LABEL = 'SEEN BY';

/** How wide the badge is drawn, matching a full-width card. */
const WIDTH = 880.0;

// ----------------------------------------------------------------- theme ---

/**
 * The two palettes, matching internal/theme in the generator.
 */
const THEMES = [
    'dark' => [
        'border' => '#30363d',
        'muted'  => '#8b949e',
        'strong' => '#e6edf3',
        'accent' => '#e08a3c',
    ],
    'light' => [
        'border' => '#d1d9e0',
        'muted'  => '#59636e',
        'strong' => '#1f2328',
        'accent' => '#bc6c1f',
    ],
];

/**
 * The class each colour role gets when the card is adapting to the viewer.
 */
const ROLE_CLASS = ['border' => 'b', 'muted' => 'm', 'accent' => 'a'];

/**
 * Reads a pinned theme from the query string, or null to adapt to the viewer.
 *
 * Adapting is the normal case and the README passes no parameter at all. The
 * parameter exists so a specific palette can be forced when checking the
 * output. Anything unrecognised adapts rather than being echoed back, so
 * nothing from the query string reaches the response.
 */
function requested_theme(): ?array
{
    $name = $_GET['theme'] ?? null;
    if (is_string($name) && isset(THEMES[$name])) {
        return THEMES[$name];
    }
    return null;
}

/**
 * Returns the attributes that colour one element.
 *
 * When adapting, the element gets a class that the stylesheet switches on the
 * viewer's scheme, plus a presentation attribute holding the light value. CSS
 * beats a presentation attribute, so the class wins wherever the stylesheet
 * applies and the attribute is what shows if it somehow does not. When a theme
 * is pinned there is nothing to switch, so only the attribute is written.
 */
function paint(?array $theme, string $role, string $prop): string
{
    if ($theme !== null) {
        return ' ' . $prop . '="' . $theme[$role] . '"';
    }
    return ' class="' . ROLE_CLASS[$role] . '" ' . $prop . '="' . THEMES['light'][$role] . '"';
}

/**
 * The rules that repaint the card on a dark page.
 *
 * This is why the counter is a single URL rather than a <picture> with a source
 * per theme. A <picture> makes the browser start fetching the <img> fallback
 * before it resolves which source to use, and abort it once it knows: two
 * requests for one view, and whether the aborted one still reaches the server
 * is a race. Counting has to be exact, so the palette moves inside the file and
 * the README asks for one image.
 */
function scheme_style(): string
{
    $d = THEMES['dark'];
    return '@media(prefers-color-scheme:dark){'
        . '.' . ROLE_CLASS['border'] . '{stroke:' . $d['border'] . '}'
        . '.' . ROLE_CLASS['muted'] . '{fill:' . $d['muted'] . '}'
        . '.' . ROLE_CLASS['accent'] . '{fill:' . $d['accent'] . '}}';
}

// --------------------------------------------------------------- storage ---

/**
 * Increments the stored count and returns the new value.
 *
 * The file is opened once and held under an exclusive lock for the whole
 * read-modify-write, so two requests arriving together cannot both read the
 * same number and write the same increment back.
 *
 * A failure here returns the last known count rather than throwing: a counter
 * that cannot write is a much smaller problem than a profile page with a broken
 * image on it.
 */
function bump_count(): int
{
    $dir = dirname(STORE);
    if (!is_dir($dir)) {
        @mkdir($dir, 0775, true);
    }

    $fh = @fopen(STORE, 'c+');
    if ($fh === false) {
        return 0;
    }

    try {
        if (!flock($fh, LOCK_EX)) {
            return 0;
        }
        $raw = stream_get_contents($fh);
        $count = max(0, (int) trim((string) $raw)) + 1;

        rewind($fh);
        ftruncate($fh, 0);
        fwrite($fh, (string) $count);
        fflush($fh);
        flock($fh, LOCK_UN);

        return $count;
    } finally {
        fclose($fh);
    }
}

// ---------------------------------------------------------------- drawing ---

/**
 * JetBrains Mono advances every glyph by the same 0.6em, which is what lets the
 * layout below be arithmetic rather than guesswork. The generator relies on the
 * same number; see internal/svgx.
 */
const ADVANCE = 0.6;

function text_width(string $s, float $size, float $spacing = 0.0): float
{
    return (float) char_count($s) * ($size * ADVANCE + $spacing);
}

/**
 * Counts characters rather than bytes.
 *
 * mbstring is usually present but is not part of a default PHP build, and this
 * file is meant to drop onto whatever shared hosting is to hand. The PCRE
 * fallback counts UTF-8 code points and is always available.
 */
function char_count(string $s): int
{
    if (function_exists('mb_strlen')) {
        return (int) mb_strlen($s, 'UTF-8');
    }
    $n = preg_match_all('/./u', $s);
    return $n === false ? strlen($s) : $n;
}

/**
 * Groups a number the way the cards do: 96301 becomes 96,301.
 */
function format_count(int $n): string
{
    return number_format($n, 0, '.', ',');
}

/**
 * Returns the @font-face rule, or an empty string when the face is not being
 * embedded or the file is missing.
 */
function font_style(): string
{
    if (!EMBED_FONT || !is_readable(FONT_FILE)) {
        return '';
    }
    $data = @file_get_contents(FONT_FILE);
    if ($data === false) {
        return '';
    }
    return "@font-face{font-family:'JBM';src:url(data:font/woff2;base64,"
        . base64_encode($data)
        . ")format('woff2');font-weight:400 800;font-style:normal}";
}

/**
 * The bird from the header flock: a crow in side profile, gliding right, in a
 * 26 x 16 box. It is repeated here rather than shared because this file is
 * deployed on its own, away from the repository; if the shape changes in
 * internal/cards/flock.go, change it here too.
 */
const BIRD_W = 26.0;
const BIRD_H = 16.0;
const BIRD_PATHS = [
    'M25.8 7.9C24.6 7.2 23.5 6.7 22.4 6.4 21.6 6.1 20.7 6 19.8 6.05 18.3 6.15 16.9 6.5 15.6 7.05 13.1 4.2 10 1.8 6.3 0 8.5 3.2 10.8 6.1 13.3 8.6 10.3 8.8 7.3 9.05 4.3 9.35 2.9 9.5 1.4 9.65 0 9.85 1.5 10.65 3.1 11.25 4.8 11.65 8.2 12.45 11.8 12.55 15.2 11.95 17.6 11.55 19.8 10.65 21.7 9.35 23.1 8.95 24.5 8.5 25.8 7.9Z',
];

/** Three birds trailing off after the number, matching the header flock. */
const BIRDS = [
    ['w' => 19.0, 'dy' => 0.0, 'o' => 0.50],
    ['w' => 15.0, 'dy' => 5.0, 'o' => 0.34],
    ['w' => 12.0, 'dy' => 1.0, 'o' => 0.20],
];

const BIRD_GAP = 9.0;

/**
 * Draws one bird, scaled to width and placed at (x, y).
 */
function bird(float $x, float $y, float $width, float $opacity, string $paint): string
{
    $scale = round($width / BIRD_W, 6);
    $svg = '<g transform="translate(' . round($x, 2) . ' ' . round($y, 2)
         . ') scale(' . $scale . ')"' . $paint . ' opacity="' . $opacity . '">';
    foreach (BIRD_PATHS as $d) {
        $svg .= '<path d="' . $d . '"/>';
    }
    return $svg . '</g>';
}

/**
 * Draws the counter.
 *
 * The card paints no background of its own. GitHub has four themes but only
 * reports whether the page is light or dark, so leaving it transparent is what
 * makes it sit correctly on the dimmed and high-contrast ones too.
 */
function render(int $count, ?array $theme): string
{
    $labelSize    = 11.0;
    $labelSpacing = $labelSize * 0.18;
    $valueSize    = 22.0;
    $height       = 62.0;

    $value  = format_count($count);
    $labelW = text_width(LABEL, $labelSize, $labelSpacing);
    $valueW = text_width($value, $valueSize);

    $birdsW = 0.0;
    foreach (BIRDS as $i => $b) {
        $birdsW += $b['w'] + ($i > 0 ? BIRD_GAP : 0.0);
    }

    // The label, the number and the flock are centred together as one group.
    $gap   = 18.0;
    $total = $labelW + $gap + $valueW + $gap + $birdsW;
    $x     = round((WIDTH - $total) / 2, 2);

    // Baselines are set from the font's own cap height, so the label and the
    // much larger number sit on a common optical centre line.
    $mid           = $height / 2;
    $labelBaseline = round($mid + $labelSize * 0.73 / 2, 2);
    $valueBaseline = round($mid + $valueSize * 0.73 / 2, 2);

    $style = font_style()
        . "text{font-family:'JBM',ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}";
    if ($theme === null) {
        $style .= scheme_style();
    }

    $svg  = '<svg xmlns="http://www.w3.org/2000/svg" width="' . WIDTH . '" height="' . $height
          . '" viewBox="0 0 ' . WIDTH . ' ' . $height . '" fill="none" role="img"'
          . ' aria-label="Seen by ' . htmlspecialchars($value, ENT_QUOTES | ENT_XML1) . ' visitors">';
    $svg .= '<defs><style>' . $style . '</style></defs>';
    $svg .= '<rect x="0.5" y="0.5" width="' . (WIDTH - 1) . '" height="' . ($height - 1)
          . '" rx="6" fill="none"' . paint($theme, 'border', 'stroke') . '/>';

    $svg .= '<text x="' . $x . '" y="' . $labelBaseline . '" font-size="' . $labelSize
          . '" letter-spacing="' . round($labelSpacing, 2) . '"' . paint($theme, 'muted', 'fill') . '>'
          . htmlspecialchars(LABEL, ENT_QUOTES | ENT_XML1) . '</text>';
    $x += $labelW + $gap;

    $svg .= '<text x="' . round($x, 2) . '" y="' . $valueBaseline . '" font-size="' . $valueSize
          . '"' . paint($theme, 'accent', 'fill') . '>'
          . htmlspecialchars($value, ENT_QUOTES | ENT_XML1) . '</text>';
    $x += $valueW + $gap;

    // The flock is centred on the label's line, each bird a little smaller and
    // fainter than the last, the way it crosses the header.
    foreach (BIRDS as $i => $b) {
        $h = $b['w'] * BIRD_H / BIRD_W;
        $svg .= bird($x, $mid - $h / 2 + $b['dy'] - 2, $b['w'], $b['o'], paint($theme, 'muted', 'fill'));
        $x += $b['w'] + BIRD_GAP;
    }

    $svg .= '</svg>';

    return $svg;
}

// ----------------------------------------------------------------- output ---

$theme = requested_theme();
$count = bump_count();
$body  = render($count, $theme);

header('Content-Type: image/svg+xml; charset=utf-8');

// Without these the response is cached by GitHub's image proxy and the count
// stops moving after the first fetch.
header('Cache-Control: no-cache, no-store, must-revalidate, max-age=0');
header('Pragma: no-cache');
header('Expires: 0');

// The response is a complete document and is not meant to be framed or sniffed
// as anything else.
header('X-Content-Type-Options: nosniff');
header('Content-Length: ' . strlen($body));

echo $body;
