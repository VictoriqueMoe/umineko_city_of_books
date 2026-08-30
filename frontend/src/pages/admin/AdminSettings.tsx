import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { DRONEBL_CLASSES } from "../../domain/dronebl";
import { KNOWN_CRAWLER_FEEDS } from "../../domain/crawlerFeeds";
import { isEnabled } from "../../domain/siteSettings";
import { useAdminSettingsForm } from "../../hooks/useAdminSettingsForm";
import { ChatbotKeyGate } from "./ChatbotKeyGate";
import styles from "./AdminSettings.module.css";

type EmailProvider = "smtp" | "cloudflare";
const EMAIL_PROVIDER_SMTP: EmailProvider = "smtp";
const EMAIL_PROVIDER_CLOUDFLARE: EmailProvider = "cloudflare";

export function AdminSettings() {
    const {
        loading,
        saving,
        settings,
        error,
        success,
        defaultSiteName,
        fieldID,
        updateField,
        toggleField,
        getNumber,
        getMB,
        setMB,
        getMP,
        setMP,
        save,
        ogImageInputRef,
        dronebl,
        feeds,
        ogImage,
        emailTest,
        chatbot,
    } = useAdminSettingsForm();

    if (loading) {
        return <div className={styles.loading}>Loading settings...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Site Settings</h1>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Feature Toggles</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Registration</span>
                        <Select
                            value={settings.registration_type ?? "open"}
                            onChange={e => updateField("registration_type", e.target.value)}
                        >
                            <option value="open">Open (anyone can register)</option>
                            <option value="invite">Invite Only</option>
                            <option value="closed">Closed (no registration)</option>
                        </Select>
                    </div>
                    <ToggleSwitch
                        label="Maintenance Mode"
                        description="Put the site into maintenance mode"
                        enabled={isEnabled(settings.maintenance_mode)}
                        onChange={v => toggleField("maintenance_mode", v)}
                    />
                    {isEnabled(settings.maintenance_mode) && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Maintenance Title</span>
                                <Input
                                    value={settings.maintenance_title ?? ""}
                                    onChange={e => updateField("maintenance_title", e.target.value)}
                                    fullWidth
                                    placeholder="The game board is being prepared"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Maintenance Message</span>
                                <Input
                                    value={settings.maintenance_message ?? ""}
                                    onChange={e => updateField("maintenance_message", e.target.value)}
                                    fullWidth
                                    placeholder="Without love, it cannot be seen. Please check back shortly."
                                />
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Private Mode</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Require a login for everything"
                        description="Nobody who is not signed in can see or do anything: no pages, no API, no uploads, no link previews, and search engines are told to index nothing. Signing in, resetting a password and verifying an email keep working."
                        enabled={isEnabled(settings.private_mode)}
                        onChange={v => toggleField("private_mode", v)}
                    />
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>DroneBL</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable DroneBL"
                        description="Refuse requests from addresses listed on the DroneBL public abuse blocklist. Signed-in members are never blocked, and a blocked visitor can still reach the login page."
                        enabled={isEnabled(settings.dronebl_enabled)}
                        onChange={v => toggleField("dronebl_enabled", v)}
                    />
                    {isEnabled(settings.dronebl_enabled) && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Listings To Ignore</span>
                                <span className={styles.fieldHint}>
                                    Tick a reason to stop it blocking. Anything left unticked still blocks. DroneBL only
                                    lists addresses it has evidence against, so a well behaved VPN is usually not listed
                                    at all.
                                </span>
                                <div className={styles.classGrid}>
                                    {DRONEBL_CLASSES.map(cls => {
                                        const ignored = dronebl.ignoredClasses.has(cls.id);
                                        return (
                                            <label key={cls.id} className={styles.classRow}>
                                                <input
                                                    type="checkbox"
                                                    checked={ignored}
                                                    onChange={e => dronebl.toggleClass(cls.id, e.target.checked)}
                                                />
                                                <span className={styles.className}>
                                                    {cls.label} <code className={styles.classId}>{cls.id}</code>
                                                    {cls.note && <em className={styles.classNote}>{cls.note}</em>}
                                                </span>
                                            </label>
                                        );
                                    })}
                                </div>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Allowlist</span>
                                <Input
                                    value={settings.dronebl_allowlist ?? ""}
                                    onChange={e => updateField("dronebl_allowlist", e.target.value)}
                                    fullWidth
                                    placeholder="203.0.113.4, 2600:387:15:4015::/64"
                                />
                                <span className={styles.fieldHint}>
                                    Comma separated addresses or CIDR ranges that are never checked. Put your own
                                    address here so a false positive cannot lock you out of this page.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Search Engine Crawlers</span>
                                <span className={styles.fieldHint}>
                                    These publish the addresses their crawlers use, and are never blocked. Untick one
                                    and it is treated like any other visitor, which can cost you search indexing.
                                </span>
                                <div className={styles.classGrid}>
                                    {KNOWN_CRAWLER_FEEDS.map(known => (
                                        <label key={known.name} className={styles.classRow}>
                                            <input
                                                type="checkbox"
                                                checked={feeds.isKnownEnabled(known)}
                                                onChange={e => feeds.toggleKnown(known, e.target.checked)}
                                            />
                                            <span className={styles.className}>{known.label}</span>
                                        </label>
                                    ))}
                                </div>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Other Crawler Feeds</span>
                                <span className={styles.fieldHint}>
                                    Any endpoint publishing the same format Google, Bing and Apple use: a prefixes array
                                    of ipv4Prefix or ipv6Prefix entries. Every URL is fetched when you save, and one
                                    that fails will block the save.
                                </span>
                                <div className={styles.feedList}>
                                    {feeds.custom.map((entry, i) => (
                                        <div key={i} className={styles.feedRow}>
                                            <Input
                                                value={entry.name}
                                                onChange={e => feeds.update(i, { name: e.target.value })}
                                                placeholder="name"
                                                aria-label={`Feed ${i + 1} name`}
                                            />
                                            <Input
                                                value={entry.url}
                                                onChange={e => feeds.update(i, { url: e.target.value })}
                                                placeholder="https://example.com/ranges.json"
                                                aria-label={`Feed ${i + 1} url`}
                                                fullWidth
                                            />
                                            <button
                                                type="button"
                                                className={styles.feedRemove}
                                                onClick={() => feeds.remove(i)}
                                                aria-label={`Remove feed ${i + 1}`}
                                            >
                                                &times;
                                            </button>
                                        </div>
                                    ))}
                                </div>
                                <div>
                                    <Button variant="ghost" size="small" onClick={feeds.add}>
                                        + Add Feed
                                    </Button>
                                </div>
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Turnstile (Cloudflare)</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Turnstile"
                        description="Require Cloudflare Turnstile verification on login and registration"
                        enabled={isEnabled(settings.turnstile_enabled)}
                        onChange={v => toggleField("turnstile_enabled", v)}
                    />
                    {isEnabled(settings.turnstile_enabled) && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Site Key</span>
                                <Input
                                    value={settings.turnstile_site_key ?? ""}
                                    onChange={e => updateField("turnstile_site_key", e.target.value)}
                                    fullWidth
                                    placeholder="0x..."
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Secret Key</span>
                                <Input
                                    type="password"
                                    value={settings.turnstile_secret_key ?? ""}
                                    onChange={e => updateField("turnstile_secret_key", e.target.value)}
                                    fullWidth
                                    placeholder="0x..."
                                />
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Watch Parties, Voice &amp; Streaming</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Watch party: shared browser (Hyperbeam)</span>
                        <Input
                            type="password"
                            value={settings.hyperbeam_api_key ?? ""}
                            onChange={e => updateField("hyperbeam_api_key", e.target.value)}
                            fullWidth
                            placeholder="sk_test_..."
                        />
                        <span className={styles.fieldHint}>
                            Lets members watch a shared virtual browser together. Leave it empty to offer screen sharing
                            only, which uses the LiveKit credentials below.
                        </span>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Shared browser region</span>
                        <Select
                            value={settings.hyperbeam_region ?? "EU"}
                            onChange={e => updateField("hyperbeam_region", e.target.value)}
                        >
                            <option value="NA">North America</option>
                            <option value="EU">Europe</option>
                            <option value="AS">Asia</option>
                        </Select>
                        <span className={styles.fieldHint}>
                            Where the shared browser runs. Pick the one nearest most of your members.
                        </span>
                    </div>
                    <ToggleSwitch
                        label="Enable Voice Chat"
                        description="Allow voice calls in chat rooms and DMs (requires a self-hosted LiveKit server)"
                        enabled={isEnabled(settings.voice_enabled)}
                        onChange={v => toggleField("voice_enabled", v)}
                    />
                    <ToggleSwitch
                        label="Enable Live Streaming"
                        description="Let members broadcast from OBS (WHIP) to a public /live page anyone can watch (requires the LiveKit ingress service)"
                        enabled={isEnabled(settings.streaming_enabled)}
                        onChange={v => toggleField("streaming_enabled", v)}
                    />
                    {(isEnabled(settings.voice_enabled) || isEnabled(settings.streaming_enabled)) && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>LiveKit URL</span>
                                <Input
                                    value={settings.livekit_url ?? ""}
                                    onChange={e => updateField("livekit_url", e.target.value)}
                                    fullWidth
                                    placeholder="wss://livekit.example.com"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Key</span>
                                <Input
                                    value={settings.livekit_api_key ?? ""}
                                    onChange={e => updateField("livekit_api_key", e.target.value)}
                                    fullWidth
                                    placeholder="APIxxxxxxxx"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Secret</span>
                                <Input
                                    type="password"
                                    value={settings.livekit_api_secret ?? ""}
                                    onChange={e => updateField("livekit_api_secret", e.target.value)}
                                    fullWidth
                                    placeholder="secret"
                                />
                            </div>
                        </>
                    )}
                    {isEnabled(settings.streaming_enabled) && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Concurrent Streams</span>
                                <Input
                                    type="number"
                                    value={settings.stream_max_concurrent ?? ""}
                                    onChange={e => updateField("stream_max_concurrent", e.target.value)}
                                    fullWidth
                                    placeholder="3"
                                />
                            </div>
                            <ToggleSwitch
                                label="Enable Smooth (HLS) playback"
                                description="Record each live broadcaster to HLS so viewers can pick a buffered, freeze-resistant stream a few seconds behind live (requires the LiveKit egress service)"
                                enabled={isEnabled(settings.stream_hls_enabled)}
                                onChange={v => toggleField("stream_hls_enabled", v)}
                            />
                            {isEnabled(settings.stream_hls_enabled) && (
                                <div className={styles.field}>
                                    <span className={styles.fieldLabel}>HLS Output Directory</span>
                                    <Input
                                        value={settings.stream_hls_output_dir ?? ""}
                                        onChange={e => updateField("stream_hls_output_dir", e.target.value)}
                                        fullWidth
                                        placeholder="/app/data/hls"
                                    />
                                </div>
                            )}
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Mobile App</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Push Notifications"
                        description="Send native push notifications to the mobile app when a recipient is offline (requires FCM_CREDENTIALS_FILE on the server)"
                        enabled={isEnabled(settings.push_enabled)}
                        onChange={v => toggleField("push_enabled", v)}
                    />
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Latest App Version</span>
                        <Input
                            value={settings.app_latest_version ?? ""}
                            onChange={e => updateField("app_latest_version", e.target.value)}
                            fullWidth
                            placeholder="1.0.0"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>App Download URL</span>
                        <Input
                            value={settings.app_download_url ?? ""}
                            onChange={e => updateField("app_download_url", e.target.value)}
                            fullWidth
                            placeholder="https://github.com/VictoriqueMoe/umineko_city_of_books/releases/latest"
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Web Push</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>VAPID Public Key</span>
                        <Input
                            value={settings.web_push_vapid_key ?? ""}
                            onChange={e => updateField("web_push_vapid_key", e.target.value)}
                            fullWidth
                            placeholder="BEl62iUYgUivxIkv69yViEuiBIa..."
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Firebase Web API Key</span>
                        <Input
                            value={settings.web_push_firebase_api_key ?? ""}
                            onChange={e => updateField("web_push_firebase_api_key", e.target.value)}
                            fullWidth
                            placeholder="AIzaSy..."
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Firebase Project ID</span>
                        <Input
                            value={settings.web_push_firebase_project_id ?? ""}
                            onChange={e => updateField("web_push_firebase_project_id", e.target.value)}
                            fullWidth
                            placeholder="city-of-books"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Firebase Messaging Sender ID</span>
                        <Input
                            value={settings.web_push_firebase_sender_id ?? ""}
                            onChange={e => updateField("web_push_firebase_sender_id", e.target.value)}
                            fullWidth
                            placeholder="1234567890"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Firebase Web App ID</span>
                        <Input
                            value={settings.web_push_firebase_app_id ?? ""}
                            onChange={e => updateField("web_push_firebase_app_id", e.target.value)}
                            fullWidth
                            placeholder="1:1234567890:web:abcdef"
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Chatbot</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Chatbot"
                        description="Let members talk to bot accounts by mentioning or replying to them in chat"
                        enabled={isEnabled(settings.chatbot_enabled)}
                        onChange={v => toggleField("chatbot_enabled", v)}
                    />
                    {isEnabled(settings.chatbot_enabled) && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Key</span>
                                <Input
                                    type="password"
                                    value={settings.chatbot_api_key ?? ""}
                                    onChange={e => updateField("chatbot_api_key", e.target.value)}
                                    fullWidth
                                    placeholder="sk-..."
                                />
                                <span className={styles.fieldHint}>
                                    The API key every bot reply is charged against. Without it no bot can answer.
                                    Anything the bots generate appears on this key's bill, so treat the limits below as
                                    your spending controls.
                                </span>
                            </div>
                            {chatbot.locked && (
                                <ChatbotKeyGate
                                    apiKeySaved={chatbot.keySaved}
                                    checking={chatbot.modelsLoading}
                                    reason={chatbot.modelsError}
                                    onRetry={chatbot.refreshModels}
                                />
                            )}
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Admin Key</span>
                                <Input
                                    type="password"
                                    value={settings.chatbot_admin_key ?? ""}
                                    onChange={e => updateField("chatbot_admin_key", e.target.value)}
                                    fullWidth
                                    placeholder="Optional, for reading the billed spend"
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    Optional organisation admin key used only to read back what has actually been
                                    billed, which is shown on the Chatbots page. Leave it empty and everything still
                                    works, you just see token counts instead of a money figure.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <label className={styles.fieldLabel} htmlFor={fieldID("chatbot-model")}>
                                    Model
                                </label>
                                <Input
                                    id={fieldID("chatbot-model")}
                                    value={settings.chatbot_model ?? ""}
                                    onChange={e => updateField("chatbot_model", e.target.value)}
                                    fullWidth
                                    placeholder="gpt-5.6-luna"
                                    aria-describedby={fieldID("chatbot-model-hint")}
                                    list={chatbot.models.length > 0 ? fieldID("chatbot-model-options") : undefined}
                                    disabled={chatbot.locked}
                                />
                                {chatbot.models.length > 0 && (
                                    <datalist id={fieldID("chatbot-model-options")}>
                                        {chatbot.models.map(model => (
                                            <option key={model} value={model} />
                                        ))}
                                    </datalist>
                                )}
                                <span id={fieldID("chatbot-model-hint")} className={styles.fieldHint}>
                                    The default model every bot uses unless it overrides it on the Chatbots page. Larger
                                    models write better replies and cost more per token, so this is the single biggest
                                    lever on your bill. The list is whatever your key can actually see, and anything not
                                    on it can still be typed in by hand.
                                </span>
                                <Button
                                    variant="secondary"
                                    onClick={chatbot.testModel}
                                    disabled={
                                        chatbot.testing || chatbot.locked || !(settings.chatbot_model ?? "").trim()
                                    }
                                >
                                    {chatbot.testing ? "Testing..." : "Test model"}
                                </Button>
                                {chatbot.testMessage && <span className={styles.success}>{chatbot.testMessage}</span>}
                                {chatbot.testError && <span className={styles.saveError}>{chatbot.testError}</span>}
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Reasoning Effort</span>
                                <Select
                                    value={settings.chatbot_reasoning_effort ?? "low"}
                                    onChange={e => updateField("chatbot_reasoning_effort", e.target.value)}
                                    disabled={chatbot.locked}
                                >
                                    <option value="none">None</option>
                                    <option value="low">Low</option>
                                    <option value="medium">Medium</option>
                                    <option value="high">High</option>
                                    <option value="xhigh">Extra high</option>
                                    <option value="max">Max</option>
                                </Select>
                                <span className={styles.fieldHint}>
                                    How much hidden thinking the model does before it answers. Reasoning tokens are
                                    billed like any other output but are never shown to anyone, so higher settings cost
                                    real money and add latency for very little gain in casual chat. Low or none suits
                                    conversation.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Verbosity</span>
                                <Select
                                    value={settings.chatbot_verbosity ?? ""}
                                    onChange={e => updateField("chatbot_verbosity", e.target.value)}
                                    disabled={chatbot.locked}
                                >
                                    <option value="">Provider default</option>
                                    <option value="low">Low</option>
                                    <option value="medium">Medium</option>
                                    <option value="high">High</option>
                                </Select>
                                <span className={styles.fieldHint}>
                                    How much detail a reply carries, set on the request rather than written into the
                                    personality. Low keeps answers short, high lets them run on. Leave it on the
                                    provider default unless replies are consistently too long or too clipped, and note
                                    that anything a personality says about length still applies on top of this.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Output Tokens</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_output_tokens")}
                                    onChange={e => updateField("chatbot_max_output_tokens", e.target.value)}
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    The hard ceiling on a single reply, counting reasoning tokens and visible text
                                    together, so a heavy thinker can spend most of this budget before it writes a word.
                                    This is a safety limit that stops a runaway generation, not a way to ask for shorter
                                    replies. Set the length you want in the bot's persona instead, and leave this high
                                    enough that normal answers are never cut off mid-sentence.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Context Messages</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_context_messages")}
                                    onChange={e => updateField("chatbot_context_messages", e.target.value)}
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    How many recent messages from the room are sent along with the question so the bot
                                    knows what is being discussed. Every one of them is billed as input on each call, so
                                    doubling this roughly doubles the input cost of every reply.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Reply Chain</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_reply_chain")}
                                    onChange={e => updateField("chatbot_max_reply_chain", e.target.value)}
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    How far back a chain of replies is followed when someone answers a bot, so a long
                                    back-and-forth keeps its thread. Longer chains give better continuity and cost more,
                                    because the whole chain is resent as input each turn.
                                </span>
                            </div>
                            <ToggleSwitch
                                label="Restrict To Chatbot Permission"
                                description="Only members granted the Summon Chatbots permission can get a reply. Grant it on the Permissions page, to Moderator or to any vanity role."
                                enabled={isEnabled(settings.chatbot_require_permission)}
                                onChange={v => toggleField("chatbot_require_permission", v)}
                            />
                            {chatbot.restrict && (
                                <div className={styles.field}>
                                    <label className={styles.fieldLabel} htmlFor={fieldID("chatbot-opt-in-role")}>
                                        Opt In Role
                                    </label>
                                    <Select
                                        id={fieldID("chatbot-opt-in-role")}
                                        value={chatbot.optInRoleID}
                                        onChange={e => updateField("chatbot_opt_in_role", e.target.value)}
                                        aria-describedby={fieldID("chatbot-opt-in-role-hint")}
                                        disabled={chatbot.rolesLoading}
                                    >
                                        <option value="">Select a role...</option>
                                        {chatbot.optInRoleID !== "" && !chatbot.optInRoleListed && (
                                            <option value={chatbot.optInRoleID}>
                                                The saved role no longer carries Summon Chatbots
                                            </option>
                                        )}
                                        {chatbot.optInRoles.map(role => (
                                            <option key={role.id} value={role.id}>
                                                {role.label}
                                            </option>
                                        ))}
                                    </Select>
                                    <span id={fieldID("chatbot-opt-in-role-hint")} className={styles.fieldHint}>
                                        The vanity role a member is given when they opt in to characters from their own
                                        settings page. Opting in grants the whole role, so anything else it carries is
                                        granted with it. Only roles that already hold Summon Chatbots are offered, and
                                        moving to a different role moves everyone who opted in across to it.
                                    </span>
                                    {!chatbot.rolesLoading && chatbot.optInRoles.length === 0 && (
                                        <span className={styles.saveError}>
                                            No vanity role holds Summon Chatbots yet. Grant it to one on the Permissions
                                            page before restricting characters.
                                        </span>
                                    )}
                                </div>
                            )}
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Reply Cooldown (seconds)</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_reply_cooldown_seconds")}
                                    onChange={e => updateField("chatbot_reply_cooldown_seconds", e.target.value)}
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    The minimum wait between one member's replies. It stops someone hammering a bot in a
                                    tight loop and turning a quiet room into a large bill.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Replies Per Member (rolling 24 hours)</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_replies_per_user_per_day")}
                                    onChange={e => updateField("chatbot_max_replies_per_user_per_day", e.target.value)}
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    How many replies one member can pull out of the bots in the last 24 hours. This is a
                                    rolling window, not a calendar day: nothing resets at midnight, and allowance comes
                                    back gradually as each old reply passes the 24 hour mark. A member who hits the
                                    limit is told so by the character, with a rough idea of when they can try again, and
                                    nothing further is charged for them. Set it to 0 for no per-member limit.
                                </span>
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Max Replies Site-wide (rolling 24 hours)</span>
                                <Input
                                    type="number"
                                    value={getNumber("chatbot_max_replies_per_day")}
                                    onChange={e => updateField("chatbot_max_replies_per_day", e.target.value)}
                                    disabled={chatbot.locked}
                                />
                                <span className={styles.fieldHint}>
                                    The ceiling across every member and every bot in the last 24 hours, also a rolling
                                    window. This is your last line of defence on cost, so pick a number whose worst-case
                                    bill you are happy to pay. 0 removes the ceiling entirely, which is rarely what you
                                    want here.
                                </span>
                            </div>
                        </>
                    )}
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>General</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Site Name</span>
                        <Input
                            value={settings.site_name ?? ""}
                            onChange={e => updateField("site_name", e.target.value)}
                            fullWidth
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Site Description</span>
                        <Input
                            value={settings.site_description ?? ""}
                            onChange={e => updateField("site_description", e.target.value)}
                            fullWidth
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Announcement Banner</span>
                        <Input
                            value={settings.announcement_banner ?? ""}
                            onChange={e => updateField("announcement_banner", e.target.value)}
                            fullWidth
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Cache</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Valkey URL</span>
                        <Input
                            value={settings.valkey_url ?? ""}
                            onChange={e => updateField("valkey_url", e.target.value)}
                            fullWidth
                            placeholder="redis://valkey-cache:6379/0"
                        />
                        <span className={styles.fieldHint}>
                            Connection URL for the app cache, separate from the LiveKit coordination Valkey. Changes
                            take effect immediately, with no restart needed.
                        </span>
                        <span className={styles.fieldHint}>
                            Caching cannot be switched off. Leave this empty and the built-in in-memory cache is used as
                            the primary store. Set a URL and Valkey takes over, with the in-memory cache demoted to a
                            fallback: if Valkey stops responding, reads and writes are served from memory automatically
                            and Valkey is retried about every 30 seconds. Invalidations are sent to both, so a recovered
                            Valkey never serves a stale value.
                        </span>
                        <span className={styles.fieldHint}>
                            The in-memory cache is per-process, and anything cached without its own expiry is capped at
                            60 seconds. It is a safety net, not a substitute for Valkey.
                        </span>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>In-Memory Cache Size (MB)</span>
                        <Input
                            type="number"
                            value={getNumber("cache_in_memory_max_mb")}
                            onChange={e => updateField("cache_in_memory_max_mb", e.target.value)}
                        />
                        <span className={styles.fieldHint}>
                            How much memory the built-in cache may use, counting the stored value, its key and the
                            per-entry overhead. When it is full the least recently used entries are dropped to make
                            room. Applies whether the in-memory cache is the primary store or only the fallback, so it
                            also bounds how much is held while Valkey is unreachable. Accepted range is 1 to 4096 MB,
                            and anything outside it falls back to 128. Lowering it takes effect immediately and evicts
                            straight away rather than waiting for the next write.
                        </span>
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Limits</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Theories Per Day</span>
                        <Input
                            type="number"
                            value={getNumber("max_theories_per_day")}
                            onChange={e => updateField("max_theories_per_day", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Responses Per Day</span>
                        <Input
                            type="number"
                            value={getNumber("max_responses_per_day")}
                            onChange={e => updateField("max_responses_per_day", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Min Password Length</span>
                        <Input
                            type="number"
                            value={getNumber("min_password_length")}
                            onChange={e => updateField("min_password_length", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Session Duration (days)</span>
                        <Input
                            type="number"
                            value={getNumber("session_duration_days")}
                            onChange={e => updateField("session_duration_days", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>New Account Restriction (hours)</span>
                        <Input
                            type="number"
                            value={getNumber("new_account_hours")}
                            onChange={e => updateField("new_account_hours", e.target.value)}
                        />
                        <span className={styles.fieldHint}>
                            For this long after signing up, a new member cannot post attachments or links, and their
                            posts do not render link previews. Set to 0 to disable.
                        </span>
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>File Size Limits</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Image Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_image_size")}
                            onChange={e => setMB("max_image_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Image Pixels (megapixels)</span>
                        <Input
                            type="number"
                            value={getMP("max_image_pixels")}
                            onChange={e => setMP("max_image_pixels", e.target.value)}
                        />
                        <span className={styles.fieldHint}>
                            Rejects images whose width x height exceeds this, however small the file is. A highly
                            compressible image can be tiny on disk yet need gigabytes of memory to decode.
                        </span>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Video Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_video_size")}
                            onChange={e => setMB("max_video_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Audio Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_audio_size")}
                            onChange={e => setMB("max_audio_size", e.target.value)}
                        />
                        <span className={styles.fieldHint}>Applies to MP3, M4A, OGG, WAV and FLAC uploads.</span>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max General Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_general_size")}
                            onChange={e => setMB("max_general_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Body Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_body_size")}
                            onChange={e => setMB("max_body_size", e.target.value)}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Email</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Email Provider</span>
                        <Select
                            value={settings.email_provider ?? EMAIL_PROVIDER_SMTP}
                            onChange={e => updateField("email_provider", e.target.value)}
                        >
                            <option value={EMAIL_PROVIDER_SMTP}>SMTP</option>
                            <option value={EMAIL_PROVIDER_CLOUDFLARE}>Cloudflare Email Service</option>
                        </Select>
                    </div>
                    {(settings.email_provider ?? EMAIL_PROVIDER_SMTP) === EMAIL_PROVIDER_SMTP && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Host</span>
                                <Input
                                    value={settings.smtp_host ?? ""}
                                    onChange={e => updateField("smtp_host", e.target.value)}
                                    fullWidth
                                    placeholder="127.0.0.1"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Port</span>
                                <Input
                                    type="number"
                                    value={getNumber("smtp_port")}
                                    onChange={e => updateField("smtp_port", e.target.value)}
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>From Address</span>
                                <Input
                                    value={settings.smtp_from ?? ""}
                                    onChange={e => updateField("smtp_from", e.target.value)}
                                    fullWidth
                                    placeholder="noreply@example.com"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Username</span>
                                <Input
                                    value={settings.smtp_username ?? ""}
                                    onChange={e => updateField("smtp_username", e.target.value)}
                                    fullWidth
                                    placeholder="Leave empty for no auth"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Password</span>
                                <Input
                                    type="password"
                                    value={settings.smtp_password ?? ""}
                                    onChange={e => updateField("smtp_password", e.target.value)}
                                    fullWidth
                                    placeholder="Leave empty for no auth"
                                />
                            </div>
                        </>
                    )}
                    {settings.email_provider === EMAIL_PROVIDER_CLOUDFLARE && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Account ID</span>
                                <Input
                                    value={settings.cloudflare_account_id ?? ""}
                                    onChange={e => updateField("cloudflare_account_id", e.target.value)}
                                    fullWidth
                                    placeholder="Cloudflare account ID"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Token</span>
                                <Input
                                    type="password"
                                    value={settings.cloudflare_api_token ?? ""}
                                    onChange={e => updateField("cloudflare_api_token", e.target.value)}
                                    fullWidth
                                    placeholder="Token with email sending permission"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>From Address</span>
                                <Input
                                    value={settings.cloudflare_email_from ?? ""}
                                    onChange={e => updateField("cloudflare_email_from", e.target.value)}
                                    fullWidth
                                    placeholder="noreply@yourdomain.com"
                                />
                            </div>
                        </>
                    )}
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Sends a test email to your own account using the saved settings. Save changes first.
                        </span>
                        <Button variant="secondary" onClick={emailTest.send} disabled={emailTest.sending}>
                            {emailTest.sending ? "Sending..." : "Send test email"}
                        </Button>
                        {emailTest.message && <span className={styles.success}>{emailTest.message}</span>}
                        {emailTest.error && <span className={styles.saveError}>{emailTest.error}</span>}
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Logging & Observability</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Log Level</span>
                        <Select
                            value={settings.log_level ?? "info"}
                            onChange={e => updateField("log_level", e.target.value)}
                        >
                            <option value="trace">Trace</option>
                            <option value="debug">Debug</option>
                            <option value="info">Info</option>
                            <option value="warn">Warn</option>
                            <option value="error">Error</option>
                        </Select>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            OTLP endpoint (OpenTelemetry traces, e.g. http://tempo:4318)
                        </span>
                        <Input
                            value={settings.otlp_endpoint ?? ""}
                            onChange={e => updateField("otlp_endpoint", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable tracing"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Pyroscope URL (continuous profiling, e.g. http://pyroscope:4040)
                        </span>
                        <Input
                            value={settings.pyroscope_url ?? ""}
                            onChange={e => updateField("pyroscope_url", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable profiling"
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Appearance</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Default Theme</span>
                        <Select
                            value={settings.default_theme ?? "featherine"}
                            onChange={e => updateField("default_theme", e.target.value)}
                        >
                            <option value="featherine">Featherine</option>
                            <option value="beatrice">Beatrice</option>
                            <option value="bernkastel">Bernkastel</option>
                            <option value="lambdadelta">Lambdadelta</option>
                            <option value="erika">Erika Furudo</option>
                            <option value="battler">Battler Ushiromiya</option>
                            <option value="virgilia">Virgilia</option>
                            <option value="rika">Rika Furude</option>
                            <option value="mion">Mion Sonozaki</option>
                            <option value="satoko">Satoko Houjou</option>
                            <option value="miyao">Miyao</option>
                            <option value="lingji">Lingji</option>
                            <option value="stanislaw">Stanis&#322;aw</option>
                        </Select>
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Link Previews</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Default embed image shown when a link to the site is shared on Discord, X, and other
                            platforms, and the page has no image of its own. JPG only.
                        </span>
                        <div className={styles.embedActions}>
                            <Button variant="secondary" onClick={ogImage.choose} disabled={ogImage.uploading}>
                                {ogImage.uploading ? "Uploading..." : "Upload image"}
                            </Button>
                            {ogImage.hasCustom && (
                                <Button variant="secondary" onClick={ogImage.clear}>
                                    Reset to built-in
                                </Button>
                            )}
                            {ogImage.error && <span className={styles.saveError}>{ogImage.error}</span>}
                        </div>
                        <input
                            ref={ogImageInputRef}
                            type="file"
                            accept="image/jpeg,.jpg"
                            className={styles.hiddenInput}
                            onChange={ogImage.onSelected}
                        />
                    </div>
                    <EmbedPreviews
                        image={settings.og_default_image || "/Featherine.jpg"}
                        siteName={settings.site_name ?? defaultSiteName}
                        baseURL={settings.base_url ?? ""}
                    />
                </div>
            </div>

            <div className={styles.saveRow}>
                <Button variant="primary" onClick={save} disabled={saving}>
                    {saving ? "Saving..." : "Save Settings"}
                </Button>
                {error && <span className={styles.saveError}>{error}</span>}
                {success && <span className={styles.success}>{success}</span>}
            </div>
        </div>
    );
}

const EMBED_PREVIEW_DESCRIPTION =
    "Welcome to the game board. Declare blue truths, solve mysteries, debate pairings, read and write fanfiction, and chronicle your journey through When They Cry.";

function EmbedPreviews({ image, siteName, baseURL }: { image: string; siteName: string; baseURL: string }) {
    const domain = baseURL.replace(/^https?:\/\//, "").replace(/\/$/, "") || "whentheycry.social";

    return (
        <div className={styles.embedPreviews}>
            <div className={styles.embedPreviewColumn}>
                <span className={styles.embedPreviewLabel}>Discord</span>
                <div className={styles.discordPreview}>
                    <div className={styles.discordBar} />
                    <div className={styles.discordBody}>
                        <span className={styles.discordSite}>{siteName}</span>
                        <span className={styles.discordTitle}>{siteName}</span>
                        <span className={styles.discordDesc}>{EMBED_PREVIEW_DESCRIPTION}</span>
                        <img src={image} alt="Embed preview" className={styles.discordImage} />
                    </div>
                </div>
            </div>
            <div className={styles.embedPreviewColumn}>
                <span className={styles.embedPreviewLabel}>X / Twitter</span>
                <div className={styles.twitterPreview}>
                    <img src={image} alt="Embed preview" className={styles.twitterImage} />
                    <div className={styles.twitterBody}>
                        <span className={styles.twitterDomain}>{domain}</span>
                        <span className={styles.twitterTitle}>{siteName}</span>
                        <span className={styles.twitterDesc}>{EMBED_PREVIEW_DESCRIPTION}</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
