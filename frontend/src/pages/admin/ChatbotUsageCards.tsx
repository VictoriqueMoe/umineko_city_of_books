import { useChatbotUsage } from "../../hooks/queries/admin";
import { channelLabel, channelRows, channelTokens } from "../../domain/chatbotUsage";
import type { ChatbotUsage } from "../../types/api";
import styles from "./AdminChatbots.module.css";

export function ChatbotUsageCards() {
    const dayUsage = useChatbotUsage(1);
    const weekUsage = useChatbotUsage(7);
    const monthUsage = useChatbotUsage(30);

    const ranges = [
        { label: "Last 24 hours", usage: dayUsage.usage, loading: dayUsage.loading },
        { label: "Last 7 days", usage: weekUsage.usage, loading: weekUsage.loading },
        { label: "Last 30 days", usage: monthUsage.usage, loading: monthUsage.loading },
    ];

    return (
        <div className={styles.usageRow}>
            {ranges.map(range => (
                <UsagePanel key={range.label} label={range.label} usage={range.usage} loading={range.loading} />
            ))}
        </div>
    );
}

interface UsagePanelProps {
    label: string;
    usage: ChatbotUsage | null;
    loading: boolean;
}

function UsagePanel({ label, usage, loading }: UsagePanelProps) {
    return (
        <div className={styles.usageCard}>
            <span className={styles.usageLabel}>{label}</span>
            {loading || !usage ? (
                <span className={styles.usageEmpty}>Loading...</span>
            ) : (
                <>
                    <span className={styles.usageHeadline}>{usage.invocations.toLocaleString()} replies</span>
                    <dl className={styles.usageStats}>
                        <div className={styles.usageStat}>
                            <dt>Input</dt>
                            <dd>{usage.prompt_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Cached input</dt>
                            <dd>{usage.cached_prompt_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Cache writes</dt>
                            <dd>{usage.cache_write_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Output</dt>
                            <dd>{usage.completion_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Reasoning</dt>
                            <dd>{usage.reasoning_tokens.toLocaleString()}</dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Failed</dt>
                            <dd className={usage.failed > 0 ? styles.error : undefined}>
                                {usage.failed.toLocaleString()}
                            </dd>
                        </div>
                        <div className={styles.usageStat}>
                            <dt>Quota blocked</dt>
                            <dd>{usage.quota.toLocaleString()}</dd>
                        </div>
                    </dl>
                    <table className={styles.channelTable}>
                        <caption className={styles.channelCaption}>Where the replies came from</caption>
                        <thead>
                            <tr>
                                <th scope="col">Channel</th>
                                <th scope="col">Replies</th>
                                <th scope="col">Tokens</th>
                            </tr>
                        </thead>
                        <tbody>
                            {channelRows(usage.channels).map(channel => (
                                <tr key={channel.channel}>
                                    <th scope="row">{channelLabel(channel.channel)}</th>
                                    <td>{channel.invocations.toLocaleString()}</td>
                                    <td>{channelTokens(channel).toLocaleString()}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                    {usage.failed > 0 && (
                        <span className={styles.fieldHint}>
                            Failures are almost always a model id the provider does not recognise, a revoked or expired
                            API key, or a quota that has run out.
                        </span>
                    )}
                    {usage.billed_usd !== null && (
                        <span className={styles.usageBilled}>Billed ${usage.billed_usd.toFixed(2)}</span>
                    )}
                </>
            )}
        </div>
    );
}
