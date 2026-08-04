import { useEffect } from "react";
import { useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useNotifications } from "../../hooks/useNotifications";
import { getStream } from "../../api/endpoints";
import type { WSMessage } from "../../types/api";
import { STREAM_CHAT_POPOUT_CLOSED } from "../../utils/streamChatPopout";
import { StreamChatPanel } from "./StreamChatPanel";
import styles from "./live.module.css";

export function StreamChatPopout() {
    const { streamID } = useParams<{ streamID: string }>();
    const qc = useQueryClient();
    const { addWSListener } = useNotifications();

    const streamQuery = useQuery({
        queryKey: ["streams", "detail", streamID],
        queryFn: () => getStream(streamID as string),
        enabled: !!streamID,
    });

    const stream = streamQuery.data;
    usePageTitle(stream ? `Chat: ${stream.title}` : "Stream chat");

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== "stream_live" && msg.type !== "stream_offline") {
                return;
            }

            const data = msg.data as { id?: string; streamId?: string };
            if (data.id === streamID || data.streamId === streamID) {
                qc.invalidateQueries({ queryKey: ["streams", "detail", streamID] });
            }
        });
    }, [addWSListener, qc, streamID]);

    useEffect(() => {
        const opener = window.opener as Window | null;
        if (!opener || !streamID) {
            return;
        }

        const notify = () => {
            if (opener.closed) {
                return;
            }
            opener.postMessage({ type: STREAM_CHAT_POPOUT_CLOSED, streamId: streamID }, window.location.origin);
        };

        window.addEventListener("pagehide", notify);
        return () => {
            window.removeEventListener("pagehide", notify);
        };
    }, [streamID]);

    if (streamQuery.isLoading) {
        return <div className="loading">Loading chat...</div>;
    }

    if (!stream) {
        return <div className="empty-state">Stream not found.</div>;
    }

    return (
        <div className={styles.popoutPage}>
            <StreamChatPanel streamId={stream.id} isLive={stream.status === "live"} />
        </div>
    );
}
