# Umineko City of Books

<p align="center">
  <img src="https://waifuvault.moe/f/2076c25a-2637-45af-bd6b-bc3a9d4d9a45/featherine%20augustus%20aurora.png" alt="Umineko City of Books" width="900">
</p>

<p align="center">
  <sub>
    Artwork by <a href="https://m.twitch.tv/meru">Meru</a>
  </sub>
</p>

A community platform for fans of Umineko no Naku Koro ni, Higurashi, Ciconia, and the wider When They Cry series. The original goal was a place to declare fan theories as **blue truth**, attach quotes from the game as evidence, and have them debated on two sides: **"With love, it can be seen"** and **"Without love, it cannot be seen"**. It has since grown into a full social platform: theory debates across all three series, a Twitter-style game board, mystery boards, fan art galleries, ship and OC declarations, fanfiction, live reading journals, chat rooms with shared-browser watch parties, DMs, secret unlock hunts, multiplayer games (chess, checkers, othello, minesweeper) with their own vanity titles, site-wide search, live notifications, and themed role-based moderation.

## Table of Contents

- [Features](#features)
  - [Theory Debates](#theory-debates)
  - [Mysteries](#mysteries)
  - [Gallery and Art](#gallery-and-art)
  - [Ships](#ships)
  - [Original Characters](#original-characters)
  - [Game Board](#game-board)
  - [Fanfiction](#fanfiction)
  - [Reading Journals](#reading-journals)
  - [Chat Rooms and DMs](#chat-rooms-and-dms)
  - [Chatbots](#chatbots)
  - [Watch Parties](#watch-parties)
  - [Voice Chat](#voice-chat)
  - [Live Streaming](#live-streaming)
  - [Games](#games)
  - [Secrets and Unlock Hunts](#secrets-and-unlock-hunts)
  - [Announcements](#announcements)
  - [Suggestions](#suggestions)
  - [Search](#search)
  - [Quote Browser](#quote-browser)
  - [Profiles and Social Graph](#profiles-and-social-graph)
  - [Notifications](#notifications)
  - [Stream Overlay](#stream-overlay)
  - [Moderation and Admin](#moderation-and-admin)
  - [Platform Features](#platform-features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [Database and Migrations](#database-and-migrations)
- [Development Workflow](#development-workflow)
- [Deployment](#deployment)
- [Adding a New Page](#adding-a-new-page)
- [License](#license)

## Features

### Theory Debates

The original heart of the site. Submit a fan theory as a **blue truth**, attach quote evidence, and let others refute or support it.

- Theory declarations with title, body, and episode / arc / chapter scope depending on series
- Evidence attachment by searching any quote (including narrator lines) across Umineko, Higurashi, and Ciconia via the Umineko Quote Finder API, with per-series chapter/episode/arc filters and a main/additional character split for Higurashi and Ciconia
- Two-sided debate with **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny), each with its own evidence
- **Credibility score** per theory (0 to 100), weighted by the truth type of evidence attached to top-level responses (gold 3.3, red 3.0, purple 2.2, blue 2.0, plain quote 1.0) and run through a `tanh` curve, so a lopsided debate saturates towards 0 or 100 instead of running away
- Threaded replies on responses with flat rendering and @username attribution
- Upvotes and downvotes on both theories and responses, separate from the credibility score
- Per-series feeds at `/theories` (Umineko), `/theories/higurashi`, and `/theories/ciconia`, each with its own sidebar entry

### Mysteries

A gamified puzzle mode where a user (the Game Master) poses a mystery with graduated clues, and other players submit attempts.

- Create mysteries with difficulty, body, and an ordered list of clues tagged by truth type (red/blue/gold/purple), plus optional inline images and downloadable document attachments
- Clues can be posted to everyone or addressed to **one player as a private red truth**, which only that player sees and which notifies them directly. Clues stay editable and removable after the fact
- Attempts are threaded with reply chains. Outside the Game Master, you can only reply inside your own attempt's thread, so players never talk over each other's solves
- Upvotes and downvotes on attempts
- Game master marks the winning attempt, which pins to the top of the page, and can later mark the mystery **permanently solved** so it closes for good
- **Pause** a mystery to stop new attempts while you are mid-adjudication, or flag yourself **away** as the Game Master. Both broadcast live and notify everyone playing
- Piece counter showing how many players have attempted
- Two **leaderboards**: top solvers (which awards the **True Detective** pill) and top game masters (**Game Master**)
- Role-based visibility: the mystery's own **Game Master**, and any **super admin**, sees attempts grouped by player with collapsible groups, player pills, and red-dot unread indicators backed by a localStorage read cursor. Admins, mods, and regular players see the normal flat thread view
- Real-time updates over WebSocket when new attempts, replies, clues, or status changes land
- Separate notification categories: **Mysteries (as Game Master)** and **Mysteries (as Player)**
- Full rich-text formatting in mystery bodies and attempts (backticks, quotes, spoilers, syntax-highlighted code fences, truth colours)

### Gallery and Art

Fan art uploads with full social features.

- Upload an image typed as **Drawing**, **Cosplay**, **Figure**, or **Other**, with a corner, description, and up to ten tags. Images only here, video lives on the game board
- Automatic WebP conversion and thumbnail generation
- **Spoiler flag** per piece: the image renders blurred behind a click-to-reveal cover everywhere it appears, in the grid, on the detail page, and before the lightbox opens
- **Galleries**: bundle related art into named collections with cover image and preview strip. The index has two views, one grouping collections by artist and one listing every individual piece
- Search, a type filter, tag browsing with popular tag listings per corner, and sort by New / Popular / Most Viewed
- Full comment system with threading, media uploads, embeds, likes, GIFs, and Discord-style formatting
- **@Mentions** in a description notify the people named
- Lightbox viewer for full-size images
- View counts unique per viewer (hashed user ID or IP)
- Per-corner filtering (Umineko, Higurashi, Ciconia), plus a per-user daily upload cap (admin configurable)

### Ships

Declare character pairings and rally votes for them.

- Pick characters from Umineko, Higurashi, or Ciconia via a character picker, or attach one of your own original characters (either chosen from your OC list or typed in by name)
- Mixed-series ships are supported
- Optional ship image with automatic WebP conversion and lightbox viewer
- Upvote and downvote each ship, sorted by popularity
- Ships whose score falls to **-3 or below** automatically get the **Crackship** badge, and a toggle narrows the list to crackships only
- Inline edit form on the ship detail page for authors and admins
- Full comment system with threading, media, GIFs, and likes
- Filter by series (Umineko, Higurashi, OC)
- Sort modes: newest, oldest, most upvoted, crackship (lowest score first), most controversial, most commented

### Original Characters

A dedicated home for player-created OCs, separate from the canon-character ship list.

- Create OCs with name, series tag (Umineko / Higurashi / Ciconia / Custom), optional custom-series label, description, and avatar with automatic WebP conversion
- **Per-OC gallery**: add extra images with captions and ordering, shown as a masonry grid with lightbox on the detail page
- Upvote and downvote OCs, and **favourite** them with a public count
- OCs whose score falls to **-3 or below** pick up the **Crack OC** badge, and the index can be narrowed to those alone
- Browse the OC index at `/oc` filtered by series (including Custom) or owner, sorted by newest, oldest, most upvoted, most favourited, most commented, or name
- OC summaries appear on the owner's profile alongside their ships, and OCs can be attached to ships and tagged in fanfics through the shared character picker
- Full comment system with threading, media, GIFs, and likes
- Edit and delete by owners and admins

### Game Board

A Twitter-style social feed for off-topic posts and discussion.

- Posts with title, body, multiple images or video, likes, threaded comments
- **Corners**: dedicated sub-feeds for **Umineko**, **Higurashi**, **Ciconia**, **Higanbana**, and **Rose Guns Days**, each with its own post count, content rules, and sitemap
- **@Mentions** with autocomplete in posts and comments, mentioned users get notified. Mentioning a character member pulls it into the thread (see [Chatbots](#chatbots))
- **Link embeds**: YouTube links embed inline, other URLs render rich OG preview cards (title, image, description, site name). Embeds refresh daily
- **Polls** on posts with multi-option voting, per-user vote tracking, and optional expiry
- **GIF picker** on the post composer (and every comment box) backed by GIPHY, sends instantly on pick
- **Quick Reply**: one-click Reply button drops an inline comment composer under the post without leaving the feed, auto-collapses after send
- **Live comments**: a new comment is pushed over WebSocket and appears on the post page as it is written, with no refresh
- Relevance-based feed algorithm with deterministic jitter for stable pagination, plus New / Most Liked / Most Replies / Most Viewed sorts. Your chosen sort is remembered on your profile
- Following tab showing only posts from users you follow
- Unique post view counts
- Live like counters pushed over WebSocket
- Comment media uploads (images and video) with the shared MediaPicker component
- Editable posts and comments with an "(edited)" marker and notification to commenters

### Fanfiction

Write and publish multi-chapter fan stories.

- Fanfic entries with title, summary, language, series tag (Umineko / Higurashi / Ciconia / OC), cover image, and character tagging across all three series
- **FFN-style metadata**: content rating (K / K+ / T / M), status (in progress or complete), one or two genres, free-form tags, and oneshot / pairing / contains-lemons flags
- **Chapter-based structure**: add, reorder, edit, and delete chapters individually. A fic can be kept as a draft until it is ready
- **Rich text editor** (TipTap) for chapter bodies: bold, italic, strike, headings, blockquotes, bullet/ordered lists, horizontal rule, text alignment (left/centre/right), colour swatches, and links
- **Server-side HTML sanitisation** (bluemonday UGCPolicy) on every write, plus client-side DOMPurify before render, so the full Tiptap toolbar survives but `<script>`, event handlers, `javascript:` URLs, iframes, and SVG payloads are stripped
- Per-fic and per-chapter **word count**, **view count**, **favourite count**, and a remembered **reading position** so you can pick up where you left off
- Favourite fanfics to follow new chapters
- Browse with filters for series, rating, status, language, two genres, a tag, and up to four characters at once
- Full comment system with threading, media, GIFs, and likes on both the fanfic and individual chapters
- Per-fanfic sitemap inclusion

### Reading Journals

Live-blog your read-throughs of Ryukishi07's works. Post reactions, theories, and predictions as you go.

- Create a journal tied to a work: General, Umineko, Higurashi, Ciconia, Higanbana, or Rose Guns Days
- Each entry is its own numbered page with prev/next navigation, a word count, media, mentions, and Discord-style formatting
- **Drafts**: save an entry privately and publish it later. Drafts stay out of the entries list, the sitemap, search, and the activity feed, and do not notify anyone until you publish
- Threaded comment system so other players can react to each entry without spoiling
- Follow a journal to get notified when the author posts a new update, and browse a feed of just the journals you follow
- Journals **auto-archive after seven days** without author activity, to keep the index clean. An archived journal is read-only until its author posts again
- Per-user daily journal cap (admin configurable)

### Chat Rooms and DMs

Real-time chat in two flavours: one-to-one direct messages and named group rooms.

- **Direct Messages** with unread counts, last-read cursors, and per-user enable/disable toggle in profile settings
- **Deleting a DM is per person**. Your copy disappears and the other party keeps theirs. Messaging that person again quietly puts you back in the same conversation, but your side starts from the moment you rejoined, so nothing either of you wrote is destroyed and nothing you deleted comes back. Group rooms are unaffected: leave and rejoin one and you still see the full history
- **Chat Rooms**: public or private group rooms with tags, an optional **Roleplay** flag that switches the room into a different visual and posting style, and a hot score that surfaces busy rooms. Rooms with no messages for seven days are marked archived and drop out of the default listing
- **Emoji reactions** on messages with live count and "you reacted" state, shown across desktop and mobile
- **Pinning**: moderators and room owners can pin messages; a dedicated pinned messages panel surfaces them
- **Message search** over one room or every room you are in
- **Room profiles**: a nickname and an avatar scoped to that room only. Site moderators can set someone's nickname and lock it, or unlock it again
- **Member management**: per-room roles, kick, and **timeouts** measured in seconds, hours, weeks, years, decades or centuries. A timeout set by site staff cannot be lifted by a room host, and site staff cannot be timed out or kicked at all
- **Invites** are open to the host and to any site moderator or admin, not just the host
- **Ghost members**: site staff can join a public room invisibly. Ghosts are hidden from the member list and their joins and leaves are announced only to other staff
- **Per-room bans** that stick. Banned users cannot rejoin, send, read, list members, or see the room in their list. Available to the room host, site moderators, and admins. Banned targets receive a live WS kick event plus an optional reason.
- **Banned-words filter** with two scopes and three match modes:
  - **Global** rules (admin `/admin/banned-words`) apply to every chat room
  - **Local** rules (per-room moderation dialog, open to host + mods + admins) apply only to that room and see global rules read-only
  - Match mode `Substring` / `Whole word` / `Regex` plus a `Case sensitive` toggle; regex syntax validated on save
  - Action `Delete message` rejects the send with an inline error; action `Kick` also evicts the sender from the room (they can rejoin; a ban is a separate, intentional moderator action)
  - Room hosts, site moderators, admins, superadmins, and character members are immune. Automated hits log to the audit trail with a NULL actor ("System")
  - Rules are editable: pattern, mode, case, and action can all be changed after creation, and the change takes effect immediately
  - Edits run the filter too, so a message cannot be posted harmlessly and then rewritten
- **Configurable limit**: max room members is site-settings driven (`max_chat_room_members`, default 100), checked on both joining and inviting
- **Replies and edits** on individual messages, with a floating action bar above the bubble on hover
- **Mute notifications** per room, so a busy room stops pinging you without leaving it
- **GIF picker**, emoji picker, media uploads, and full Discord-style text formatting (backticks, quotes, spoilers, syntax highlighting) everywhere text is typed
- WebSocket-driven real-time delivery, pin/unpin events, reaction updates, and typing presence
- **Character members** can be invited into a room like anyone else and answer when mentioned or replied to (see [Chatbots](#chatbots))
- **Watch parties** launchable from any group room: share a remote Hyperbeam browser VM or broadcast your own screen, with optional in-party voice, for everyone in the room to watch together (see [Watch Parties](#watch-parties))
- **Voice chat** in group rooms, DMs, and watch parties via a self-hosted LiveKit SFU, shown as a slim in-room bar so you can talk and chat at once, with local and moderator mute controls (see [Voice Chat](#voice-chat))
- Mobile-first composer: full-width text box with Media / GIF / Send stacked below, bubbles spanning edge to edge

### Chatbots

Character accounts that answer in their own voice, backed by an OpenAI model. Each bot is a real user row with a username, display name, avatar, and the system **Bot** vanity role, so it can be @mentioned, replied to, and opened as a profile like anyone else. Managed at `/admin/chatbots`, tuned under **Admin → Settings → Chatbot**, and completely dormant until an API key is saved.

- **Where they answer**: @mention or reply to a bot in a group chat room, @mention or reply to one on the **Game Board** (both posts and comments), or just open a DM with one. In a DM every message summons the bot, no mention needed
- **Per-bot persona**: system prompt, model, reasoning effort, verbosity, and max output tokens are set per bot. Leave any of them blank and that bot inherits the site default, so one character can reason harder or talk longer than the rest
- **Conversation context**: a DM replays the last `chatbot_context_messages` messages of that thread, a threaded reply walks up to `chatbot_max_reply_chain` parents, and the assembled prompt is trimmed from the oldest end to stay inside a character budget. The bot's own past messages come back as assistant turns; everyone else's are labelled with the speaker and, in chat, the line they were replying to
- **Opt-in permission**: answering is gated on the `use_chatbot` permission. Turn on `chatbot_require_permission` and nominate a vanity role that carries it, and users grant themselves that role from **Settings → Characters**. Someone who summons a bot without it gets one polite refusal pointing at their settings page, at most once every ten minutes, rather than silence. Changing the nominated role migrates every existing holder onto the new one in the background
- **Throttles**: a per-user reply cooldown (skipped in DMs), per-user and site-wide daily invocation caps, one reply in flight per room or post, and a bounded worker queue that drops the trigger rather than backing up. Every drop is counted by reason
- **Prompt caching**: the system prompt is sent as a cacheable prefix with an explicit breakpoint and the bot's own user ID as the cache key, so repeat turns for the same character bill at the cached rate. The summoning user reaches the provider only as a salted hash, used as the safety identifier
- **Presence and typing**: enabled bots always read as online, and a bot emits the same `typing` event as a human for as long as its reply is being generated
- **Usage and cost**: every invocation is persisted with its final status (`replied`, `refused`, `quota`, `failed`) and a full token breakdown including cached, cache-write, and reasoning tokens. The admin page shows 24h / 7d / 30d totals, plus the real billed amount when an organisation admin key is also saved
- **Model picker**: the model list is read live from the provider and never filtered, so a newly released model is selectable the day it ships. The whole chatbot admin surface stays locked until a saved key answers with a model list, and a one-click test pings the selected model before you commit to it
- Enabled bots are listed in the sidebar under **Chatbots**, each linking to its character profile

### Watch Parties

Shared-viewing sessions launched from inside a group chat room. A party is one of two `type`s: a **virtual browser** (a remote Hyperbeam Chromium VM everyone loads and takes turns driving, so you can stream a video, browse, or play a web game together without anyone capturing their own screen) or a **screen share** (the starter broadcasts their own screen, with tab/system audio, over LiveKit). Either way the whole room watches together.

- Started by a room member from the chat composer. The popover asks only for **Virtual browser** or **Screen share** and an optional title. There is nothing else to fill in: the client works out the nearest Hyperbeam region itself (cached for a day in local storage, falling back to the `hyperbeam_region` site setting), and the VM inherits light or dark from whichever theme the starter is using
- Virtual-browser VMs are created with an ad blocker on, WebGL enabled, and the `smooth` picture mode, which favours video over crisp text. They idle out after 5 minutes with nobody connected and hard-stop after 4 hours
- Screen sharers choose between two presets on the fly: **Gaming** (1080p60, VP9, favours framerate, up to 6 Mbps) and **Screenshare** (1080p15, VP9, favours resolution, up to 2.5 Mbps). The starter is the sole sharer; the grant's `CanPublishSources` enforces that no one else can publish a screen
- A side panel renders the live VM iframe or the shared-screen video (with a **fullscreen** toggle on screen shares), plus a participants list, control-handoff request, a copy-invite link, and a **Hide** button that closes the window without ending the party
- **Party chat is the real chat**, not a stripped-down box. It is an actual chat room whose id is the session id, driven by the same panel the live-stream page uses, so replies, edits and deletes, @mentions, image and GIF uploads, the lightbox, and scroll-up paging all work. It stays private to the party, does not clutter your room list, and is excluded from message search. When the party ends the room and every image posted in it are deleted. Because it goes through the normal chat system, the word filter, room timeouts and account locks apply inside a party too
- **In-party voice**: an opt-in **Join Voice** inside any party connects to a session-scoped LiveKit room (`wp_<sessionID>`), so people can talk over what they're watching; everyone in the party hears talkers, with the same local and moderator mute controls as room voice (see [Voice Chat](#voice-chat))
- **Control passing** (virtual-browser parties): the host hands the keyboard/mouse to a specific participant; everyone else watches. Control swaps emit a live WS event so the cursor follows the new driver
- **Kick** by host or room mods, broadcast as a `watch_party_kicked` event that closes the panel for the target without dropping them from the room
- Leaving is handled for you as well as by the Leave button: closing the tab sends a leave beacon, and a party left hidden in a background tab for 10 minutes drops you out
- Server-side reconciliation: idle parties past `watchPartyReconcileIdleAfter` (6 minutes, swept every 5) are torn down automatically so abandoned VMs don't burn Hyperbeam minutes
- Both backends are configured in **Admin → Settings → Watch Parties, Voice & Streaming**: virtual-browser parties need the Hyperbeam API key, screen-share parties and in-party voice need the LiveKit credentials. Each option hides itself when its backend is absent, and the API returns `ErrWatchPartyDisabled`

### Voice Chat

Real-time voice in group rooms, DMs, and watch parties, backed by a self-hosted [LiveKit](https://livekit.io/) SFU so calls scale past the handful of people a peer-to-peer mesh can manage. Audio flows through LiveKit; the Go backend only mints signed join tokens and tracks who is in each call.

- **Join Voice** lives in the chat composer next to the watch-party button; the call renders as a slim bar above the message list (with speaking indicators, mute, and leave), never a takeover modal, so chatting continues during the call
- Joining is gated by the same room membership check as messaging, and DMs additionally respect blocks
- Room/DM call: LiveKit room name = chat room UUID, participant identity = user UUID; presence is tracked from signed LiveKit webhooks (`participant_joined` / `participant_left` / `room_finished`) and broadcast as a `voice_presence` WS event, which also drives an "in call" badge in the room list
- **Presence is reconciled, not just observed**: every 30 seconds a background job rebuilds the whole presence map from LiveKit's own room and participant listing and re-broadcasts only the rooms whose member set actually changed, so a dropped webhook cannot leave a ghost in the badge
- Watch-party call: a separate session-scoped room named `wp_<sessionID>` (so two parties in one room never share a channel); party presence is read live from LiveKit rather than tracked here, so it never pollutes the room's "in call" badge. In-party voice only needs LiveKit configured, independent of the `voice_enabled` toggle
- **Mute controls**: each listener can mute a single participant or everyone just for themselves (client-side volume, nobody else affected); hosts/mods/staff get a **mute-for-everyone** that is permission-based (`UpdateParticipant` revokes the mic publish grant) and **stored in the database**. The backend re-applies the revoked grant from the `participant_joined` webhook every time a muted user reconnects, for room calls and watch-party calls alike, so an old token does not hand the mic back and a restart does not quietly un-mute anyone. The mute is released when LiveKit finishes the room (`room_finished`, i.e. the call empties out), when the watch party ends, or when a moderator lifts it
- **Join tokens are short-lived** (1 hour) and losing access to a room drops the live session immediately: kicks, bans and word-filter kicks call LiveKit `RemoveParticipant` for the room call and for every watch-party call in that room, so an evicted user stops hearing the call at once rather than riding out their token
- Admin-managed and off by default: enable it and set the LiveKit URL / API key / secret under **Admin → Settings → Watch Parties, Voice & Streaming** (the `voice_enabled` toggle reveals the fields, which are shared with live streaming). The Join button is hidden and the token endpoint returns `ErrVoiceDisabled` until configured. See [Deployment → Voice Chat](#voice-chat-livekit) for the server side

### Live Streaming

A public broadcast directory at `/live`. Any member can go live from OBS 30+ or Streamlabs over WHIP, and anyone, logged in or not, can watch. Audio and video ride the same self-hosted LiveKit stack as voice chat, through the bundled ingress. Off by default; see [Deployment → Live Streaming](#live-streaming-livekit-ingress) for the server side.

- **Go live** from the `/live` panel: give the stream a title, pick which playback mode viewers start on, and (when Smooth is configured) enter the bitrate you have set in OBS. The panel hands back a WHIP server URL and stream key with a step-by-step OBS walkthrough and a bitrate calculator
- **Stream credentials** are per user and persistent, so the key only has to be pasted into OBS once. They can be reset from the same panel if the key leaks
- **Two playback modes** per stream: **Low latency** (WebRTC, sub-second) and **Smooth** (HLS, a few seconds behind but rides out network hiccups). The streamer picks which one viewers land on, and each viewer can flip between them on the player (see [Deployment → Smooth Playback](#smooth-playback-livekit-egress--hls))
- **Per-stream chat** is a real chat room whose ID is the stream ID, created with the `live_stream` system kind and the broadcaster as host. Any viewer joins by opening the stream, and the chat can be popped out into its own window for a second monitor
- **Live directory** with thumbnails, titles, and viewer counts that update over WebSocket (`stream_live`, `stream_offline`, `stream_viewers`, `stream_title`), plus a dedicated mobile view
- **Thumbnails** are captured client-side by a watching browser as a 480px-wide WebP frame, posted back on an interval and throttled server-side, so the directory preview stays current without running a compositor
- **Limits**: one live stream per user, a site-wide `stream_max_concurrent` cap (default 3), a 120-character title, and a 500 to 50000 Kbps bitrate range
- **Reconciliation**: a background job every minute tears down streams that never received a broadcaster and streams whose LiveKit room has emptied. Teardown stops the egress, deletes the chat room and its media, removes the thumbnail, and broadcasts the offline event, so nothing accumulates when OBS just disappears

### Games

Multiplayer mini-games hosted entirely inside the site. Each game has live games, past games, and a personal "My Games" view in the sidebar.

- **Chess**, **Checkers**, **Othello**: correspondence-style matches with no clocks. Invite a user by username or pick from your mutual followers, the invitee plays the second-mover side. Drag-to-move, server-side legality, full move history. Disconnects start a 60-second forfeit timer.
- **Minesweeper**: real-time duel on a shared minefield. After both players pick an Umineko character (Bernkastel, Erika, Dlanor, Lambdadelta), independent reveal grids run in parallel, and the first to clear all safe cells wins, hitting a mine instantly loses. Mines are placed lazily after both first clicks so opening reveals are always safe.
- **Snakes & Ladders**: pure dice luck, no decisions. Press Roll, the server rolls a fair six-sided die, ladders carry you up and snakes drop you back. You must land exactly on 100 to win; a roll that would overshoot leaves you where you are and passes the turn. Same correspondence pacing, forfeit timer and spectators as the rest.
- **Spectators**: active games are public. Anyone can open the board and watch live; spectators have their own side chat invisible to the players. Finished games are archived to **Past Games**.
- **Vanity titles** awarded to the top player of chess, checkers, othello and minesweeper (most wins, ties broken by win-loss differential), shown as a pill next to their display name:
  - **Grandmaster** for chess
  - **King of the Board** for checkers
  - **Discmaster** for othello
  - **Minemaster** for minesweeper
- **Notifications** for invites, your-turn nudges, forfeit warnings on disconnect, and game-over results
- Per-game hub (`/games/chess`, `/games/snakes_and_ladders`, etc.) with a How to play panel, the games live right now, and the scoreboard; `/games` is your own games, and the sidebar carries a live games count badge

### Secrets and Unlock Hunts

Hidden puzzles scattered across the UI, declared in code (`internal/secrets/`), surfaced on a public hub page at `/secrets`.

- Each hunt is a **parent secret** (e.g. `witchHunter`) plus a set of piece sub-secrets. Pieces are collected by finding tiny sparkles (`PieceTrigger`) tucked in ordinary UI spots (a tagline, a button, a rule, a subtitle, a chip, a sentence), deliberately varied so pattern-spotting doesn't shortcut the hunt
- **Listed metadata** (title, description, riddle, icon, reward vanity role) is kept in the registry; pieces stay hidden implementation detail
- **Server-side guard** refuses submission of the final phrase unless every piece is already unlocked for the caller, so even a leaked answer can't bypass the hunt. The phrase itself is never stored, only the SHA-256 hash it has to match
- **First solve closes the hunt.** Once anyone answers the parent, every further unlock is rejected (pieces included), everyone who had collected at least one piece gets a "solved it before you could" notification, and their open hunt panel is closed live by a `secret_closed` event
- **Secrets hub page** (`/secrets`) lists every declared hunt with your viewer progress, the first solver, comment count, and a **solvers leaderboard** ranking every user with at least one solved hunt
- **Detail page** (`/secrets/:id`) shows the riddle, a live **progress leaderboard** that reorders in real time via WebSocket as people collect pieces, a pinned first-solver row, and a full-featured discussion thread that stays open forever with its own comment, reply, and like notifications
- **WS presence per secret**: viewers join a `secret:<id>` topic on mount and leave on unmount. Progress and solve events only fan out to current viewers, not the whole site
- **Global events on solve**: when a hunt with a reward role is solved, a `vanity_roles_changed` broadcast refreshes site-info on every connected client so the new role pill appears without a reload
- **Trophy case** on every profile: solved hunts show as live-updating trophies in an Achievements section, owner-clickable to re-open the hunt panel; the in-progress hunt icon lives next to the owner's display name and disappears once they solve
- The v5 hunt is **The Witch's Epitaph**. Maria has hidden twelve letters across the site; finding all twelve unlocks the Maria theme and the sparkling Witch Hunter role

### Announcements

Site-wide announcements with pinning.

- Admins post announcements visible to everyone
- Pinned announcements stay at the top
- Full markdown support in the announcement body
- Full comment system reusing the shared CommentItem component, with threading, media, embeds, and likes
- Optional site-wide announcement banner settable from the admin panel

### Suggestions

A dedicated feedback channel for site improvements and bug reports.

- Posts written in the same composer as the game board, living under a dedicated "Site Improvements" corner
- Status filters: **Open**, **Done**, **Archived**
- Admins can resolve a suggestion (mark done) or archive it, with the status reflected back to the reporter
- Follows the same commenting, voting, and notification rules as the game board

### Search

A single search bar covers the whole site. Backed by Postgres `tsvector` columns on every searchable entity, with a `SearchSource` registry mapping each entity type back to its canonical URL.

- **Full search** (`/search?q=...`) returns paginated hits across theories and theory responses, game board posts and comments, art and art comments, mysteries with their attempts and comments, ships and ship comments, OCs and OC comments, announcements and announcement comments, fanfics and fanfic comments, journals with their entries and comments, users, chat messages, and live streams
- **Quick search** in the header returns up to a small number of hits per entity type, with the right deep link (e.g. a post comment links to `#comment-<id>` on its parent post)
- **Query syntax** comes straight from `websearch_to_tsquery`: bare words are ANDed, `OR` widens, a leading `-` excludes, and `"quoted phrases"` must sit adjacent and in order. Typos still land, because every title (plus usernames) also scores on a `pg_trgm` similarity that is added to the rank, so "beatice" finds "Beatrice"
- **Chat messages are viewer-scoped**, resolved through the chat service rather than the shared SQL, so you only ever see messages from rooms you belong to and signed-out visitors get no chat hits at all
- Filter chips narrow results to one section, plus a **Comments only** chip that spans every section. Drafts, archived journals, and anything authored by a banned or locked user never appear
- Adding a new searchable entity means registering a `SearchSource` and URL builder; nothing else in the search pipeline needs to change, and an `init()` panic on boot catches a source registered without a matching URL builder

### Quote Browser

A standalone interface for browsing the full quote corpus across all three series, sourced from the Umineko Quote Finder API. Switch between Umineko / Higurashi / Ciconia tabs, filter by chapter/episode/arc, filter by truth type (red, blue, gold, purple) on Umineko, and filter by character with a main/additional split where the quote service exposes one. Ciconia and Higurashi quotes ship with Japanese text inline, and the language picker now honours it across all series.

### Profiles and Social Graph

- Avatar, draggable banner positioning, bio, pronouns (preset or custom), gender, date of birth with an optional public toggle, social links (Twitter/X, Discord, Tumblr, WaifuList, GitHub, Bluesky, personal site), favourite character picked from the Umineko / Higurashi / Ciconia casts or from your own OCs
- **Per-user theme, font, and wide layout preferences** persisted on the profile so they follow you across devices. The particles toggle is deliberately per-device and lives in local storage
- Activity feed with recent theories, responses, posts, and comments
- Tabs for posts, theories, art, galleries, ships, OCs, mysteries, fanfics, saved fics, journals, followed journals, and activity
- **Achievements** panel showing every solved unlock hunt as a live-updating trophy
- Stats box: theory count, response count, votes received, ship count, mystery count, fanfic count, follower/following counts
- Follow system with follower and following lists, "Follows you" label, follower counts
- Online/offline status
- **Players Page**: browse all users grouped by role (Reality Authors, Voyager Witches, Witches) and online/offline status, with a name search
- Per-user **blocks** with enforcement across feeds, comments, DMs, and notifications, managed from a blocked-users panel in settings
- Configurable **home page** (the page you land on) and **default profile tab**, each picked from a dropdown in settings
- Email with optional public visibility, a per-user email notification toggle, and separate toggles for the chat message sound and the notification sound
- **Reading progress** recorded per series (Umineko episode, Higurashi arc, Ciconia chapter), used for spoiler gating
- **Favourite GIFs**: star any GIF in the picker or posted by someone else to save it to a personal Favourites tab
- **Danger zone**: change your password, or delete the account behind a password confirmation

### Notifications

A notification is both a DB row (so it shows in the notifications page) and a live event (so the bell counter updates without a reload). `notification.Service.Notify` takes a single `dto.NotifyParams` and fans out from there.

```
   event (e.g. new response on your theory)
       │
       ▼
   notification.Service.Notify(ctx, dto.NotifyParams{...})
       │
       ├─ drop if recipient == actor, or if either side has blocked the other
       │  (a fixed list survives a block: reports, resolved suggestions,
       │   room bans/kicks/unbans, content edits, your-turn and game-over,
       │   GM pauses/away and private clues)
       │
       ├──▶ repository.Notification.Create(...)   (persisted, paginated feed)
       ├──▶ hub.SendToUser(userID, "notification")  (live bell + toast)
       ├──▶ overlay.DispatchNotification(...)       (OBS alert overlay, if connected)
       ├──▶ if the recipient has no socket open: push.Service.SendToUser (FCM)
       └──▶ if EmailAction is set, the type is not chat-room traffic, and there
            is no recent duplicate: email.Service.Send(template, deep-link)
```

`NotifyMany` is the fan-out helper for the many-recipients case; it logs per-recipient failures instead of aborting the batch. Email additionally respects the recipient's `email_notifications` opt-out (reports to staff ignore it) and no-ops when no provider is configured, SMTP or Cloudflare Email. Mobile push is gated on the `push_enabled` site setting plus the FCM credentials file, and chatbots never receive one because the hub reports them as always online. A daily job calls `PruneOld`, which deletes notifications older than 90 days in batches of 5000.

### Stream Overlay

Site events can drive on-stream alert popups through SAMMI. A streamer downloads a personal connector from **Settings → Stream Overlay**, imports it into SAMMI, and their site notifications start arriving as extension triggers they can wire to any overlay they like.

- **Personal connector**: the site generates a `.sef` extension file with your own connection token and the site name baked in, ready to import into SAMMI (Insert → Extension). It ships `Overlay: Connect` and `Overlay: Disconnect` commands and reconnects on its own if the socket drops
- **Events forwarded**: post liked, new follower, post commented, theory upvote, theory response, comment liked, mention, content shared, and art liked. Each arrives on the `overlay_event` trigger with the actor's username, display name, avatar, a human-readable action line, and a timestamp, so one SAMMI button can branch on the event type
- **Token auth**: the overlay connects to `/api/v1/overlay?token=...` with a random 32-byte token, entirely separate from your session cookie, and the origin is still checked against the live base URL. The token is re-validated periodically while the socket is open, so resetting it or banning the account drops the connection rather than waiting for it to close on its own
- **Reset and test**: the token can be rotated at any time (which retires the downloaded connector), the settings panel shows whether SAMMI is currently connected, and **Send test overlay** fires a dummy event so you can prove the wiring before going live

### Moderation and Admin

- **Role system** with themed names and colour-coded usernames with glow:
  - **Reality Author** (super admin)
  - **Voyager Witch** (admin)
  - **Witch** (moderator)
- **Vanity Roles**: admin-defined custom roles with bespoke colour, label, and sort order. Assign one or more to a user independently of their moderation role. System-level vanity roles (hunt rewards, game leaderboard titles) are distinguished from user-created ones
- Permission-based authorisation layer (`internal/authz`), not a raw role check. Every permission carries a **scope**: `staff` permissions can be granted to the moderator role, `general` permissions can additionally be carried by a vanity role, and `restricted` permissions (`manage_settings`, `manage_roles`) can never be granted to anything
- **Permissions page** (`/admin/permissions`): every moderator ability is an individual toggle, so you decide exactly what a Witch may do. Admin and super admin always hold everything and are deliberately absent from the page, so no edit here can lock an administrator out. Saving broadcasts `permissions_changed` and takes effect immediately
- **Vanity roles can carry permissions** too, drawn from the `general` set only. Handing out or taking back a permission-carrying vanity role runs the same protected-user guard as a role change, so it is refused against anyone at or above your own rank
- Admin dashboard with site stats: total users, theories, responses, posts, comments, per-corner breakdown, 24h/7d/30d growth windows, most active users
- User management: assign or revoke roles, ban with reason, unban, lock and unlock, force logout, reset the password, set or clear the email address, mark the email verified, rename and lock the display name, delete the account, and assign vanity roles. The user detail page records **Banned By** (linked profile) alongside Ban Reason and Banned At, and lists **other accounts sharing the same IP**. Bot accounts, and anyone at or above your own rank, are refused
- DB-backed site settings with hot reload: body and upload limits, log level, registration mode, maintenance mode, turnstile, per-action rate limits, announcement banner, email provider (SMTP or Cloudflare), Sentry/GlitchTip DSN, OTLP and Pyroscope endpoints, Valkey cache URL, default theme, LiveKit voice, Hyperbeam, live streaming, mobile push, and the chatbots
- **Invite system**: open, invite-only, or closed registration. Admins generate one-time invite codes
- **Maintenance mode** with custom title and message. Admins bypass it
- **Audit log** for admin actions, filterable by action. Automated moderation events (word-filter hits) log with a NULL actor and render as "System" in the admin audit page, distinguishing them from human-initiated actions
- **Reports**: users can report theories, theory responses, game board posts, art, mysteries, mystery attempts, journals, and comments on every commentable surface (game board, art, mysteries, ships, OCs, fanfics, journals, announcements, and unlock hunts). Admins resolve from the admin panel with an optional comment sent back to the reporter
- **Banned GIFs**: admins block specific GIPHY IDs from being embedded anywhere on the site; the content filter rejects matches before they render
- **Banned Words** (`/admin/banned-words`): global word-filter rules for chat rooms with regex / whole-word / substring match modes, editable in place, behind its own `manage_banned_words` permission
- **Chatbots** (`/admin/chatbots`): character accounts backed by an OpenAI model, each with its own username, avatar, system prompt, model, reasoning effort, verbosity, and token cap, alongside live usage figures. They can be left open to everyone, or gated behind a vanity role that members opt into themselves from **Settings -> Characters**
- **Content Filter Pipeline** (`internal/contentfilter`): pluggable rule-based validation that runs on all user-generated text before it lands in the DB
- **Content rules** per section (welcome page, theories for each of the three series, mysteries, ships, fanfiction, reading journals, the general game board and each of its five corners, the general gallery and each of its three corners, site improvements, and chat rooms), admin-editable and displayed at the top of each page
- **Per-action rate limits**: max theories, responses, posts, art, journals, and chat rooms per day, plus max members per chat room, all settable from the admin panel
- **Cloudflare Turnstile** on login and registration, toggle-able from admin settings

### Platform Features

- **Fourteen themes** grouped by series in the theme picker:
  - **Umineko**: Featherine (gold/purple, default), Beatrice (warm gold/brown), Bernkastel (blue), Lambdadelta (pink), Erika Furudo (cyan/pink), Battler, Virgilia (light mode)
  - **Higurashi**: Rika, Mion, Satoko
  - **Ciconia**: Miyao (deep navy with gold and sky-blue), Lingji (crimson and gold), Stanis&#322;aw (silver on near-black)
  - **Unlockable**: Maria Ushiromiya (rosy pink), granted by solving the Witch's Epitaph hunt. It stays out of the picker until you hold the reward, and choosing it without the reward falls back to the site default
- **Two font families**: default serif set (Cinzel and Garamond) or **IM Fell English** for a period-correct look, per-user preference
- **Wide layout toggle** and **ambient particles toggle** (floating butterflies plus theme-specific motifs such as candy and lollipops on Lambdadelta)
- **Discord-style text formatting** across posts, comments, DMs, chat rooms, mysteries, and art/ship descriptions:
  - `**bold**`, `*italic*`, `__underline__`, `~~strikethrough~~`, and `***bold italic***`
  - Backticks for inline code, triple backticks for multi-line code blocks with syntax highlighting via highlight.js
  - `>` for block quotes that flow across wrapped lines and terminate on a blank line
  - `||spoiler||` for hover-to-reveal spoilers
  - Truth colours (`[red]...[/red]` etc.) that still glow inside quotes
- **GIPHY integration** on posts, comments, DMs, and chat rooms with Trending and per-user Favourites tabs, one-click send, and an admin banlist
- **OG embeds** for rich previews when sharing on Twitter and Discord, covering theories, posts, game board corners, profiles, mysteries, ships, OCs, art, galleries, announcements, fanfics, journals and journal entries, chat rooms, watch parties, unlock hunts, and live streams, with locale, image dimensions, and canonical URL tags. WebP uploads are re-served as JPEG through `/og-image/*`, because the scrapers will not render WebP
- **Auto-generated sitemap** with a sitemap index and sub-sitemaps for static pages, theories, posts, art, users, mysteries, ships, fanfics, and journals
- **Media processing**: image-to-WebP (cwebp) and video-to-MP4 (ffmpeg, H.264 CRF 28) encoding via a background worker pool, local FFmpeg thumbnail generation
- **Client-side validation** of file sizes before upload, pulled from live server settings
- **Auto-expanding composers**: every text box grows as you type, capped at half the viewport before scrolling internally
- **Security headers** on every response via helmet: HSTS with preload, `X-Frame-Options: DENY`, nosniff, a strict referrer policy, a narrow permissions policy, and an enforced CSP (`base-uri`, `form-action`, `frame-ancestors`, `object-src`) shipped alongside a much fuller `Content-Security-Policy-Report-Only`
- **Structured logging** with zerolog, configurable log levels, settings change listener pattern
- **GlitchTip / Sentry** error tracking via a configurable DSN, with structured attribute mapping in `internal/logger/glitchtip_shipper.go`
- **Native mobile app**: the same React frontend packaged with Capacitor, using bearer-token auth and FCM push (see [Mobile app (Capacitor)](#mobile-app-capacitor))
- Fully **mobile responsive** across all pages
- **Cache headers**: `/static/assets/*` and HLS segments are immutable, uploads and static media are 30 days, HLS playlists and API responses are `no-cache`, HTML is `no-store`

## Tech Stack

**Backend**

- Go 1.26
- Fiber v3 (HTTP router)
- PostgreSQL via `jackc/pgx/v5` (through the `pgx/v5/stdlib` adapter so `database/sql` and `otelsql` instrumentation still apply)
- Goose for migrations
- testcontainers-go for repo-layer tests against a real Postgres
- fasthttp/websocket for the WebSocket hub
- zerolog for structured logging
- wneessen/go-mail for email delivery
- getsentry/sentry-go for GlitchTip / Sentry error tracking
- disintegration/imaging for server-side image manipulation
- openai/openai-go v3 (Responses API) for the chatbot character accounts
- valkey-io/valkey-go, with `valkeyhook` for the tracing and metrics hook, for the optional app cache
- prometheus/client_golang for the `/metrics` registry and the custom collectors
- OpenTelemetry Go SDK with the OTLP/HTTP trace exporter, plus XSAM/otelsql for per-statement SQL spans
- grafana/pyroscope-go for continuous profiling
- hellofresh/health-go for the `/health` dependency checks
- firebase.google.com/go for native push (FCM) to the mobile app
- livekit/server-sdk-go for the SFU, ingress, and egress control plane

**Frontend**

- React 19 with TypeScript 6
- Vite 8
- React Router v7 (not react-router-dom)
- CSS Modules
- DOMPurify + marked for safe markdown rendering
- highlight.js for syntax-highlighted code blocks
- TipTap 3 (with StarterKit, Placeholder, TextAlign, Color, TextStyle extensions) for the fanfiction rich text editor
- emoji-picker-react for chat reactions and emoji insertion
- @marsidev/react-turnstile for bot protection

**Infrastructure**

- Docker multi-stage build (Node build stage + Go build stage + Alpine runtime)
- FFmpeg and libwebp-tools in the runtime image
- Two Valkey instances in the compose files: one coordinating the LiveKit ingress and egress, one (`valkey-cache`, LRU-capped) for the optional app cache
- Designed to sit behind Caddy or another reverse proxy in production
- Session auth with httpOnly cookies, no JWTs
- Mockery v3 (`.mockery.yml`) for generated Go interface mocks
- `docker-compose.prod.yml` carries `prometheus-*` labels for label-based scrape discovery and runs a `postgres-exporter` sidecar

**External**

- [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html) for game quote search and evidence attachment
- GIPHY API for GIF search, trending, and favourites
- [Hyperbeam](https://hyperbeam.com/) for the shared-browser VM that powers virtual-browser watch parties
- [LiveKit](https://livekit.io/) (self-hosted) as the SFU that carries voice in chat rooms, DMs, and watch parties, plus screen-share watch parties and live streaming
- OpenAI for the chatbot character accounts, keyed from the admin panel
- Firebase Cloud Messaging for native push to the packaged mobile app
- Optional and entirely external: a Prometheus scraper, an OTLP trace collector, a Pyroscope server, and a GlitchTip / Sentry instance. None are bundled in the compose files

## Architecture

The server is a single Go binary that embeds the compiled Vite bundle and serves both the SPA and the JSON API from one process. Every layer has a single responsibility: controllers parse HTTP, services orchestrate business logic, repositories own SQL, the hub owns live events, and the media processor owns encoding off the hot path.

### High-Level Component Map

```
        ┌──────────────────────────────┐  ┌──────────────────────────────┐
        │      Browser (React 19)      │  │   Capacitor app (same SPA)   │
        │  session cookie + WebSocket  │  │   bearer token + WebSocket   │
        └───────────────┬──────────────┘  └───────────────┬──────────────┘
                        │ HTTP / WS                       │ HTTP / WS
                        └────────────────┬────────────────┘
                                         ▼
        ┌─────────────────────────────────────────────────────────────────┐
        │                          Fiber v3 app                           │
        │  recover → tracing → host allow-list → security headers → etag  │
        │  → cache headers → cors → log/sentry → maintenance → metrics    │
        │  → last-seen IP        (auth and authz attach per route)        │
        └────────┬────────────────────────────────────────┬───────────────┘
                 │                                        │
                 ▼                                        ▼
        ┌────────────────┐                       ┌──────────────────────┐
        │  Controllers   │                       │    WebSocket hub     │
        │  (HTTP → DTO,  │                       │  per user / room /   │
        │   authz gate)  │                       │  topic, in-process   │
        └────────┬───────┘                       └───────┬──────────────┘
                 │                                       │
                 ▼         notify / push                  │
        ┌────────────────┐ ─────────────────────────────▶ │
        │    Services    │                                │
        │ (rules, filter,│                                ▼
        │ orchestration) │                       ┌──────────────────────┐
        └────────┬───────┘                       │    Media processor   │
                 │                               │  (image/video queue  │
                 ▼                               │   → ffmpeg / cwebp)  │
        ┌────────────────┐   ┌────────────────┐  └──────────────────────┘
        │  Repositories  │──▶│ internal/cache │
        │ (interfaces +  │◀──│ Valkey, opt-in │
        │  cache seam)   │   └────────────────┘
        └────────┬───────┘
                 ▼
        ┌────────────────┐
        │      DAOs      │
        │   (all SQL,    │
        │   db.WithTx)   │
        └────────┬───────┘
                 ▼
        ┌────────────────┐
        │   PostgreSQL   │
        │ (UUID, JSONB,  │
        │  CITEXT, FKs)  │
        └────────────────┘
```

Voice, watch parties, and live streaming run on a separate media plane: audio and video travel from the browser to the LiveKit SFU (or to the Hyperbeam VM) directly, and the Go process only mints signed tokens and tracks presence, so none of that traffic passes through the layers above.

### Request Lifecycle

A typical `POST /api/v1/theories` request walks through a fixed global middleware chain, picks up its auth and permission middleware at the route itself, lands in a controller, then flows down through service, repository, and DAO:

```
 POST /api/v1/theories
     │
     ▼
 ┌─────────────┐   panic guard, stack trace to the log
 │   recover   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   OpenTelemetry server span, trace_id local + X-Trace-ID header
 │   tracing   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   Host must match the base_url hostname
 │ host allow  │   (/health, /livez, /metrics, LiveKit webhook exempt)
 └──────┬──────┘
        ▼
 ┌─────────────┐   helmet: HSTS, nosniff, frame-deny, enforced + report-only CSP
 │  sec hdrs   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   shortcuts 304 responses before hitting handlers
 │    etag     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   per-path Cache-Control (immutable assets, no-cache API,
 │ cache hdrs  │   split playlist/segment rules for /hls)
 └──────┬──────┘
        ▼
 ┌─────────────┐   origin gated against live SettingBaseURL or a Capacitor app origin
 │    CORS     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   request-scoped client_ip, Sentry transaction, access log
 │  logger +   │
 │   sentry    │
 └──────┬──────┘
        ▼
 ┌─────────────┐   JSON 503 on /api unless the caller has manage_settings
 │ maintenance │   (site-info, login, session, and ws stay reachable)
 └──────┬──────┘
        ▼
 ┌─────────────┐   duration and in-flight histograms exported on /metrics
 │   metrics   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   per route: RequireAuth / OptionalAuth / RequirePermission
 │    auth     │   bearer or cookie → session row → ban, lock, and verify gates
 └──────┬──────┘
        ▼
 ┌─────────────┐   binds the DTO, reads userID from ctx.Locals
 │ controller  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   content filter → per-day caps → business rules → DTO mapping
 │   service   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   interface plus cache seam (internal/repository)
 │ repository  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   all SQL, db.WithTx for multi-table writes (internal/dao)
 │     DAO     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   PostgreSQL, FKs enforced, native UUID/JSONB/CITEXT
 │     DB      │
 └─────────────┘
```

Rate limiting is not part of the global chain. `RateLimitCredentials` (10 per minute per IP) and `RateLimitMail` (5 per hour per IP) are attached only to the auth routes, and Turnstile only to register, login, and forgot-password.

### Data Layer

- **All SQL lives in `internal/dao/`**, one file per domain (theory.go, post.go, art.go, mystery.go, ship.go, fanfic.go, journal.go, chat.go, permission.go, etc.). The DAO structs are unexported and built through `dao.NewTheory(db)` and friends in `internal/dao/new.go`.
- **`internal/repository/` owns the contract, not the queries.** Each domain file declares the interface services depend on, the row models (`internal/repository/model/`), and a thin passthrough struct that wraps the DAO. `internal/store/new.go` is the single wiring point: `repository.NewXRepo(dao.NewX(db), cache)`.
- **The passthrough is the cache seam.** Because every read and write already funnels through it, it is the only place caching is allowed to intercept: read-through on gets, explicit `Del` on writes. `internal/repository/permission.go` is the canonical example, with a shared `cachedRead` helper on the gets and cache invalidation on `SetRolePermissions`.
- **`internal/cache` is a Valkey client behind a hot-reloadable manager.** The `valkey_url` site setting swaps the client at runtime, generic `cache.Get[T]` / `cache.Set[T]` JSON round-trip any value, and namespaces with their TTLs are declared in `internal/cache/keys.go`. A nil client turns every read into a miss, so the whole cache is optional and the site runs identically without it. Hits, misses, command latency, and Valkey server stats are exported to Prometheus on `/metrics`. Eight repositories currently take the manager: user, role, mystery, vanity role, permission, user secret, game room, and chatbot.
- **Shared DAOs for repeated shapes**: comments, likes, media, and view counters are generic over the parent key (`newCommentDAO[K]`, `newLikeDAO`, `newMediaDAO`, `newViewDAO`) and embedded into each domain DAO by promotion, so nine comment systems share one implementation parameterised by table and foreign-key name.
- **Transactions** use the `db.WithTx(ctx, db, func(tx) error)` helper in `internal/db/tx.go`. DAO methods that touch multiple tables wrap themselves in `WithTx` and expose a single combined method (e.g. `CreateWithCharacters`, `UpdateWithTags`, `MarkSolved`). Services do not handle transactions directly.
- **Native Postgres types** throughout the schema: `UUID` for primary and foreign keys, `BIGINT GENERATED BY DEFAULT AS IDENTITY` for auto-increment columns, `BOOLEAN` for flags (no more `INTEGER 0/1`), `TIMESTAMPTZ` for time columns, `JSONB` for `state_json` / `action_json`, and `CITEXT` (case-insensitive text) for unique-by-name lookups like fanfic series, languages, and OC characters.
- **Foreign keys** are enforced by Postgres. Most deletes cascade through `ON DELETE CASCADE`; `galleries -> art.gallery_id` is `ON DELETE SET NULL`, so the gallery delete path explicitly removes child art inside a transaction.
- **Hot-reloadable settings** live in the `site_settings` table and are served through `internal/settings`. Listeners registered at startup react to changes (e.g. re-reading the log level, reconnecting the cache) without a server restart.
- **The one exception to "no SQL outside the DAO"** is `internal/repository/search.go`, which holds the `SearchSource` registry: a declarative SQL fragment per searchable entity that the search DAO assembles into the union query.
- **DAO tests** boot a real `postgres:18` container per test binary via testcontainers-go, then create a per-test database from a pre-migrated template. Public test API: `daotest.NewRepos(t)`, `daotest.CreateUser(t, repos, opts...)`, `daotest.CreateSession(t, repos, userID)`. Tests need Docker on the host.

```
  controller ──▶ service ──▶ repository ──▶ dao ──▶ db.WithTx(ctx, func(tx) {
                                 │           ▲            INSERT ...
                                 │           │            INSERT ...
                                 ▼           │            UPDATE ...
                          internal/cache     │      })
                          (read-through,     │
                           Del on write)     │ one method per logical operation,
                                             │ not one-method-per-table
```

### Cache (Valkey)

`internal/cache` is an optional read-through cache in front of the hot lookups every request makes. It stays off unless a `valkey_url` is saved, and it is deliberately a different instance from the Valkey that LiveKit's ingress and egress coordinate over.

- **Hot-reloadable connection**: the manager subscribes to the `valkey_url` site setting. Saving a new URL closes the old client and opens and pings a new one; clearing the URL closes the client and puts every lookup straight through to Postgres. No restart, and the admin panel probes the URL as a setting validator so an unreachable address is rejected at save time rather than at first read.
- **Typed helpers**: `cache.Get[T]`, `cache.Set[T]`, and `cache.SetMany[T]` marshal through JSON, with `string` and `[]byte` passed through untouched. A nil manager or a missing key both return `ErrMiss`, so every call site is a plain cache-miss branch and no caller has to know whether caching is switched on.
- **Namespaces own their TTL**: each cached thing is a `cache.Namespace` with a key prefix and a TTL, declared in one file (`internal/cache/keys.go`) rather than scattered across call sites. Site settings, user roles, vanity assignments, secret progress, and the mystery and game leaderboards are held indefinitely and invalidated by whoever writes them; the authz role and vanity permission tables hold for a minute; OG metadata for five minutes and rendered OG images for a day.
- **Interception lives in the repository passthroughs**, not in services and not in the DAOs that own the SQL, so a cached read and its invalidation sit next to each other and there is still exactly one writer per table.
- **Instrumented**: hit and miss counters, per-command duration and error metrics, an OpenTelemetry client span per command, and a collector that scrapes the server's own `INFO` for key count, memory, evictions, and connection stats (see [Observability](#observability)).

### Auth and Sessions

- Server-side sessions stored in Postgres with httpOnly cookies. No JWTs.
- The cookie (`ut_session`) carries a random 32-byte hex token and nothing else. The `sessions` row stores the **SHA-256 of that token** as its primary key, alongside the user ID and expiry, so a database leak does not hand over usable sessions.
- Cookie flags: `HTTPOnly`, `SameSite=Lax`, `Path=/`, and `Secure` whenever the live `base_url` starts with `https://`. `SettingSessionDurationDays` (default 30) sets both the cookie `MaxAge` and the row's `expires_at`.
- **Sessions are not renewed.** Lifetime is fixed at creation, so changing the duration setting only affects sessions created afterwards. Expired rows are swept by a background job every 24 hours.
- **The mobile app uses the same session, carried differently.** When a request arrives with an `X-Client-Platform` header, login and register echo the token back in an `X-Session-Token` response header (exposed through CORS); the app stores it in Capacitor Preferences and sends it as `Authorization: Bearer <token>` on every later call. `SessionToken()` prefers the bearer header over the cookie, so both clients hit the same validation path. The WebSocket upgrade reads the cookie and falls back to a `?token=` query parameter, since a webview cannot set headers on an upgrade.
- Auth middleware is attached **per route**, not globally: `RequireAuth`, `OptionalAuth`, or `RequirePermission`. Beyond resolving the user it drops the session and returns 403 for banned accounts, blocks write methods from locked accounts, and blocks write methods from unverified emails, each with a small exempt list (marking notifications read, reading a chat room, and the email-verification endpoints themselves).
- Password reset and moderator action delete every session row for the user and immediately disconnect their live WebSockets through the hub, so a revoked account cannot ride out its cookie.

```
   Browser / app          Server
  ───────────────        ──────────
   login form ──────────▶ auth.Login
                          │
                          │ verify credentials, reject if banned
                          │ 32 random bytes → hex token
                          │ INSERT sessions (sha256(token), user_id, expires_at)
                          │
   Set-Cookie ◀───────────┤  ut_session (httpOnly, SameSite=Lax, Secure on https)
   X-Session-Token ◀──────┘  only when the request carried X-Client-Platform
        │
        │ every subsequent request
        ▼
   ┌────────────────┐
   │ auth middleware│── Authorization: Bearer … or the ut_session cookie
   │  (per route)   │── sha256 → sessions row → expiry → ban / lock / verify gates
   └────────────────┘── userID into ctx.Locals
```

### Account Security

Registration takes an email address, and the account is only half usable until that address is confirmed.

- **Email verification**: a 24-hour token is mailed on registration and again whenever the address changes. Until it is used, write requests (anything that is not a GET, minus a small exempt list so verifying and resending still work) are refused with an `email_unverified` code. Each user row carries a `verify_grace_until` timestamp, which defaults to "now" for new accounts but let the existing population be migrated with a window rather than locked out on the day the requirement landed.
- **Password reset**: a one-hour token mailed to the address on file. Changing an email also notifies the previous address, so a hijacked account cannot quietly move itself.
- **Ban versus lock**: a ban is terminal, the session is deleted on the next request and the account is refused outright. A **lock** is the softer, reversible option: the account can still read the site but every write is refused. Both record who applied them and why, and both surface on the admin user detail page.
- **Rate limits on credentials**: ten credential attempts per client IP per minute (login, register, reset) and five mail-sending attempts per hour (forgot password, set email, resend verification), keyed off the same resolved client IP the logger uses, so `CF-Connecting-IP` behind Cloudflare rather than the proxy address.
- **Reserved usernames** are refused at registration, and Cloudflare Turnstile can be required on login and registration from the admin panel.

### Permission Model

- Every action is gated on a **permission**, not a raw role check. Permissions are declared once in `internal/authz/permissions.go` as a catalogue of `PermissionDef{Permission, Label, Scope}`, and that catalogue is what the admin UI renders.
- **Permissions are editable at runtime.** `/admin/permissions` (gated on `manage_roles`) writes them to the database rather than to a compiled table: the moderator role and every custom vanity role can have permissions ticked on and off, and a change takes effect on save.
- **Admin and super admin are immutable.** They always hold every permission and are deliberately absent from the permissions page, so nothing edited there can lock an administrator out of the site.
- Every permission carries a **scope** that decides who may ever be granted it:
  - `staff`: assignable to the moderator role, never to a vanity role. Most of the catalogue, e.g. `ban_user`, `edit_any_post`, `delete_any_comment`, `view_audit_log`
  - `general`: assignable to the moderator role *and* to vanity roles, which is how an opt-in perk reaches ordinary members. Currently just `use_chatbot`
  - `restricted`: `manage_settings` and `manage_roles`, assignable to neither, because handing either one out is an escalation path to effective admin
- A user's effective permissions are the union of their system role's grants and the grants of every vanity role assigned to them. When the Valkey cache is on, both lookup tables and each user's vanity-role list are cached with a one-minute TTL, so a permission change propagates within about a minute rather than instantly.
- Some features (the "game master" view in mysteries) check `role == super_admin` directly because the behaviour is intentionally scoped to that one role, not to a permission grant. Ownership checks ("is this your own post") are likewise separate from the permission catalogue.

```
  role            permissions
  ──────────────  ─────────────────────────────────────────────────
  super_admin     everything, not editable
  admin           everything, not editable
  moderator       editable, seeded with view_admin_panel, view_users,
                  ban_user, edit_any_*, delete_any_*, use_chatbot, ...
  vanity roles    editable, general-scope permissions only
  member          no catalogue permissions; owns its own content
```

### Content Filter

`internal/contentfilter` is a pluggable validation pipeline. Every text-bearing service runs its payload through the manager before writing: registration (username and display name), profile, theories, posts and comments, art, ships, OCs, mysteries, fanfics, journals, secrets, game-room spectator chat, chat rooms and DMs.

```
   user text ──▶ ┌──────────────────────────────────┐
                 │  contentfilter.Manager           │
                 │                                  │
                 │  ┌─ RuleSlurs       ──┐          │  first failing rule
                 │  └─ RuleBannedGiphy ──┘  ──────▶ │  stops the chain and
                 │                                  │  returns *RejectedError
                 └──────────────────────────────────┘  to the caller
                              │
                              ▼
                         accept → service writes to repo
```

Rules are registered in order in `initServices` (slurs, then banned GIPHY) and each one sees every non-empty text field of the payload in a single call. Text is NFKC-normalised with Unicode format characters stripped before matching, so zero-width joiners and lookalike compositions cannot smuggle a match past a rule.

The banned-GIPHY rule reads the live banlist from `internal/giphy/banlist` and rejects both individual GIF IDs and whole GIPHY channels, resolving the uploader of an unrecognised ID through the GIPHY API when it has to. The admin banned-GIFs UI writes to that list and changes apply instantly without a restart.

The per-room chat word filter lives in the same package but deliberately outside the manager. `ChatBannedWordsRule` is held by the chat service and called as `CheckForRoom(ctx, roomID, texts...)`, because its rules are scoped to one room and carry an action (delete or kick) rather than a plain reject. Compiled patterns are cached per rule row and invalidated when a moderator edits the rule.

### WebSocket Hub

`internal/ws` is a single in-process hub that multiplexes every live event on the site. Clients open one socket per tab, the hub keys them by user ID, and services push events through `SendToUser`, `Broadcast`, `BroadcastPublic`, `BroadcastToRoom`, or `BroadcastToTopic`. A signed-out visitor still gets a socket: it registers in the anonymous set, may only ping, and receives `BroadcastPublic` events (live-stream and chatbot changes) so public pages update without an account.

```
  authed clients (many tabs)            anonymous clients (no session)
     │ websocket upgrade                   │ websocket upgrade
     ▼                                     ▼
  ┌───────────────────────────────────────────────────────────────┐
  │                            ws.Hub                             │
  │                                                               │
  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────────────┐  │
  │  │ by user ID   │  │ by room ID   │  │ viewers per room    │  │
  │  │  {u: [c,c]}  │  │  {r: {u,u}}  │  │  {r: {u: tabs,      │  │
  │  └──────────────┘  └──────────────┘  │       active/idle}} │  │
  │  ┌──────────────┐  ┌──────────────┐  └─────────────────────┘  │
  │  │ anon clients │  │ always online│                           │
  │  │  {c, c}      │  │  {bot, bot}  │                           │
  │  └──────────────┘  └──────────────┘                           │
  └──────▲────────────────────▲──────────────────▲────────────────┘
         │ SendToUser         │ BroadcastToRoom  │ Broadcast
         │                    │ BroadcastToTopic │ BroadcastPublic
  ┌──────┴───────┐   ┌────────┴─────┐   ┌────────┴────┐
  │ notification │   │ chat service │   │ like / view │
  │   service    │   │ (msg, react, │   │  counters   │
  │              │   │  pin, typing)│   │             │
  └──────────────┘   └──────────────┘   └─────────────┘
```

- **Rooms** are UUID-keyed sets of user IDs. Chat rooms are joined at connect time from the caller's membership list. **Topics** reuse the same map with a synthetic UUID derived from a string (`TopicUUID`), which is how `secret:<id>` progress and per-game spectator chat fan out only to the people currently looking at that page.
- **Viewer presence** is separate from room membership: `AddViewer` / `RemoveViewer` reference-count open tabs per room and carry an `active` / `idle` state, driven by the client's `join_room`, `leave_room`, and `viewer_state` frames. Losing the last socket clears the viewer rows and broadcasts the departure.
- **Always-online set**: chatbot users are pushed into `SetAlwaysOnline` whenever the chatbot config reloads, so `IsOnline` reports them as present in member lists without a socket of their own, and the notification path does not try to send them a mobile push.
- **Back pressure**: every client has a 64-message buffer and a non-blocking enqueue. A consumer that cannot keep up is killed and reaped rather than stalling the broadcaster.
- **Connection hygiene**: 8 KB inbound frame cap, 100 frames per second with a burst of 200, a 30-second server ping against a 90-second read deadline, and a session revalidation plus ban check at most every five minutes on pong. `session.Manager` holds the hub as its `Disconnector`, so revoking a user's sessions closes their live sockets.
- Connections and inbound frames are exported to Prometheus as `ws_connections{hub,authed}`, `ws_connections_total`, `ws_inbound_messages_total{type}`, and `ws_inbound_dropped_total{authed}`. There are two hub instances: `main` for the site and `overlay` for the OBS browser-source alert feed.

The frontend keeps one socket for the whole app on `/api/v1/ws`, pings every 20 seconds, closes the socket itself if nothing arrives for 90 seconds while the tab is visible, and reconnects with full-jitter exponential backoff capped at 30 seconds. On reconnect it refetches the affected queries; there is no polling fallback.

### Notifications

A notification is both a DB row (so it shows in the notifications page) and a live event (so the bell counter updates without a reload). The notification service fans out through the hub and optionally through email.

```
   event (e.g. new response on your theory)
       │
       ▼
   notification.Service.Notify(ctx, userID, type, payload)
       │
       ├──▶ repository.Notification.Insert(...)  (persisted, paginated feed)
       ├──▶ hub.SendToUser(userID, msg)          (live bell + toast)
       └──▶ if user has email opt-in and SMTP configured:
                email.Service.Send(template, deep-link)
```

### Media Pipeline

Every upload lands on disk first, then goes through `media.Processor`, a fixed pool of worker goroutines fed by a buffered channel. Images and video take different routes through it: an image upload waits for its own encode so the caller can persist the final `.webp` URL, while video is recorded at its raw path and transcoded behind the request.

```
   controller receives multipart upload
         │
         ├── image ──▶ upload.Service.SaveImage
         │               original bytes land on disk (uploads/<subdir>/)
         │               pixel guard (max_image_pixels) rejects decode bombs
         │               enqueue JobImage, then block on the callback
         │
         └── video ──▶ media.Uploader.SaveAndRecord
                         original bytes land on disk, media row written,
                         enqueue JobVideo and return to the client
                                 │
   ┌─────────────────────────────┴───────────────┐
   │  buffered job channel (cap 256)             │
   └───────┬─────────────────────────────────────┘
           │ N worker goroutines (4 at startup)
           ▼
   ┌─────────────────────────────────────────────┐
   │ image worker → cwebp                        │
   │   q80 by default, q60 square 96px avatars,  │
   │   q72 1600px banners, EXIF auto-orient,     │
   │   GIF → animated WebP via ffmpeg            │
   │ video worker → ffmpeg libx264 CRF 28,       │
   │   AAC 128k, faststart; .webm untouched      │
   └───────┬─────────────────────────────────────┘
           │
           ▼
   image: caller gets the .webp path, source file removed
   video: callback repoints the media row at the .mp4, removes the source,
          then ffmpeg grabs a random frame as a 200px-tall WebP thumbnail
```

The image path is synchronous on purpose. `SaveImage` enqueues the job and selects on the result, the error, and the request context, so a failed or dropped encode surfaces as a failed upload instead of a media row pointing at a file nobody will serve. A `.webp` upload is re-encoded in place, and an animated one is left alone. The video path is fire and forget: if the transcode fails, the row keeps pointing at the raw upload and the failure is logged.

If the queue is full the job is dropped and its error callback fires immediately rather than back-pressuring the request. Shutdown behaves the same way: the processor stops accepting work, waits up to 15 seconds for in-flight encodes, then fails whatever is still queued.

### Background Jobs

Recurring work runs as plain goroutines started at boot by `registerListeners` in `init_jobs.go`. Each job is a `scheduleJob(name, interval, fn)`: it runs once immediately, then on a ticker, logs a count only when it actually did something, and stops on a shared channel at shutdown so a redeploy never kills work halfway through.

| Job | Interval |
|---|---|
| Reconcile voice presence | 30 seconds |
| Reconcile live streams | 1 minute |
| Cancel idle games | 5 minutes |
| Refresh stale link embeds | 1 hour |
| Archive stale journals | 1 hour |
| Archive stale chat rooms | 1 hour |
| Clean orphaned upload files | 24 hours |
| Prune old notifications | 24 hours |
| Clean expired sessions | 24 hours |

The same function registers the settings listeners that make hot reload work: log level, OTLP endpoint, Pyroscope URL, request body limit, native push credentials, the Valkey cache URL, the chatbot opt-in role migrator, SMTP, and the chatbot and OpenAI settings blocks. It also ensures the system chat rooms exist at startup.

Shutdown drains in order: the chatbot worker pool, then the media processor, then the job tickers, then the cache client, all inside a single 15-second budget.

### OG and SEO

`internal/og` owns the SEO meta surface. The SPA catch-all serves every extension-less path through `og.Resolver.Resolve(ctx, path, partyID)`, which matches the URL to a resolver and rewrites the meta tags of the embedded Vite `index.html` in memory. The base document already carries `og:type`, `og:locale`, `twitter:card`, and the default image dimensions; the resolver overwrites `<title>`, `description`, `og:title` / `og:description` / `og:url` / `og:site_name` / `og:image`, the matching `twitter:*` tags, and `<link rel="canonical">`. Paths that match nothing keep the site-wide defaults, and `__BASE_URL__` in the document is substituted once at startup.

```
   GET /theory/<id>
       │
       ▼
   og.Resolver.Resolve(ctx, path, party)
       │
       ├─ resolveMeta() ─▶ app cache (og:meta:, 5 min)
       │                     │ miss
       │                     ▼
       │                 metaForPath() ─▶ theoryMeta(ctx, id) ─▶ repo.GetByID
       │                                                          │
       │                                                          ▼
       │                                                     Meta{Title, Description,
       │                                                          Image, URL}
       │
       └─ inject(meta)  ──▶ rewrites og:*, twitter:*, <title>, description
                            and the canonical link in the embedded index.html
```

- Detail resolvers exist for theories, posts, profiles, art, galleries, mysteries, ships, OCs, fanfics, announcements, journals and journal entries, chat rooms, watch parties (`?party=<id>` on a room URL), secrets, live streams, and individual games. Section index pages, game-board corners, gallery corners, and the games hub get static per-page copy.
- Resolved `Meta` is cached in the Valkey app cache for five minutes, so a link passed around Discord does not re-query the repo for every unfurl. With caching switched off the lookup simply runs every time.
- Uploaded images are WebP, which several scrapers still refuse, so `og:image` is rewritten to `/og-image/<path>.jpg`. That route decodes the WebP (downscaling through `dwebp` when it is wider than 1200px, pulling frame one out with `webpmux` when it is animated), encodes JPEG at quality 85, and caches the bytes for 24 hours keyed on the file's modification time and size. If conversion fails it serves the original WebP. When a per-entity image is injected, the static `og:image:width` / `og:image:height` tags are stripped because they no longer describe it.
- The fallback image is the built-in `/Featherine.jpg`, overridable from admin settings (`og_default_image`, which must be an uploaded `.jpg`).

Adding a new page means adding a branch to `metaForPath()`; see [Adding a New Page](#adding-a-new-page).

### Service Composition (init_services.go)

Wiring is explicit and split across four files at the repo root. There is no DI container: `initServices` builds every service in dependency order and returns the `services` struct, which is the dependency graph.

- `init_db.go` (`initDatabase`): telemetry, `db.Open`, `db.Migrate`, `db.SeedContent`, the cache manager, the repositories, and the settings service. Once settings are loaded it re-inits the logger and applies the GlitchTip DSN, the OTLP endpoint, and the Pyroscope endpoint.
- `init_services.go` (`initServices`): every service, in dependency order.
- `server.go` (`initServer`, `initApp`): builds the Fiber app, installs middleware, hands the services to `controllers.NewService`, registers routes, and returns the app plus a cleanup func.
- `init_jobs.go` (`registerListeners`): settings listeners and the background job tickers, returning a stop func.

```
  config + env
     │
     ▼
  telemetry.Init → db.Open → db.Migrate → db.SeedContent    (init_db.go)
     │
     ▼
  cache.NewManager → store.New(db, cache)  ──▶  one repo per domain,
     │                                          SQL in internal/dao
     ▼
  settings.NewService (DB-backed, hot reload)
     │  └─▶ re-init logger, GlitchTip DSN, OTLP, Pyroscope
     ▼                                                  (init_services.go)
  session, media.Processor, upload, authz, giphy + banlist,
  contentfilter, user, ws.Hub (main and overlay), email, push,
  block, overlay, notification, report, hyperbeam, livekit, stream
     │
     ▼
  domain services: chat, post, openai + chatbot, follow, art, ship, oc,
                   mystery, fanfic, journal, secret, gameroom, announcement,
                   homefeed, sidebar, vanityrole, usersecret, search, auth,
                   health, sitemap, siteinfo, og.Resolver, og.ImageService
     │
     ▼                                                        (server.go)
  fiber.New → middleware.Setup(app, settings, session, authz)
            → metrics + pprof routes
            → controllers.NewService(...) → routes.PublicRoutes(ctrl, app)
     │
     ▼                                                     (init_jobs.go)
  settings listeners + background jobs (stale embeds, orphaned uploads,
  notification prune, expired sessions, journal and room archiving,
  idle games, voice presence, live-stream reconcile)
     │
     ▼
  utils.StartServerWithGracefulShutdown(app, ":4323")         (main.go)
```

A few edges are genuinely circular and are closed with setters after construction rather than by a container: `sessionMgr.SetDisconnector(hub)`, `streamSvc.SetChatBinder(chatSvc)`, `chatSvc.SetMessageObserver(chatbotSvc)`, and `postSvc.SetCommentObserver(chatbotSvc)`.

Shutdown runs the other way. The cleanup func returned by `initServer` drains the chatbot, drains the media processor, stops the background jobs, and closes the cache, all sharing one 15-second budget.

### Observability

Four independent signals, all optional and all pointed at their collector from **Admin → Settings**, so none of them needs a redeploy or an env var.

- **Metrics**: a Prometheus registry is served on `/metrics`, and every request is timed by route, method, and status (`http_request_duration_seconds`, `http_requests_in_flight`), with static assets and uploads exempt so the histogram is not swamped. Alongside it sit database pool gauges (`db_pool_*`), WebSocket connection and inbound-frame counters (`ws_*`), Valkey hit / miss / latency and server stats (`cache_*`), and chatbot invocation, drop, and token counters (`chatbot_*`). `/metrics`, `/health`, `/livez`, and the LiveKit webhook are exempt from host authorisation so an internal scraper can reach them by IP; `docker-compose.prod.yml` carries the matching `prometheus-*` labels for scrape discovery and runs a `postgres-exporter` sidecar.
- **Traces**: an OpenTelemetry span for every HTTP request, with W3C trace context extracted from the incoming headers and the trace ID echoed back as `X-Trace-ID`, plus a span per SQL statement through `otelsql` and per Valkey command through the cache hook. Set an **OTLP endpoint** and a batch exporter is registered on the live tracer provider; clear it and the processor is flushed, shut down, and unregistered, all without a restart.
- **Profiling**: setting a **Pyroscope URL** starts continuous profiling (CPU, alloc objects and space, in-use objects and space, goroutines, mutex, and block) tagged with the hostname, and clearing it stops the profiler. The standard `/debug/pprof/*` handlers are also mounted, gated behind the `manage_settings` permission rather than left open.
- **Errors**: GlitchTip / Sentry through a DSN setting, with structured attribute mapping in `internal/logger/glitchtip_shipper.go`. Only Error and Fatal are shipped; Warn and Info stay local.
- **Health**: `/livez` is a bare liveness probe, which is what the container healthcheck hits. `/health` runs real dependency checks and returns 503 when one fails: Postgres is a hard dependency, LiveKit is checked only when voice is configured and is allowed to fail without failing the whole probe.

```
   scraper ──▶ GET /metrics ──▶ http_* , db_pool_* , ws_* , cache_* , chatbot_*
   probe   ──▶ GET /livez   ──▶ 200 always (process is up)
   probe   ──▶ GET /health  ──▶ 200 / 503 (postgres hard, livekit soft)

   otlp_endpoint  ──▶ batch span processor ──▶ collector
   pyroscope_url  ──▶ continuous profiler  ──▶ pyroscope
   sentry_dsn     ──▶ error shipper        ──▶ glitchtip
```

## Getting Started

### Prerequisites

- Go 1.26 or newer
- Node.js LTS
- Docker (for the Postgres and Valkey containers, plus repo-layer tests via testcontainers-go)
- FFmpeg, both `ffmpeg` and `ffprobe`, for video transcoding and thumbnails
- libwebp-tools for WebP work: `cwebp` for conversion, `dwebp` and `webpmux` for the OG JPEG path
- The goose CLI, if you need to author a migration (see [Database and Migrations](#database-and-migrations))
- `psql` CLI (optional, handy for poking at the DB)

### Environment

Two env files live next to each other:

- **`postgres.env`**, the Postgres bootstrap credentials. Read by the postgres container at init and by the app at connect time. Copy from `postgres.env.example`:
  ```bash
  cp postgres.env.example postgres.env
  ```
  | Variable            | Description                                           |
  |---------------------|-------------------------------------------------------|
  | `POSTGRES_USER`     | DB role for the app (`umineko` by convention)         |
  | `POSTGRES_PASSWORD` | DB password                                           |
  | `POSTGRES_DB`       | Database name (`umineko_city_of_books` by convention) |

- **`.env`**, everything else the app reads. Copy from `.env.example`:
  ```bash
  cp .env.example .env
  ```

Only a short list of variables is read from the environment for its own sake. Everything else in `.env` is a first-boot seed for a row in `site_settings`.

**Read from the environment**

| Variable               | Default     | Description                                                                                                                                                                     |
|------------------------|-------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `POSTGRES_HOST`        | `localhost` | Postgres host. `postgres` (the compose service name) under docker-compose                                                                                                       |
| `POSTGRES_PORT`        | `5432`      | Postgres port. The internal container port, not the host-mapped 5007                                                                                                            |
| `POSTGRES_SSL_MODE`    | `disable`   | Postgres SSL mode (`disable`, `require`, `verify-ca`, `verify-full`)                                                                                                            |
| `DATABASE_URL`         | (empty)     | Full connection string. If set, overrides the discrete `POSTGRES_*` vars                                                                                                        |
| `GIPHY_API_KEY`        | (empty)     | GIPHY API key. There is no admin setting for it: without it the GIF picker is disabled and direct-URL GIF bans cannot resolve uploaders                                          |
| `FCM_CREDENTIALS_FILE` | (empty)     | Path to the Firebase service-account JSON for native push. Compose mounts `./fcm-service-account.json` read-only at `/app/fcm-service-account.json`. Also needs the `push_enabled` site setting turned on |

**First-boot seeds for site settings**

At startup the app uppercases every site-setting key and, when an env var of that name exists, uses its value as that setting's default. Missing settings are then written into the database with those defaults the first time the app boots. From then on the stored row wins and the env var is ignored, so these variables only bite on a fresh database and editing one later changes nothing.

| Variable            | Seeds                | Default                 | Description                                                                                                                                                                                                |
|---------------------|----------------------|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `BASE_URL`          | `base_url`           | `http://localhost:4323` | Public base URL, used for CORS and absolute links. No admin field today, so the seeded value sticks                                                                                                          |
| `UPLOAD_DIR`        | `upload_dir`         | `uploads`               | Directory for uploaded files (relative to working dir). No admin field today, so the seeded value sticks                                                                                                     |
| `LOG_LEVEL`         | `log_level`          | `info`                  | Initial log level, overridable from the admin panel at runtime                                                                                                                                              |
| `SENTRY_DSN`        | `sentry_dsn`         | (empty)                 | GlitchTip / Sentry DSN for error shipping. Editable under **Admin → Settings**, alongside the OTLP endpoint and Pyroscope URL                                                                                |
| `VALKEY_URL`        | `valkey_url`         | (empty)                 | App cache connection URL, separate from the LiveKit Valkey. Normally left empty and enabled from **Admin → Settings → Cache (Valkey)** (`redis://valkey-cache:6379/0` in Docker, `redis://localhost:6381/0` on the host) |
| `HYPERBEAM_API_KEY` | `hyperbeam_api_key`  | (empty)                 | Hyperbeam API key for virtual-browser watch parties. Now set in admin, see below                                                                                                                            |
| `HYPERBEAM_REGION`  | `hyperbeam_region`   | `EU`                    | Default Hyperbeam VM region (`NA`, `EU`, or `AS`), overridable per session from the start-party dialog. Now set in admin, see below                                                                          |

Any site-setting key works this way, not just the rows above, but these are the ones worth setting before the first boot.

> **Hyperbeam moved.** `HYPERBEAM_API_KEY` and `HYPERBEAM_REGION` were plain `.env` config. They are now the `hyperbeam_api_key` and `hyperbeam_region` site settings, edited under **Admin → Settings → Watch Parties, Voice & Streaming**, and the watch-party code reads them live from the settings service. The env vars survive only as first-boot seeds, which is why `.env.example` keeps them commented out.

`.env.example` ships a deliberate subset: `GIPHY_API_KEY`, `POSTGRES_PORT`, `VALKEY_URL`, and `SENTRY_DSN` are all read by the app but are not in the template, so add them by hand when you need them.

Everything else (registration mode, maintenance mode, turnstile keys, upload limits, rate limits, log level, email provider and SMTP settings, LiveKit and streaming credentials, chatbot configuration, default theme, Sentry DSN) is stored in the database via the `site_settings` table and editable from the admin panel at runtime with hot reload. The env file is only for things that must exist before the DB is reachable, and for the handful of secrets that never round-trip through the DB.

### Running Locally

The Go binary embeds the built frontend (`//go:embed static/*` in `server.go`) and reads `static/index.html` at startup, and `static/` is gitignored. On a fresh clone nothing Go-side will even compile until that directory exists, so build the frontend once first:

```bash
cd frontend
npm ci
npm run build     # writes ../static/
```

If you only care about compiling and testing the backend, a placeholder is enough: `mkdir -p static && touch static/.gitkeep`, which is exactly what CI does.

The app also needs Postgres reachable before it'll boot. Two paths:

**Option A: Run only Postgres in Docker, the app on the host**

```bash
# start just the postgres service from compose
docker compose up -d postgres

# backend (from repo root), connects to Postgres on host port 5007
POSTGRES_HOST=localhost POSTGRES_PORT=5007 go run .

# frontend (separate terminal)
cd frontend
npm run dev
```

Work against `http://localhost:5173`, the Vite dev server. It proxies `/api`, `/api/v1/ws`, `/uploads`, and `/sitemap` through to the Go server on `:4323`. Hitting `:4323` directly serves the last `npm run build` output from `static/`, not your live edits.

**Option B: Run the full stack in Docker**

A bare `docker compose up -d --build` starts every service in the file, including `livekit`, `livekit-ingress`, and `livekit-egress`, which bind-mount the gitignored `livekit.yaml`, `ingress.yaml`, and `egress.yaml`. Unless you have already done the [Voice Chat](#voice-chat-livekit) and [Live Streaming](#live-streaming-livekit-ingress) setup, Docker creates empty directories in their place and those three containers crash-loop. Name the app service instead, and compose brings up the two it depends on (`postgres` and `valkey-cache`) with it:

```bash
docker compose up -d --build umineko-city-of-books
```

Visit `http://localhost:2312`. The container picks up `POSTGRES_HOST=postgres` from `.env`, which compose loads via `env_file`, which is why `.env.example` ships `postgres` rather than `localhost` as the host.

The backend serves on `:4323` (mapped to `:2312` from the host under docker-compose).

**The first user to register is automatically assigned the super admin role**, so start there to unlock the admin panel.

## Database and Migrations

All migrations live in `internal/db/migrations/` and are embedded into the binary via `go:embed`. They run automatically on startup via goose against the configured Postgres database (`db.Migrate` in `internal/db/db.go`).

The schema started as a single consolidated initial migration squashed during the SQLite-to-Postgres cutover. Everything since is a fresh migration stacked on top of it, around fifty of them now.

**Always create migrations with the goose CLI**, never by hand, so the timestamp format stays consistent:

```bash
goose -dir internal/db/migrations create <name> sql
```

goose is a library dependency here, not a `tool` directive in `go.mod` (only mockery and staticcheck are), so install the CLI separately if you do not already have it:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then edit the generated file to fill in the `-- +goose Up` and `-- +goose Down` sections. On next `go run .` the migration applies automatically.

**The Down half is not optional.** `internal/dao/migration_roundtrip_test.go` migrates a throwaway database all the way up, rolls it back to `20260726205627` with `db.MigrateDownTo`, then migrates up again. A missing or broken `-- +goose Down` in any migration newer than that fails the test suite, so write the rollback at the same time as the forward change rather than leaving it empty.

To inspect the database directly (host-side, via the mapped port):

```bash
psql -h localhost -p 5007 -U umineko -d umineko_city_of_books
\dt
\d theories
```

Or from inside the running postgres container:

```bash
docker compose exec postgres psql -U umineko -d umineko_city_of_books
```

## Development Workflow

### Backend

```bash
go build ./...            # compile
go vet ./...              # static analysis
go tool staticcheck ./... # linter, pinned by the tool directive in go.mod
go test ./...             # run tests
./scripts/test.sh         # regenerate mocks, then vet, staticcheck, test
./scripts/regen_mocks.sh  # regenerate mockery mocks only
```

All of those need `static/` to exist first, because the root package embeds it. See [Running Locally](#running-locally). The repository and DAO tests boot a real Postgres through testcontainers-go, so Docker has to be running for `go test ./...` to pass.

Interfaces listed in `.mockery.yml` get a mock generated next to the interface and named after it, so `Service` becomes `service_mock.go`. `internal/repository` is configured with `all: true`, so every interface in that package is mocked without being listed individually. Regenerate whenever you add or change an interface signature.

CI (`.github/workflows/ci.yml`) creates a `static/.gitkeep` placeholder, then runs `go vet ./...`, `go tool staticcheck ./...`, `go test ./... -count=1`, and `go build ./...` in that order. Putting `[skip tests]` in the commit message skips the test step only.

### Frontend

```bash
cd frontend
npm run dev         # dev server with HMR on :5173
npm run build       # tsc + vite build into ../static/
npm run typecheck   # tsc -b only, no bundle
npm test            # vitest run
npm run test:watch  # vitest in watch mode
npm run test:coverage
npm run lint        # eslint, --max-warnings=0
npm run lint:fix    # eslint with autofix
npm run prettier    # prettier check
npm run prettier:fix
```

Tests are vitest and React Testing Library under jsdom, colocated with the code (`Foo.tsx` next to `Foo.test.tsx`), with the shared render helpers, fixtures and jsdom setup in `frontend/src/test-utils/`. `tsconfig.json` includes the test files, so `npm run typecheck` (and therefore `npm run build`) typechecks them too.

CI runs `npm run prettier`, `npm run lint`, `npm test`, then `npm run build`. Run the same four before committing frontend changes; all of them need to pass cleanly.

### Mobile app (Capacitor)

The same React frontend is packaged as a native iOS/Android app via [Capacitor](https://capacitorjs.com/). The Capacitor project lives in `frontend/` (config in `frontend/capacitor.config.ts`, generated native project in `frontend/android/`).

```bash
cd frontend
npm run build:app   # builds the app bundle into frontend/dist-app/ using .env.app
npm run cap:sync    # build:app + copy assets into the native projects
npm run cap:android # build:app + sync android + open Android Studio
npm run cap:local   # sync android against a live dev server, for local iteration
npm run build:ota   # build:app + zip it into ../static/app-bundles/ as an OTA bundle
npm run assets      # regenerate launcher icons and splash screens
```

The web build (`npm run build`) and the app build (`npm run build:app`) are separate artefacts: the web build goes to `../static/` (served by the Go server, same-origin), the app build goes to `dist-app/` (bundled into the app). iOS must be built on macOS (`npx cap add ios` then Xcode); Android builds on any OS with Android Studio installed.

#### Local app development

`npm run cap:local` re-syncs the Android project with the Capacitor `server.url` pointed at a live Vite dev server, so the installed app loads your working tree instead of the bundled `dist-app/`. It defaults to `http://10.0.2.2:5173`, the emulator's alias for the host machine. Pass a LAN IP for a physical device:

```bash
npm run cap:local -- 192.168.1.50
```

Both `npm run dev` (Vite on `:5173`) and the Go backend (`:4323`) need to be running, and the app itself is launched from the IDE rather than by the script.

#### Over-the-air bundles

The app ships `@capgo/capacitor-updater` with `autoUpdate` turned off. `npm run build:ota` writes `../static/app-bundles/<VITE_APP_VERSION>.zip` plus a `latest.json` manifest, which the Go server then serves from the embedded static bundle like any other asset. The Docker image build runs it, so a deployed image publishes its own OTA bundle. On the client, `frontend/src/utils/appUpdate.ts` fetches `/app-bundles/latest.json` on launch and on resume, downloads a newer version in the background, and stages it for the next start rather than swapping under the user.

#### Changing the app's API base URL

The website's base URL is dynamic (`base_url` site setting). The packaged **app** is different: it has no server origin, so it needs an absolute API URL that is baked into the bundle at build time.

That URL comes from `VITE_API_BASE` in `frontend/.env.app`, the mode file that `npm run build:app` (`vite build --mode app`) loads:

```
VITE_API_BASE=https://whentheycry.social
```

To point the app at a different domain:

1. Edit `VITE_API_BASE` in `frontend/.env.app`.
2. Rebuild the app: `npm run build:app` (or `cap:android`).
3. Ship a new app version to the stores.

`frontend/.env.app` is committed; `frontend/.env` is gitignored and is your personal local override. Vite loads `.env` in every mode and lets the mode file win, so editing `.env` will not change what the app bundle points at, but it will change what a plain `npm run build` bakes into the website. Keep `frontend/.env` to dev values, or leave `VITE_API_BASE` out of it entirely so the website falls back to same-origin relative URLs.

Because the value is compiled into the bundle, already-installed apps keep the old domain until users update. If you migrate domains, keep the old one reachable (even as a redirect/proxy to the API) until old installs age out. The backend allows the app to connect cross-origin via `config.IsAppOrigin` (the fixed Capacitor webview origins) in addition to `base_url`.

#### Native push notifications

The packaged app registers an FCM device token per install and receives notifications natively, so the phone still buzzes when the app is closed and no WebSocket is open. The website is unaffected either way.

- Two things are required: the `push_enabled` site setting turned on in admin, **and** an `FCM_CREDENTIALS_FILE` env var pointing at a Firebase service-account JSON mounted read-only into the container. With the setting on and no credentials file, the app logs a warning at startup and native push simply stays off.
- The Firebase client is rebuilt whenever `push_enabled` changes, so switching it on does not need a restart.
- Tokens are registered and unregistered through the API and stored per user and per device, so each install is addressed on its own.

## Deployment

### Self-hosted Docker

```bash
docker compose up -d --build
```

This builds the multi-stage image locally (frontend -> static assets -> Go binary -> Alpine runtime with FFmpeg and libwebp-tools) and runs it on port `2312` by default, forwarding to the container's `:4323`.

`docker-compose.yml` defines seven services, not just the app and its database: `postgres`, `umineko-city-of-books`, `valkey-cache` (the app cache), `valkey` (the LiveKit coordination bus), `livekit`, `livekit-ingress`, and `livekit-egress`. The three LiveKit services bind-mount `livekit.yaml`, `ingress.yaml` and `egress.yaml`, all of which are gitignored, so on a fresh clone they restart-loop until you copy the templates (see the LiveKit sections below). The same applies to the `./fcm-service-account.json` mount: create the file, or drop the mount, unless you are using native push, otherwise Docker creates a directory in its place.

If you do not want voice, streaming or HLS, bring up the core only and let compose pull in its own dependencies:

```bash
docker compose up -d --build umineko-city-of-books
```

That starts `postgres` and `valkey-cache` (both declared under `depends_on`) and nothing else.

### Prebuilt image

```bash
docker network create observability
docker compose -f docker-compose.prod.yml up -d
```

This pulls `ghcr.io/victoriquemoe/umineko_city_of_books:latest` instead of building locally. The compose file declares `observability` as an **external** network, so it has to exist before the stack will start. The app and a `postgres-exporter` sidecar both join it and carry `prometheus-scrape` labels, so a Prometheus living on that network scrapes the app on port `4323` at `/metrics` and the exporter on `9187`.

Two things differ from the dev compose beyond the image source:

- The app publishes on **`127.0.0.1:2312`**, not on every interface, so it is only reachable through a reverse proxy on the same box. It also carries a `wget` healthcheck against `/livez`.
- `valkey`, `livekit`, `livekit-ingress` and `livekit-egress` all run `network_mode: host`, so their ports bind straight onto the host and container-name DNS does not resolve inside them. Every address in `livekit.yaml` / `ingress.yaml` / `egress.yaml` has to be `127.0.0.1:<port>` rather than a compose service name, and the coordination valkey binds `127.0.0.1:6380` to stay clear of a `6379` that is usually already taken.

### Persistent Data

Two stores hold real site data and need to survive container rebuilds:

- **Postgres data**, the named docker volume `postgres_data` mounted at `/var/lib/postgresql` inside the postgres container. Survives `docker compose up -d` and image upgrades.
- **Uploaded media**, the `umineko-city-of-books` service bind-mounts `./data:/app/data` so `data/uploads/` lives on the host. Set `UPLOAD_DIR=data/uploads` in your `.env` so the app reads from this mount. Note that `UPLOAD_DIR` only seeds the initial default of the `upload_dir` site setting; once it has been saved from the admin panel, the stored value wins.

The container runs as a non-root user (uid `10001`, `cap_drop: ALL`, `no-new-privileges`), so the host `./data` directory has to be writable by that uid. Live HLS segments land under the same mount at `data/hls/`, and the app does the cleanup itself, removing each per-stream directory when the broadcast ends and sweeping orphans on the reconcile pass, so it needs write access there and not only read.

`./fcm-service-account.json` is bind-mounted read-only into the container by both compose files and is gitignored. Create it (or remove the mount) before the first `up`, otherwise Docker creates a directory in its place.

`docker-compose.prod.yml` adds a third named volume, `valkey_data`, for the host-networked LiveKit coordination valkey it runs with `--appendonly yes`. That holds ephemeral SFU coordination state rather than site data, so it does not need backing up.

For backups: a daily `pg_dump | gzip` cron is the recommended path for the database, plus a periodic tarball of `./data/uploads/` for media. Restore via `gunzip -c <dump>.sql.gz | docker compose exec -T postgres psql -U umineko -d umineko_city_of_books`.

### Reverse Proxy

Run behind Caddy, Nginx, or similar for TLS. The server sets the right cache headers itself (`/static/assets/*` immutable, `/uploads/*` and `/hls/*` segments long-lived, `.m3u8` playlists `no-cache`, HTML `no-store`, API `no-cache`), so the proxy mostly just forwards. Four things it does have to get right:

- **Preserve the `Host` header.** The host-authorisation middleware answers 403 to any request whose host does not match the hostname of the `base_url` site setting, so a proxy that rewrites Host takes the whole site down. Only `/health`, `/livez`, `/metrics` and `/api/v1/livekit/webhook` are exempt.
- **Upgrade WebSocket connections on `/api/v1/ws`**, not `/ws`.
- **Set `CF-Connecting-IP`.** The app is configured with `ProxyHeader: "CF-Connecting-IP"` and trusts loopback and private peers, so behind anything other than Cloudflare the proxy has to set that header itself. Without it every request is attributed to the proxy's own address, and the session IP hash, rate limits and last-seen IP all read it.
- **Do not expose `/metrics`.** It is unauthenticated and host-authorisation exempt, so block it at the proxy and let Prometheus reach it over the docker network instead. `/debug/pprof/*` needs no such treatment, it is gated behind the `manage_settings` permission.

`/uploads/*` and `/hls/*` are served by the app on the same origin, so they need no extra proxy config. If you enable voice chat, also add a `wss://` route to the `livekit` container (see below).

### Voice Chat (LiveKit)

Voice chat needs the bundled `livekit` service (already in `docker-compose.yml` / `docker-compose.prod.yml`) plus a small amount of host setup. The code ships disabled, so none of this is required unless you want voice. The same LiveKit setup also powers **in-party voice and screen-share watch parties**, and live streaming builds on top of it. Once voice is configured there is no extra server-side work for those (screen capture is a browser feature served over the existing `wss://`).

1. **Create the config file.** The compose file bind-mounts `./livekit.yaml`, which is gitignored. Copy the template first (otherwise Docker creates an empty directory in its place):

   ```bash
   cp livekit.yaml.example livekit.yaml
   ```

2. **Set the server key/secret.** Edit the `keys:` block in `livekit.yaml` and replace the `devkey` / placeholder secret with a real key name and a long random secret.

3. **Enter the matching values in the app.** In **Admin → Settings → Watch Parties, Voice & Streaming**, toggle **Enable Voice Chat** and fill in:
   - **LiveKit URL**, the public `wss://` URL browsers connect to (see step 4)
   - **API Key** / **API Secret**, the **same** key name and secret you put in `livekit.yaml`

   The app reads these from the database (hot-reloaded, no restart), so they are never stored in `.env`. The three fields appear as soon as either voice chat or live streaming is switched on, and a save that turns voice on while any of them is empty is rejected.

4. **Reverse-proxy the signalling.** Point a public `wss://livekit.example.com` at the `livekit` container's port `7880` and use that URL as the LiveKit URL above.

5. **Open the media port on the host firewall.** LiveKit carries all WebRTC audio over a single UDP port `7882` (with TCP `7881` as a fallback). Open `7882/udp` on the host; without it, calls fail to connect. (A single fixed port avoids the Windows reserved-range binding errors and the slow per-port proxying you get when publishing a large range on Docker Desktop.)

6. **Advertise the right IP.** The production half of `livekit.yaml.example` keeps `rtc.use_external_ip: false` and instead pins `rtc.node_ip` to the box's public IPv4, with `rtc.ips.excludes` dropping IPv6 (`::/0`) and the docker bridge range (`172.16.0.0/12`). Under `network_mode: host` LiveKit otherwise sees every host interface and hands remote peers a docker-bridge or broken-IPv6 candidate, which is how calls end up one-way or silent. For local dev leave that block out, keep `use_external_ip: false`, and let the compose `NODE_IP` env var (`127.0.0.1`) stand in.

7. **TURN (recommended).** Roughly 15-25% of users behind strict NATs need a relay. Enable LiveKit's embedded TURN with a public hostname and TLS certs in `livekit.yaml`, or accept reduced connectivity.

The webhook LiveKit posts back to (`/api/v1/livekit/webhook`) is verified by the shared secret and is exempt from host authorisation, so it never has to go through the public proxy. Point `webhook.urls` at wherever the app is reachable from the LiveKit container: `http://umineko-city-of-books:4323/api/v1/livekit/webhook` on the dev compose's bridge network, and `http://127.0.0.1:2312/api/v1/livekit/webhook` in production, where LiveKit is host-networked and the app publishes on loopback.

### Live Streaming (LiveKit Ingress)

The `/live` page lets any member broadcast from OBS 30+ / Streamlabs (over WHIP) to a public viewer directory anyone can watch without an account. It builds on the Voice Chat setup above and adds the bundled `livekit-ingress` service plus a Valkey (Redis-compatible) coordination bus (both already in the compose files). Disabled by default.

1. **Create the ingress config.** The compose bind-mounts `./ingress.yaml`, which is gitignored. Copy the template (otherwise Docker creates an empty directory in its place):

   ```bash
   cp ingress.yaml.example ingress.yaml
   ```

   Set `api_key` / `api_secret` to the **same** key + secret as `livekit.yaml`.

2. **Wire up the coordination bus.** The ingress and the SFU coordinate over Valkey, so `livekit.yaml` needs a `redis:` block pointing at the **same** instance as `ingress.yaml`. The template does not ship one, add it. That instance is the `valkey` service, **not** `valkey-cache`: the latter is the app's own cache and stays separate. In production both the SFU and the ingress run `network_mode: host`, so they must use `127.0.0.1:<port>` (a host-networked container cannot resolve the `valkey` service name), and `docker-compose.prod.yml` already binds that valkey on `127.0.0.1:6380` to stay clear of a `6379` that is usually taken.

3. **Set the public WHIP endpoint.** Add an `ingress` block to `livekit.yaml` with the public URL OBS/Streamlabs will point at, and reverse-proxy that host to the ingress `whip_port`:

   ```yaml
   # livekit.yaml
   ingress:
     whip_base_url: "https://ingress.example.com/w"
   ```

   ```caddy
   ingress.example.com {
       reverse_proxy :8090   # whatever whip_port you set (use a free one if 8080 is taken)
   }
   ```

4. **Open the media port.** WHIP media is a single UDP port (`rtc_config.udp_port`, `7885` in the template); open it **directly** on the host firewall (UDP cannot go through the HTTP proxy). Set `rtc_config.use_external_ip: true` so the ingress advertises the host's public IP. The WHIP signalling port itself only needs to be reachable by your reverse proxy, not the public internet.

5. **Enable it in the app.** **Admin → Settings → Watch Parties, Voice & Streaming** → toggle **Enable Live Streaming**, and optionally set **Max Concurrent Streams** (default `3`). It reuses the same LiveKit URL + key/secret as voice, and does not need voice chat itself to be switched on.

Broadcasters then hit **Go live** on `/live`, paste the returned WHIP URL + stream key into OBS / Streamlabs (Service: **WHIP**), and appear on the public directory.

> WHIP media (WebRTC over UDP) requires the ingress on **host networking**. That works on a Linux host but not reliably on Docker Desktop (Windows/Mac), where the broadcaster app lives outside the container's network namespace, test broadcasting on the Linux host.

### Smooth Playback (LiveKit Egress / HLS)

By default a viewer watches a stream over **WebRTC**: sub-second latency, but it can freeze on a shaky connection, which is the nature of real-time UDP. Enabling the bundled `livekit-egress` service adds a second, **Smooth** option per stream, a buffered **HLS** rendition that plays a few seconds behind live but rides out network hiccups. Each viewer flips between **Low latency** (WebRTC) and **Smooth** (HLS) on the player, and the streamer picks which one viewers start on in the Go Live panel. It builds on the Ingress setup above. Disabled by default.

1. **Create the egress config.** The compose bind-mounts `./egress.yaml`, which is gitignored. Copy the template (otherwise Docker creates an empty directory in its place):

   ```bash
   cp egress.yaml.example egress.yaml
   ```

   Set `api_key` / `api_secret` / `ws_url` / `redis.address` to the **same** values as `ingress.yaml`. The egress must share the identical coordination plane or it never finds the room to record. In production (host networking) that is `ws://127.0.0.1:7880` and `127.0.0.1:<valkey-port>`.

2. **Where the files go.** The egress writes HLS segments to `stream_hls_output_dir` (default `/app/data/hls`), which the compose bind-mounts into the egress as `./data/hls` and into the app as part of `./data`. The app serves them at `/hls/*` on the **existing** site domain, exactly like `/uploads`, so there is **no new A record and no new reverse-proxy block**: your current Caddy/Nginx config already covers it. The app also owns the cleanup, removing each per-stream directory once the broadcast ends, so it needs write access to that path and not only read. The egress is outbound-only and receives no inbound media, so it opens **no new firewall ports** (unlike the ingress).

3. **Enable it in the app.** **Admin → Settings → Watch Parties, Voice & Streaming** → toggle **Enable Smooth (HLS) playback** (revealed when streaming is on). Leave **HLS Output Directory** at `/app/data/hls` unless you remap the mount.

The egress runs **participant egress** (no headless Chrome compositor), so it is light, but the container still needs `--cap-add=SYS_ADMIN` (already in the compose) because the egress image enables Chrome sandboxing regardless. When the broadcaster's video track appears, the app polls the ingress (up to six times, two seconds apart) for the **actual width / height / framerate** it is measuring from OBS and starts the egress at those, falling back to 60 fps if nothing has been reported yet, so the Smooth rendition mirrors whatever resolution the streamer set rather than a fixed one. The bitrate is not measured: the streamer types it into **Stream bitrate (Kbps)** in the Go Live panel, required whenever Smooth is available and accepted between 500 and 50000, and the egress encodes at that. Output is **H.264 Baseline** video with **320 kbps AAC** audio at 48 kHz, in two-second segments with a matching keyframe interval, for Safari/iOS compatibility. It is a **single rendition** (LiveKit egress encodes once per job, there is no adaptive-bitrate ladder), and the per-stream segment directory is deleted when the stream ends, with the reconcile pass sweeping up anything a crash orphaned, so nothing accumulates on disk.

> Smooth playback transcodes (HLS over `.ts` cannot carry the source codecs untouched), so it is a near-transparent re-encode at the bitrate the streamer entered, not bit-identical; the Low-latency WebRTC path stays the zero-loss option.

## Adding a New Page

When creating a new page or section, update **all** of the following:

1. **OG tags** - `internal/og/og.go`: add path matching in `metaForPath()` and a meta method for detail pages. Canonical URL, og:title, og:description, og:image, and twitter:* tags are auto-injected from the returned `Meta`.
2. **Admin Content Rules** - `frontend/src/pages/admin/AdminContentRules.tsx`: add to the `pages` array with a `rules_<page_name>` key, and register the matching `SettingRules...` in `internal/config/config.go`. Declaring the `SiteSettingDef` is only half of it: it must also go into the `AllSiteSettings` slice, or the key is never persisted or served.
3. **Sidebar** - `frontend/src/components/layout/Sidebar/Sidebar.tsx`: add `<NavLink>` in the appropriate section.
4. **Profile settings default page** - `frontend/src/pages/profile/SettingsPage.tsx`: add `<option>` to the Home Page dropdown.
5. **Lazy page export** - `frontend/src/pages/lazyPages.ts`: export the page through the `named()` adapter. `App.tsx` imports every routed page from here, never from the page file directly, so a page missing from this module cannot be routed.
6. **Home page routes** - `frontend/src/App.tsx`: add to the `homePageRoutes` object and add a `<Route>` element.
7. **Backend routes** - `internal/controllers/`: routes are never registered inline. Add a `getAllXRoutes() []FSetupRoute` method returning one `setupX(r fiber.Router)` method per endpoint, then append that list to `GetAPIRoutes()` (mounted under `/api/v1`) or `GetPageRoutes()` (mounted at the root) in `internal/controllers/service.go`. `internal/routes/public_routes.go` walks both lists and needs no change.
8. **Sitemap** - `internal/sitemap/service.go`: add the URL to the `staticPaths` slice. For a collection, add a `Service` method that builds its `Entry` list, add the sub-sitemap suffix to `IndexEntries()`, and wire a handler plus route in `internal/controllers/sitemap_controller.go`.
9. **Content filter rules** - `internal/contentfilter`: if the new page accepts user text, make sure its service runs input through the content filter pipeline.
10. **Search** - if the new page introduces a searchable entity, add a `SearchSource` to `searchSources` in `internal/repository/search.go` and a matching URL builder to `urlBuilders` in `internal/search/urls.go`. An `init()` in `urls.go` panics at startup if you register one without the other.

## License

Released under the [MIT License](LICENSE). Umineko no Naku Koro ni and the wider When They Cry series are © 07th Expansion; this project is an unofficial fan platform with no affiliation.
