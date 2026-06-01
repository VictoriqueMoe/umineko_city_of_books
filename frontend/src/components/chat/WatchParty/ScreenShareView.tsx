import { useTracks, VideoTrack } from "@livekit/components-react";
import { Track } from "livekit-client";

import styles from "./WatchParty.module.css";

export function ScreenShareView({ placeholder }: { placeholder: string }) {
    const tracks = useTracks([Track.Source.ScreenShare]);
    const screen = tracks.length > 0 ? tracks[0] : null;

    if (!screen) {
        return <div className={styles.empty}>{placeholder}</div>;
    }

    return <VideoTrack trackRef={screen} className={styles.screenVideo} />;
}
