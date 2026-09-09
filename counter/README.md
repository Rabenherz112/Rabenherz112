# Profile view counter

`counter.php` serves the "SEEN BY" card at the bottom of the profile: the
number, and three birds trailing off after it to match the flock crossing the
header. It is the only thing on the page that is not generated ahead of time,
because the number has to be current and **fetching the image is what counts
the view**.

## Why it cannot be a card

Everything else on the profile is an SVG built by the workflow and committed, so
GitHub's image proxy can cache it forever. This one has to do the opposite: the
response carries `Cache-Control: no-cache, no-store, must-revalidate` so the
proxy comes back each time. Drop those headers and the count freezes after the
first fetch.

## Why it is one URL and not a `<picture>`

Every other panel on the profile ships as two files and a `<picture>` block
picks one. The counter deliberately does not, because a `<picture>` costs
accuracy: the browser starts fetching the `<img>` fallback before it has
resolved which `<source>` applies, then aborts it. That is two requests for one
view, and whether the aborted one still reaches PHP is a race — locally it did
not, but GitHub's image proxy sits in between and may well complete the fetch it
already started.

So the palette moved inside the file. An SVG loaded through an `<img>` tag still
honours a `prefers-color-scheme` media query, so one image serves both themes
and one view is one request.

`?theme=dark` and `?theme=light` still work, to pin a palette when checking the
output. The README passes neither.

## What it counts

Camo fetching the image, which tracks page views closely but is not the same
thing. GitHub may re-fetch on its own, and a viewer whose browser already has
the image will not be counted again immediately. Every GitHub view counter
works this way; there is no way to do better from inside a README.

## Requirements

**PHP 7.0 or newer, and nothing else.** No Composer, no framework, no database,
no extensions beyond what a default build has. It is one `.php` file and one
`.woff2` font file on ordinary shared hosting.

There is no Docker container and no daemon. The script runs on request, the way
any PHP page does.

mbstring is used when present and falls back to PCRE when it is not, so a
stripped-down build is fine too.

## Deploying

1. Copy `counter.php` and `jetbrains-mono-subset.woff2` into the same directory
   on the web server, at the path the README points to.
2. Make sure the directory in `STORE` is writable by the web server user. It
   defaults to `../var/profile-views.count`, one level **above** the directory
   the script sits in, so the count file is not itself fetchable. The script
   creates the directory on first run if it can.
3. Check it: `curl -sS 'https://your.host/path/' | head -c 200` should return
   an SVG, and the number should go up on each request.

Migrating from the previous counter: put this at the same URL and the README
needs no change. The old endpoint ignores the `?theme=` parameter, so until it
is replaced the profile shows the old badge rather than a broken image.

To carry an existing count over, write it into the count file before the first
request:

```bash
echo 5500 > /path/to/var/profile-views.count
```

## Configuration

Everything is a constant at the top of `counter.php`:

| Constant | What it does |
| --- | --- |
| `STORE` | Where the count is kept. Put it outside the web root. |
| `EMBED_FONT` | Whether to inline the typeface. See below. |
| `LABEL` | The text to the left of the number. |
| `WIDTH` | How wide the card is drawn. 880 matches a full-width card. |

`EMBED_FONT` is the one worth a thought. On it, the card is set in JetBrains
Mono exactly like the others, at the cost of about 8KB on every view, since the
response cannot be cached. Off, it falls back to the viewer's own monospace and
costs nothing, at the price of the number no longer matching the cards above it.

The bird is duplicated here from `internal/cards/flock.go`, because this file is
deployed on its own away from the repository. If the shape changes there, change
`BIRD_PATHS` here too.

`jetbrains-mono-subset.woff2` is cut down to the characters the badge can print,
which is what makes 8KB possible rather than the 31KB the cards carry.
Regenerate it with:

```bash
python tools/gensubset/main.py
```

## Storage

The default is a flat file, incremented under an exclusive `flock` so two
requests arriving together cannot both read the same number and write the same
increment back. That is enough for a profile page and has no dependencies.

If you would rather keep it in MariaDB alongside everything else, replace
`bump_count()` with something like:

```php
function bump_count(): int
{
    $pdo = new PDO('mysql:host=localhost;dbname=utility;charset=utf8mb4', $user, $pass, [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
    ]);
    $pdo->exec("INSERT INTO counters (name, hits) VALUES ('profile', 1)
                ON DUPLICATE KEY UPDATE hits = hits + 1");
    return (int) $pdo->query("SELECT hits FROM counters WHERE name = 'profile'")->fetchColumn();
}
```

Whatever it reads from, keep the two behaviours the current one has: the
increment must be atomic, and a storage failure must return a number rather than
throwing. A counter that cannot write is a much smaller problem than a broken
image on the profile page.

## Security notes

- The only input is `?theme=`, and it is matched against a fixed list rather
  than being echoed anywhere. Nothing from the request reaches the output.
- The count is printed through `htmlspecialchars` with `ENT_XML1`, so even a
  corrupted store cannot inject markup.
- `X-Content-Type-Options: nosniff` is set, and the response is a complete SVG
  document with a fixed `Content-Length`.
