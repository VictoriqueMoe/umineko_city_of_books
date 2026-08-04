import { useEffect, useState } from "react";
import { joinStreamChat } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { RoomChatPanel } from "../../components/chat/RoomChatPanel/RoomChatPanel";

const MAX_LIVE_MESSAGES = 50;

interface StreamChatPanelProps {
    streamId: string;
    isLive: boolean;
    onPopOut?: () => void;
    flush?: boolean;
    hideHeader?: boolean;
}

export function StreamChatPanel(props: StreamChatPanelProps) {
    return <StreamChatPanelInner key={props.streamId} {...props} />;
}

function StreamChatPanelInner({ streamId, isLive, onPopOut, flush, hideHeader }: StreamChatPanelProps) {
    const { user } = useAuth();
    const [joined, setJoined] = useState(false);
    const [joinError, setJoinError] = useState(false);

    useEffect(() => {
        if (!isLive || !user) {
            return;
        }
        let cancelled = false;
        joinStreamChat(streamId)
            .then(() => {
                if (!cancelled) {
                    setJoined(true);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setJoinError(true);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [streamId, isLive, user]);

    let notice: string | null = null;
    if (isLive && joinError) {
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
