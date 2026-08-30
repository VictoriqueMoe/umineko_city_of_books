import { useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useStreamDetail } from "../../hooks/useLiveStream";
import { useStreamChatPopoutReporter } from "../../hooks/useStreamPopout";
import { LIVE_STATUS } from "../../domain/live/playback";
import { StreamChatPanel } from "./StreamChatPanel";
import styles from "./live.module.css";

export function StreamChatPopout() {
    const { streamID } = useParams<{ streamID: string }>();

    const { stream, loading } = useStreamDetail(streamID);

    usePageTitle(stream ? `Chat: ${stream.title}` : "Stream chat");
    useStreamChatPopoutReporter(streamID);

    if (loading) {
        return <div className="loading">Loading chat...</div>;
    }

    if (!stream) {
        return <div className="empty-state">Stream not found.</div>;
    }

    return (
        <div className={styles.popoutPage}>
            <StreamChatPanel streamId={stream.id} isLive={stream.status === LIVE_STATUS} />
        </div>
    );
}
