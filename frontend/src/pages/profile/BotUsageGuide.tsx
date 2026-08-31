import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { useSiteInfo } from "../../hooks/useSiteInfo";

function messageLimit(value: number | undefined): number | null {
    if (typeof value !== "number" || !Number.isFinite(value) || value < 1) {
        return null;
    }

    return Math.floor(value);
}

export function BotUsageGuide() {
    const siteInfo = useSiteInfo();
    const contextMessages = messageLimit(siteInfo.chatbot_context_messages);
    const replyChain = messageLimit(siteInfo.chatbot_max_reply_chain);

    const replyChainLine = replyChain
        ? `Replying is how I remember, back through up to ${replyChain} messages.`
        : "Replying is how I remember, back through a limited number of messages.";
    const directMessageLine = contextMessages
        ? `I read back over the last ${contextMessages} messages, so the conversation carries on by itself.`
        : "I read back over a limited number of recent messages, so the conversation carries on by itself.";

    return (
        <InfoPanel title="How to talk to me">
            <p>
                <strong>Send me a direct message</strong> to start chatting, or find me in a chat room, on the game
                board and in comments.
            </p>
            <p>
                <strong>Mention me with @</strong> in a chat room, on the game board or in comments to start a new
                conversation. Each mention begins fresh, with no memory of the conversation before it.
            </p>
            <p>
                <strong>Reply</strong> to one of my messages to carry on the same conversation. {replyChainLine}
            </p>
            <p>In a direct message there is no need to mention me. {directMessageLine}</p>
            <p>I read text only, and cannot see images.</p>
            <p>
                To start again in a direct message, use <strong>Delete Chat</strong> and then message me again.
            </p>
        </InfoPanel>
    );
}
