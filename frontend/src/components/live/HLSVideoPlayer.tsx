import { useEffect, useRef } from "react";
import Hls from "hls.js";

interface HLSVideoPlayerProps {
    src: string;
    className?: string;
}

export function HLSVideoPlayer({ src, className }: HLSVideoPlayerProps) {
    const videoRef = useRef<HTMLVideoElement>(null);

    useEffect(() => {
        const video = videoRef.current;
        if (!video || !src) {
            return;
        }

        const tryPlay = () => {
            video.play().catch(() => {
                video.muted = true;
                video.play().catch(() => {});
            });
        };

        if (Hls.isSupported()) {
            const hls = new Hls({ backBufferLength: 30 });
            let reloadTimer = 0;

            hls.on(Hls.Events.MANIFEST_PARSED, tryPlay);
            hls.on(Hls.Events.ERROR, (_event, data) => {
                if (!data.fatal) {
                    return;
                }

                if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
                    hls.recoverMediaError();
                    return;
                }

                window.clearTimeout(reloadTimer);
                reloadTimer = window.setTimeout(() => {
                    hls.loadSource(src);
                    hls.startLoad();
                }, 2000);
            });

            hls.loadSource(src);
            hls.attachMedia(video);

            return () => {
                window.clearTimeout(reloadTimer);
                hls.destroy();
            };
        }

        if (video.canPlayType("application/vnd.apple.mpegurl")) {
            video.src = src;
            video.addEventListener("loadedmetadata", tryPlay);
            return () => {
                video.removeEventListener("loadedmetadata", tryPlay);
            };
        }
    }, [src]);

    return <video ref={videoRef} className={className} autoPlay playsInline controls />;
}
