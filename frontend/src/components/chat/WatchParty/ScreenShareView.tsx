import { useEffect, useState } from "react";
import { useTracks, VideoTrack } from "@livekit/components-react";
import { RemoteAudioTrack, Track } from "livekit-client";

import { VolumeSlider } from "../../VolumeSlider/VolumeSlider";
import styles from "./WatchParty.module.css";

interface ScreenShareViewProps {
    placeholder: string;
    onReload?: () => void;
}

export function ScreenShareView({ placeholder, onReload }: ScreenShareViewProps) {
    const videoTracks = useTracks([Track.Source.ScreenShare]);
    const audioTracks = useTracks([Track.Source.ScreenShareAudio]);
    const screen = videoTracks.length > 0 ? videoTracks[0] : null;
    const audioTrack = audioTracks.length > 0 ? audioTracks[0].publication.track : null;

    const [volume, setVolume] = useState(1);

    useEffect(() => {
        if (audioTrack instanceof RemoteAudioTrack) {
            audioTrack.setVolume(volume);
        }
    }, [audioTrack, volume]);

    if (!screen) {
        return <div className={styles.empty}>{placeholder}</div>;
    }

    return (
        <>
            <VideoTrack trackRef={screen} className={styles.screenVideo} />
            {!screen.participant.isLocal && onReload && (
                <button
                    type="button"
                    className={styles.reloadBtn}
                    onClick={onReload}
                    title="Reload if the screen is black"
                >
                    {"↻"} Reload stream
                </button>
            )}
            {audioTrack && (
                <VolumeSlider
                    value={volume}
                    onChange={setVolume}
                    ariaLabel="Screen share volume"
                    className={styles.volumeControl}
                />
            )}
        </>
    );
}
