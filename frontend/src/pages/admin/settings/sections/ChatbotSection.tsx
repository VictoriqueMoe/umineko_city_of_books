import { Button } from "../../../../components/Button/Button";
import { Input } from "../../../../components/Input/Input";
import { Select } from "../../../../components/Select/Select";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import { ChatbotKeyGate } from "../../ChatbotKeyGate";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function ChatbotSection({ form }: SectionProps) {
    const { settings, updateField, toggleField, fieldID, getNumber, chatbot } = form;

    return (
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
                                The API key every bot reply is charged against. Without it no bot can answer. Anything
                                the bots generate appears on this key's bill, so treat the limits below as your spending
                                controls.
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
                                Optional organisation admin key used only to read back what has actually been billed,
                                which is shown on the Chatbots page. Leave it empty and everything still works, you just
                                see token counts instead of a money figure.
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
                                models write better replies and cost more per token, so this is the single biggest lever
                                on your bill. The list is whatever your key can actually see, and anything not on it can
                                still be typed in by hand.
                            </span>
                            <Button
                                variant="secondary"
                                onClick={chatbot.testModel}
                                disabled={chatbot.testing || chatbot.locked || !(settings.chatbot_model ?? "").trim()}
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
                                How much hidden thinking the model does before it answers. Reasoning tokens are billed
                                like any other output but are never shown to anyone, so higher settings cost real money
                                and add latency for very little gain in casual chat. Low or none suits conversation.
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
                                personality. Low keeps answers short, high lets them run on. Leave it on the provider
                                default unless replies are consistently too long or too clipped, and note that anything
                                a personality says about length still applies on top of this.
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
                                The hard ceiling on a single reply, counting reasoning tokens and visible text together,
                                so a heavy thinker can spend most of this budget before it writes a word. This is a
                                safety limit that stops a runaway generation, not a way to ask for shorter replies. Set
                                the length you want in the bot's persona instead, and leave this high enough that normal
                                answers are never cut off mid-sentence.
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
                                How many recent messages from the room are sent along with the question so the bot knows
                                what is being discussed. Every one of them is billed as input on each call, so doubling
                                this roughly doubles the input cost of every reply.
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
                                rolling window, not a calendar day: nothing resets at midnight, and allowance comes back
                                gradually as each old reply passes the 24 hour mark. A member who hits the limit is told
                                so by the character, with a rough idea of when they can try again, and nothing further
                                is charged for them. Set it to 0 for no per-member limit.
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
                                bill you are happy to pay. 0 removes the ceiling entirely, which is rarely what you want
                                here.
                            </span>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
