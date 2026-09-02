export type StreamDefaultMode = "webrtc" | "hls";

export interface LiveStream {
    id: string;
    userId: string;
    title: string;
    status: string;
    viewerCount: number;
    thumbnailUrl?: string;
    startedAt?: string;
    streamerUsername: string;
    streamerDisplayName: string;
    streamerAvatarUrl: string;
    defaultMode: StreamDefaultMode;
    hlsUrl?: string;
}

export interface StreamOwner {
    stream: LiveStream;
    whipUrl: string;
    streamKey: string;
}

export interface StreamCredentials {
    whipUrl: string;
    streamKey: string;
    hlsEnabled: boolean;
}

export interface LiveStreamListResponse {
    streams: LiveStream[];
    enabled: boolean;
}

export interface OverlayConnection {
    token: string;
    connect_url: string;
    connected: boolean;
}
