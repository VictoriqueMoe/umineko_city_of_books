import { useStreamChatJoin } from "../../hooks/useStreamChatJoin";
import { RoomChatPanel } from "../../components/chat/RoomChatPanel/RoomChatPanel";

const MAX_LIVE_MESSAGES = 50;

interface StreamChatPanelProps {
    streamId: string;
    isLive: boolean;
    onPopOut?: () => void;
    flush?: boolean;
    hideHeader?: boolean;
}

export function StreamChatPanel({ streamId, isLive, onPopOut, flush, hideHeader }: StreamChatPanelProps) {
    const { joined, failed } = useStreamChatJoin(streamId, isLive);

    let notice: string | null = null;
    if (isLive && failed) {
        notice = "Couldn't join the chat.";
    } else if (isLive && !joined) {
        notice = "Joining chat...";
    }

    return (
        <RoomChatPanel
            roomId={isLive && joined ? streamId : undefined}
            title="Stream chat"
            canSend={isLive && joined}
            notice={notice}
            closedNotice={isLive ? null : "Chat is closed while the stream is offline."}
            maxMessages={MAX_LIVE_MESSAGES}
            onPopOut={onPopOut}
            flush={flush}
            hideHeader={hideHeader}
        />
    );
}
