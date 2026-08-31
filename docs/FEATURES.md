# Features

The full feature catalogue for Umineko City of Books: all 24 areas of the site in detail, moved out of the README so the front page can stay a front page.

## Theory Debates

The original heart of the site. Submit a fan theory as a **blue truth**, attach quote evidence, and let others refute or support it.

- Theory declarations with title, body, and episode / arc / chapter scope depending on series
- Evidence attachment by searching any quote (including narrator lines) across Umineko, Higurashi, and Ciconia via the Umineko Quote Finder API, with per-series chapter/episode/arc filters and a main/additional character split for Higurashi and Ciconia
- Two-sided debate with **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny), each with its own evidence
- **Credibility score** per theory (0 to 100), weighted by the truth type of evidence attached to top-level responses (gold 3.3, red 3.0, purple 2.2, blue 2.0, plain quote 1.0) and run through a `tanh` curve, so a lopsided debate saturates towards 0 or 100 instead of running away
- Threaded replies on responses with flat rendering and @username attribution
- Upvotes and downvotes on both theories and responses, separate from the credibility score
- Per-series feeds at `/theories` (Umineko), `/theories/higurashi`, and `/theories/ciconia`, each with its own sidebar entry
- **Lifecycle status** on every theory, stored as a native `theory_status` enum: **Open** while nobody has responded, **Contested** once a top-level response exists, and **Refuted** once the theory's own author (or a holder of `edit_any_theory`) accepts one opposing top-level response as the refutation. Refuted is terminal and stamps the theory permanently with who struck it down and a link to the killing response, mirroring how a mystery Game Master marks the winning attempt. A composite foreign key on `(id, refuted_by_response_id)` makes "the refutation belongs to this theory" a schema fact rather than a service check
- @mentions in theory bodies and responses notify the mentioned member, with response mentions deep-linking to `#response-<id>`

## Mysteries

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
- **Knox's Decalogue** as a published fair-play contract. Ten per-mystery booleans, all defaulting to sworn, toggled in the composer and rendered as a contract card above the Red Truths. Whatever the Game Master leaves sworn binds them; what they switch off is openly permitted. The contract locks as soon as the first attempt is submitted (`ErrContractLocked`, surfaced as a 409), so the terms cannot move mid-game, and a staff edit to the contract is listed in the edit notification sent to the author. A separate `knox_contract_published` flag defaults to false, so mysteries created before the feature publish nothing: a contract is a promise, and the backfill must not invent one on the author's behalf. Publishing for the first time is allowed even mid-game, since it can only ever add constraints on the Game Master; the lock applies to changes to an already-published contract

## Gallery and Art

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

## Ships

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

## Original Characters

A dedicated home for player-created OCs, separate from the canon-character ship list.

- Create OCs with name, series tag (Umineko / Higurashi / Ciconia / Custom), optional custom-series label, description, and avatar with automatic WebP conversion
- **Per-OC gallery**: add extra images with captions and ordering, shown as a masonry grid with lightbox on the detail page
- Upvote and downvote OCs, and **favourite** them with a public count
- OCs whose score falls to **-3 or below** pick up the **Crack OC** badge, and the index can be narrowed to those alone
- Browse the OC index at `/oc` filtered by series (including Custom) or owner, sorted by newest, oldest, most upvoted, most favourited, most commented, or name
- OC summaries appear on the owner's profile alongside their ships, and OCs can be attached to ships and tagged in fanfics through the shared character picker
- Full comment system with threading, media, GIFs, and likes
- Edit and delete by owners and admins

## Game Board

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

## Fanfiction

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

## Reading Journals

Live-blog your read-throughs of Ryukishi07's works. Post reactions, theories, and predictions as you go.

- Create a journal tied to a work: General, Umineko, Higurashi, Ciconia, Higanbana, or Rose Guns Days
- Each entry is its own numbered page with prev/next navigation, a word count, media, mentions, and Discord-style formatting
- **Drafts**: save an entry privately and publish it later. Drafts stay out of the entries list, the sitemap, search, and the activity feed, and do not notify anyone until you publish
- Threaded comment system so other players can react to each entry without spoiling
- Follow a journal to get notified when the author posts a new update, and browse a feed of just the journals you follow
- Journals **auto-archive after seven days** without author activity, to keep the index clean. An archived journal is read-only until its author posts again
- Per-user daily journal cap (admin configurable)

