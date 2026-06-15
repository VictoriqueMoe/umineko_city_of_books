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

        if (video.canPlayType("application/vnd.apple.mpegurl")) {
            video.src = src;
            return;
        }

        if (Hls.isSupported()) {
            const hls = new Hls({ lowLatencyMode: true, backBufferLength: 30 });
            hls.loadSource(src);
            hls.attachMedia(video);
            return () => {
                hls.destroy();
            };
        }

        video.src = src;
    }, [src]);

    return <video ref={videoRef} className={className} autoPlay playsInline controls />;
}
