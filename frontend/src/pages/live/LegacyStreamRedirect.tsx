import { Navigate, useParams } from "react-router";
import { streamWatchPath } from "../../domain/live/streamUrl";
import { useStream } from "../../hooks/queries/stream";

export function LegacyStreamRedirect() {
    const { streamID } = useParams<{ streamID: string }>();
    const { stream, loading } = useStream(streamID);

    if (loading) {
        return <div className="loading">Loading stream...</div>;
    }

    if (!stream) {
        return <Navigate to="/live" replace />;
    }

    return <Navigate to={streamWatchPath(stream.streamerUsername)} replace />;
}