## Chat Rooms and DMs

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

## Chatbots

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

## Watch Parties

Shared-viewing sessions launched from inside a group chat room. A party is one of two `type`s: a **virtual browser** (a remote Hyperbeam Chromium VM everyone loads and takes turns driving, so you can stream a video, browse, or play a web game together without anyone capturing their own screen) or a **screen share** (the starter broadcasts their own screen, with tab/system audio, over LiveKit). Either way the whole room watches together.

- Started by a room member from the chat composer. The popover asks only for **Virtual browser** or **Screen share** and an optional title. There is nothing else to fill in: the client works out the nearest Hyperbeam region itself (cached for a day in local storage, falling back to the `hyperbeam_region` site setting), and the VM inherits light or dark from whichever theme the starter is using
- Because that region is resolved in the starter's own browser, it is a *preference*, not a guarantee: a region your Hyperbeam plan has no machines in answers `503 err_no_available_vm` for that user and nobody else. `StartWatchParty` therefore walks a fallback chain (browser's region, then `hyperbeam_region`, then the remaining `EU`/`NA`/`AS`), retrying only on that capacity code and storing whichever region actually served the VM. Exhausting the chain raises `ErrWatchPartyNoCapacity` (503, logged at `Error` so it surfaces in Grafana) rather than a generic 500, and the popover shows the reason instead of failing silently
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

## Voice Chat

