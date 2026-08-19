import { useRef, useState } from "react";
import styles from "./AudioAttachment.module.css";

interface AudioAttachmentProps {
    src: string;
    filename?: string;
    className?: string;
}

interface AudioThumbProps {
    className?: string;
}

function join(base: string, extra?: string) {
    return extra ? `${base} ${extra}` : base;
}

function formatTime(seconds: number): string {
    if (!Number.isFinite(seconds) || seconds < 0) {
        return "0:00";
    }

    const total = Math.floor(seconds);
    const mins = Math.floor(total / 60);
    const secs = total % 60;

    return `${mins}:${secs.toString().padStart(2, "0")}`;
}

export function AudioAttachment({ src, filename, className }: AudioAttachmentProps) {
    const audioRef = useRef<HTMLAudioElement>(null);
    const [playing, setPlaying] = useState(false);
    const [current, setCurrent] = useState(0);
    const [duration, setDuration] = useState(0);
    const [failed, setFailed] = useState(false);

    function togglePlay() {
        const audio = audioRef.current;
        if (!audio) {
            return;
        }

        if (audio.paused) {
            audio.play().catch(() => setFailed(true));
            return;
        }

        audio.pause();
    }

    function seek(e: React.ChangeEvent<HTMLInputElement>) {
        const audio = audioRef.current;
        const next = Number(e.target.value);
        setCurrent(next);

        if (audio) {
            audio.currentTime = next;
        }
    }

    const progress = duration > 0 ? (current / duration) * 100 : 0;

    return (
        <div className={join(styles.wrapper, className)} onClick={e => e.stopPropagation()}>
            {filename && (
                <span className={styles.filename} title={filename}>
                    {filename}
                </span>
            )}
            <div className={styles.player}>
                <button
                    type="button"
                    className={styles.playButton}
                    onClick={togglePlay}
                    disabled={failed}
                    aria-label={playing ? "Pause" : "Play"}
                >
                    {playing ? "‖" : "▶"}
                </button>
                <input
                    type="range"
                    className={styles.scrubber}
                    style={{ "--progress": `${progress}%` } as React.CSSProperties}
                    min={0}
                    max={duration || 0}
                    step={0.1}
                    value={current}
                    onChange={seek}
                    disabled={failed || duration === 0}
                    aria-label="Seek"
                />
                <span className={styles.time}>
                    {failed ? "unavailable" : `${formatTime(current)} / ${formatTime(duration)}`}
                </span>
            </div>
            <audio
                ref={audioRef}
                src={src}
                preload="metadata"
                onLoadedMetadata={e => setDuration(e.currentTarget.duration)}
                onTimeUpdate={e => setCurrent(e.currentTarget.currentTime)}
                onPlay={() => setPlaying(true)}
                onPause={() => setPlaying(false)}
                onEnded={() => setPlaying(false)}
                onError={() => setFailed(true)}
            />
        </div>
    );
}

export function AudioThumb({ className }: AudioThumbProps) {
    return (
        <span className={join(styles.thumb, className)} role="img" aria-label="Audio file">
            &#9835;
        </span>
    );
}
