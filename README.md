<!--
  This file is the hand-written shell of the profile. The panels are SVGs under
  cards/, rebuilt by .github/workflows/cards.yml; do not edit those by hand.

  Each panel is placed twice, in a <picture> block, so GitHub can swap the dark
  and light variant per viewer. The cards paint no background of their own, so
  they also sit correctly on the Dark dimmed and Dark high contrast themes,
  which prefers-color-scheme cannot tell apart from the default dark.

  Two layout rules hold the page square, and both are easy to undo by accident:

  Every panel is wrapped in a <p>. Markdown gives a paragraph a 16px bottom
  margin and gives a bare <picture> nothing at all, so a panel written without
  the wrapper sits about 6px from the next one while its neighbours sit 22px
  apart. The wrapper is the whole of what keeps the vertical rhythm even.

  A row of two panels is written on one line with no space anywhere inside it,
  and each half is 50% wide. Images are inline, so any newline or space between
  the two <picture> tags -- or between an <img> and its closing </picture> --
  becomes a real 4px space on the line. That space is why the halves were 49%
  for as long as this was written across several lines: at 50% they no longer
  fit, and the second one wrapped onto a line of its own. The cost was a row
  12px narrower than every full-width card around it. The gap between the two
  boxes is transparent padding inside the images instead, which is what
  internal/cards.RenderPair is for.

  The link tiles and the view counter are the only things here that are not
  flat images baked at build time. Each tile is its own image because a link
  has to wrap the whole of one.

  The only generated part of this file is the date line between the
  "generated:start" and "generated:end" markers at the bottom. It is currently
  written commented out; see stampVisible in cmd/gencards/main.go.
-->

<p>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./cards/header-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./cards/header-light.svg">
  <img alt="Rabenherz, aka Lp_Zombie. IT specialist at Serviceware SE; self-hoster, awesome-selfhosted reviewer, keeper of a large manga library." src="./cards/header-dark.svg" width="100%">
</picture>
</p>

<p align="center">
  <a href="mailto:rabenherz@theravenhub.com" title="Mail"><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/link-mail-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/link-mail-light.svg"><img alt="Mail" src="./cards/link-mail-dark.svg" width="38" height="38"></picture></a>
  <a href="https://discord.gg/ySk5eYrrjG" title="Discord"><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/link-discord-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/link-discord-light.svg"><img alt="Discord" src="./cards/link-discord-dark.svg" width="38" height="38"></picture></a>
  <a href="https://lemmy.world/u/Rabenherz112" title="Lemmy"><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/link-lemmy-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/link-lemmy-light.svg"><img alt="Lemmy" src="./cards/link-lemmy-dark.svg" width="38" height="38"></picture></a>
  <a href="https://anilist.co/user/Rabenherz" title="AniList"><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/link-anilist-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/link-anilist-light.svg"><img alt="AniList" src="./cards/link-anilist-dark.svg" width="38" height="38"></picture></a>
  <a href="https://steamcommunity.com/id/rabenherz" title="Steam"><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/link-steam-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/link-steam-light.svg"><img alt="Steam" src="./cards/link-steam-dark.svg" width="38" height="38"></picture></a>
  <a href="https://theravenhub.com" title="theravenhub.com"><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/link-web-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/link-web-light.svg"><img alt="theravenhub.com" src="./cards/link-web-dark.svg" width="38" height="38"></picture></a>
</p>

<p>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./cards/fleet-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./cards/fleet-light.svg">
  <img alt="The fleet runs Debian on every server, Pi and container; Windows is the daily driver at work and at home. The toolbox: JavaScript, Bash, MariaDB, MySQL, MongoDB, Netcup, DigitalOcean, Git, GitHub, GitLab, Nano, Vim, Docker and VMware." src="./cards/fleet-dark.svg" width="100%">
</picture>
</p>

<!-- One line, no spaces: see the note at the top of this file. -->
<p><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/languages-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/languages-light.svg"><img alt="Share of coding time per language over the last twelve months." src="./cards/languages-dark.svg" width="50%"></picture><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/wakatime-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/wakatime-light.svg"><img alt="Coding time over the last fourteen days, charted by day of the week." src="./cards/wakatime-dark.svg" width="50%"></picture></p>

<p>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./cards/activity-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./cards/activity-light.svg">
  <img alt="The ten most recent public reviews and pull requests." src="./cards/activity-dark.svg" width="100%">
</picture>
</p>

<p>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./cards/anilist-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./cards/anilist-light.svg">
  <img alt="AniList manga totals: series tracked, chapters read, volumes read and mean score, with the series currently being read." src="./cards/anilist-dark.svg" width="100%">
</picture>
</p>

<!-- One line, no spaces: see the note at the top of this file. -->
<p><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/steam-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/steam-light.svg"><img alt="Games played in the last two weeks, with hours." src="./cards/steam-dark.svg" width="50%"></picture><picture><source media="(prefers-color-scheme: dark)" srcset="./cards/kavita-dark.svg"><source media="(prefers-color-scheme: light)" srcset="./cards/kavita-light.svg"><img alt="The self-hosted manga library: series, size on disk and genre count." src="./cards/kavita-dark.svg" width="50%"></picture></p>

<!--
  The counter is the one thing on this page that is fetched live rather than
  generated: being fetched is what counts the view, so it cannot be cached and
  cannot be baked into a card. Source in counter/.

  It is a bare <img> on purpose, where every other panel here uses <picture>.
  A <picture> makes the browser start fetching the <img> fallback before it
  resolves which <source> to use and abort it once it knows, so one view can
  reach the server twice and the count drifts up. This image switches palette
  inside itself instead, on a prefers-color-scheme media query.

  A bare <img> on its own line is already a paragraph to markdown, so it needs
  no <p> of its own to be spaced like the panels above it.
-->
<img alt="Profile views" src="https://utility.theravenhub.com/scripts/github-pf-counterv2/" width="100%">

<!-- generated:start -->
<!-- <p align="center"><sub>generated 2026-09-12 · rebuilt daily by <code>.github/workflows</code></sub></p> -->
<!-- generated:end -->