Real-time voice in group rooms, DMs, and watch parties, backed by a self-hosted [LiveKit](https://livekit.io/) SFU so calls scale past the handful of people a peer-to-peer mesh can manage. Audio flows through LiveKit; the Go backend only mints signed join tokens and tracks who is in each call.

- **Join Voice** lives in the chat composer next to the watch-party button; the call renders as a slim bar above the message list (with speaking indicators, mute, and leave), never a takeover modal, so chatting continues during the call
- Joining is gated by the same room membership check as messaging, and DMs additionally respect blocks
- Room/DM call: LiveKit room name = chat room UUID, participant identity = user UUID; presence is tracked from signed LiveKit webhooks (`participant_joined` / `participant_left` / `room_finished`) and broadcast as a `voice_presence` WS event, which also drives an "in call" badge in the room list
- **Presence is reconciled, not just observed**: every 30 seconds a background job rebuilds the whole presence map from LiveKit's own room and participant listing and re-broadcasts only the rooms whose member set actually changed, so a dropped webhook cannot leave a ghost in the badge
- Watch-party call: a separate session-scoped room named `wp_<sessionID>` (so two parties in one room never share a channel); party presence is read live from LiveKit rather than tracked here, so it never pollutes the room's "in call" badge. In-party voice only needs LiveKit configured, independent of the `voice_enabled` toggle
- **Mute controls**: each listener can mute a single participant or everyone just for themselves (client-side volume, nobody else affected); hosts/mods/staff get a **mute-for-everyone** that is permission-based (`UpdateParticipant` revokes the mic publish grant) and **stored in the database**. The backend re-applies the revoked grant from the `participant_joined` webhook every time a muted user reconnects, for room calls and watch-party calls alike, so an old token does not hand the mic back and a restart does not quietly un-mute anyone. The mute is released when LiveKit finishes the room (`room_finished`, i.e. the call empties out), when the watch party ends, or when a moderator lifts it
- **Join tokens are short-lived** (1 hour) and losing access to a room drops the live session immediately: kicks, bans and word-filter kicks call LiveKit `RemoveParticipant` for the room call and for every watch-party call in that room, so an evicted user stops hearing the call at once rather than riding out their token
- Admin-managed and off by default: enable it and set the LiveKit URL / API key / secret under **Admin → Settings → Watch Parties, Voice & Streaming** (the `voice_enabled` toggle reveals the fields, which are shared with live streaming). The Join button is hidden and the token endpoint returns `ErrVoiceDisabled` until configured. See [Deployment → Voice Chat](DEPLOYMENT.md#voice-chat-livekit) for the server side

## Live Streaming

A public broadcast directory at `/live`. Any member can go live from OBS 30+ or Streamlabs over WHIP, and anyone, logged in or not, can watch. Audio and video ride the same self-hosted LiveKit stack as voice chat, through the bundled ingress. Off by default; see [Deployment → Live Streaming](DEPLOYMENT.md#live-streaming-livekit-ingress) for the server side.

- **Go live** from the `/live` panel: give the stream a title, pick which playback mode viewers start on, and (when Smooth is configured) enter the bitrate you have set in OBS. The panel hands back a WHIP server URL and stream key with a step-by-step OBS walkthrough and a bitrate calculator
- **Stream credentials** are per user and persistent, so the key only has to be pasted into OBS once. They can be reset from the same panel if the key leaks
- **Two playback modes** per stream: **Low latency** (WebRTC, sub-second) and **Smooth** (HLS, a few seconds behind but rides out network hiccups). The streamer picks which one viewers land on, and each viewer can flip between them on the player (see [Deployment → Smooth Playback](DEPLOYMENT.md#smooth-playback-livekit-egress--hls))
- **Per-stream chat** is a real chat room whose ID is the stream ID, created with the `live_stream` system kind and the broadcaster as host. Any viewer joins by opening the stream, and the chat can be popped out into its own window for a second monitor
- **Live directory** with thumbnails, titles, and viewer counts that update over WebSocket (`stream_live`, `stream_offline`, `stream_viewers`, `stream_title`), plus a dedicated mobile view
- **Thumbnails** are captured client-side by a watching browser as a 480px-wide WebP frame, posted back on an interval and throttled server-side, so the directory preview stays current without running a compositor
- **Limits**: one live stream per user, a site-wide `stream_max_concurrent` cap (default 3), a 120-character title, and a 500 to 50000 Kbps bitrate range
- **Reconciliation**: a background job every minute tears down streams that never received a broadcaster and streams whose LiveKit room has emptied. Teardown stops the egress, deletes the chat room and its media, removes the thumbnail, and broadcasts the offline event, so nothing accumulates when OBS just disappears

## Games

Multiplayer mini-games hosted entirely inside the site. Each game has live games, past games, and a personal "My Games" view in the sidebar.

- **Chess**, **Checkers**, **Othello**: correspondence-style matches with no clocks. Invite a user by username or pick from your mutual followers, the invitee plays the second-mover side. Drag-to-move, server-side legality, full move history. Disconnects start a 60-second forfeit timer.
- **Minesweeper**: real-time duel on a shared minefield. After both players pick an Umineko character (Bernkastel, Erika, Dlanor, Lambdadelta), independent reveal grids run in parallel, and the first to clear all safe cells wins, hitting a mine instantly loses. Mines are placed lazily after both first clicks so opening reveals are always safe.
- **Snakes & Ladders**: pure dice luck, no decisions. Press Roll, the server rolls a fair six-sided die, ladders carry you up and snakes drop you back. You must land exactly on 100 to win; a roll that would overshoot leaves you where you are and passes the turn. Same correspondence pacing, forfeit timer and spectators as the rest.
- **Pong**: real-time paddle duel, the only game on the site with a live simulation. The ball is stepped server-side at 60 Hz in one goroutine per active room, snapshotted to both players and every spectator at 20 Hz over the existing game room WebSocket topic, and clients interpolate 100 ms behind the server clock while predicting only their own paddle. Input is an absolute paddle target rather than a key state, so mouse, touch and keyboard all steer at the same top speed. Where on the paddle the ball lands sets the outgoing angle, and every return speeds the ball up to a cap. First to 7, win by 2, hard capped at 11, with a 15-minute match limit. State is persisted per point and on a 60-second heartbeat, and a match interrupted by a restart resumes lazily on the first client to rejoin, from the stored score with a fresh serve.
- **Spectators**: active games are public. Anyone can open the board and watch live; spectators have their own side chat invisible to the players. Finished games are archived to **Past Games**.
- **Vanity titles** awarded to the top player of chess, checkers, othello and minesweeper (most wins, ties broken by win-loss differential), shown as a pill next to their display name:
  - **Grandmaster** for chess
  - **King of the Board** for checkers
  - **Discmaster** for othello
  - **Minemaster** for minesweeper
- **Notifications** for invites, your-turn nudges, forfeit warnings on disconnect, and game-over results
- Per-game hub (`/games/chess`, `/games/snakes_and_ladders`, etc.) with a How to play panel, the games live right now, and the scoreboard; `/games` is your own games, and the sidebar carries a live games count badge

## Secrets and Unlock Hunts

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

## Announcements

Site-wide announcements with pinning.

- Admins post announcements visible to everyone
- Pinned announcements stay at the top
- Full markdown support in the announcement body
- Full comment system reusing the shared CommentItem component, with threading, media, embeds, and likes
- Optional site-wide announcement banner settable from the admin panel

## Suggestions

A dedicated feedback channel for site improvements and bug reports.

- Posts written in the same composer as the game board, living under a dedicated "Site Improvements" corner
- Status filters: **Open**, **Done**, **Archived**
- Admins can resolve a suggestion (mark done) or archive it, with the status reflected back to the reporter
- Follows the same commenting, voting, and notification rules as the game board

## Search

A single search bar covers the whole site. Backed by Postgres `tsvector` columns on every searchable entity, with a `SearchSource` registry mapping each entity type back to its canonical URL.

- **Full search** (`/search?q=...`) returns paginated hits across theories and theory responses, game board posts and comments, art and art comments, mysteries with their attempts and comments, ships and ship comments, OCs and OC comments, announcements and announcement comments, fanfics and fanfic comments, journals with their entries and comments, users, chat messages, and live streams
- **Quick search** in the header returns up to a small number of hits per entity type, with the right deep link (e.g. a post comment links to `#comment-<id>` on its parent post)
- **Query syntax** comes straight from `websearch_to_tsquery`: bare words are ANDed, `OR` widens, a leading `-` excludes, and `"quoted phrases"` must sit adjacent and in order. Typos still land, because every title (plus usernames) also scores on a `pg_trgm` similarity that is added to the rank, so "beatice" finds "Beatrice"
- **Chat messages are viewer-scoped**, resolved through the chat service rather than the shared SQL, so you only ever see messages from rooms you belong to and signed-out visitors get no chat hits at all
- Filter chips narrow results to one section, plus a **Comments only** chip that spans every section. Drafts, archived journals, and anything authored by a banned or locked user never appear
- Adding a new searchable entity means registering a `SearchSource` and URL builder; nothing else in the search pipeline needs to change, and an `init()` panic on boot catches a source registered without a matching URL builder

## Quote Browser

A standalone interface for browsing the full quote corpus across all three series, sourced from the Umineko Quote Finder API. Switch between Umineko / Higurashi / Ciconia tabs, filter by chapter/episode/arc, filter by truth type (red, blue, gold, purple) on Umineko, and filter by character with a main/additional split where the quote service exposes one. Ciconia and Higurashi quotes ship with Japanese text inline, and the language picker now honours it across all series.

## Profiles and Social Graph

- Avatar, draggable banner positioning, bio, pronouns (preset or custom), gender, date of birth with an optional public toggle, social links (Twitter/X, Discord, Tumblr, WaifuList, GitHub, Bluesky, personal site), favourite character picked from the Umineko / Higurashi / Ciconia casts or from your own OCs
- **Per-user theme, font, and wide layout preferences** persisted on the profile so they follow you across devices. The particles toggle is deliberately per-device and lives in local storage
- Activity feed with recent theories, responses, posts, and comments
- Tabs for posts, theories, art, galleries, ships, OCs, mysteries, fanfics, saved fics, journals, followed journals, and activity
- **Achievements** panel showing every solved unlock hunt as a live-updating trophy
- Stats box: theory count, response count, votes received, ship count, mystery count, fanfic count, follower/following counts
- Follow system with follower and following lists, "Follows you" label, follower counts
- Following is a live subscription, not just a count: when someone you follow goes live, posts a mystery, or posts a theory, the fan-out runs over `follows` through `notification.SendFollowerNotification`. Governed by a `follow_activity_notifications` preference (default on) filtered inside the DAO query, with an actor-scoped cooldown (`HasRecentFromActor`) so a flapping stream cannot spam every follower. These notifications deliberately carry no email leg
- Online/offline status
- **Players Page**: browse all users grouped by role (Reality Authors, Voyager Witches, Witches) and online/offline status, with a name search
- Per-user **blocks** with enforcement across feeds, comments, DMs, and notifications, managed from a blocked-users panel in settings
- Configurable **home page** (the page you land on) and **default profile tab**, each picked from a dropdown in settings
- Email with optional public visibility, a per-user email notification toggle, and separate toggles for the chat message sound and the notification sound
- **Reading progress** recorded per series (Umineko episode, Higurashi arc, Ciconia chapter), used for spoiler gating
- **Favourite GIFs**: star any GIF in the picker or posted by someone else to save it to a personal Favourites tab
- **Danger zone**: change your password, or delete the account behind a password confirmation

## Notifications

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

## Stream Overlay

Site events can drive on-stream alert popups through SAMMI. A streamer downloads a personal connector from **Settings → Stream Overlay**, imports it into SAMMI, and their site notifications start arriving as extension triggers they can wire to any overlay they like.

- **Personal connector**: the site generates a `.sef` extension file with your own connection token and the site name baked in, ready to import into SAMMI (Insert → Extension). It ships `Overlay: Connect` and `Overlay: Disconnect` commands and reconnects on its own if the socket drops
- **Events forwarded**: post liked, new follower, post commented, theory upvote, theory response, comment liked, mention, content shared, and art liked. Each arrives on the `overlay_event` trigger with the actor's username, display name, avatar, a human-readable action line, and a timestamp, so one SAMMI button can branch on the event type
- **Token auth**: the overlay connects to `/api/v1/overlay?token=...` with a random 32-byte token, entirely separate from your session cookie, and the origin is still checked against the live base URL. The token is re-validated periodically while the socket is open, so resetting it or banning the account drops the connection rather than waiting for it to close on its own
- **Reset and test**: the token can be rotated at any time (which retires the downloaded connector), the settings panel shows whether SAMMI is currently connected, and **Send test overlay** fires a dummy event so you can prove the wiring before going live

## Moderation and Admin

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
- DB-backed site settings with hot reload: body and upload limits, log level, registration mode, maintenance mode, turnstile, per-action rate limits, announcement banner, email provider (SMTP or Cloudflare), OTLP and Pyroscope endpoints, Valkey cache URL, default theme, LiveKit voice, Hyperbeam, live streaming, mobile push, and the chatbots
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

## Platform Features

- **Echoes** on the landing activity strip. Once per UTC day the `/home/activity` payload carries a short list of candidates drawn from exactly one year ago today, falling back to one month ago today, falling back to nothing, cached under the `home:echoes:` namespace with a 24 hour TTL. The endpoint is anonymous and therefore cannot know the viewer, so the server ships candidates and the client picks the first one the viewer is allowed to see, applying the same `userProgressForSeries` rule the theory cards use and skipping art flagged as a spoiler. Authors can opt their own work out with the `echoes_enabled` preference, filtered inside the SQL
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
- **Logs in Grafana**: with `LOG_FORMAT=json` the app writes structured JSON to stdout and ships nothing itself; Grafana Alloy tails the container from the Docker API into Loki. `trace_id` and `span_id` on a log line link straight to the matching Tempo trace
- **Native mobile app**: the same React frontend packaged with Capacitor, using bearer-token auth and FCM push (see [Mobile app (Capacitor)](MOBILE.md))
- Fully **mobile responsive** across all pages
- **Cache headers**: `/static/assets/*` and HLS segments are immutable, uploads and static media are 30 days, HLS playlists and API responses are `no-cache`, HTML is `no-store`

