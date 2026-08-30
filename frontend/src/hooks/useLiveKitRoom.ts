import { useEffect, useRef, useState } from "react";
import type { Room } from "livekit-client";
import { connectRoom, disconnectRoom } from "../api/livekit/connect";
import { reportClientError } from "../api/telemetry";
import type { VoiceTokenResponse } from "../types/api";

export const ROOM_CONNECT_FAILED = "Could not connect to this stream.";

export interface UseLiveKitRoomOptions {
    streamId: string | undefined;
    isLive: boolean;
    wantsRoom: boolean;
    wantsMedia: boolean;
    requestToken: (streamId: string) => Promise<VoiceTokenResponse>;
}

export interface LiveKitRoomState {
    room: Room | null;
    error: string | null;
}

export function useLiveKitRoom(options: UseLiveKitRoomOptions): LiveKitRoomState {
    const { streamId, isLive, wantsRoom, wantsMedia, requestToken } = options;

    const [room, setRoom] = useState<Room | null>(null);
    const [error, setError] = useState<string | null>(null);
    const requestTokenRef = useRef(requestToken);

    useEffect(() => {
        requestTokenRef.current = requestToken;
    });

    useEffect(() => {
        if (!streamId || !isLive || !wantsRoom) {
            return;
        }

        const controller = new AbortController();
        const { signal } = controller;
        let connected: Room | null = null;

        requestTokenRef
            .current(streamId)
            .then(({ token, url }) =>
                connectRoom({
                    url,
                    token,
                    autoSubscribe: wantsMedia,
                    signal,
                    on: {
                        onConnected: joined => {
                            connected = joined;
                            if (!signal.aborted) {
                                setRoom(joined);
                            }
                        },
                        onDisconnected: left => {
                            setRoom(previous => (previous === left ? null : previous));
                        },
                    },
                }),
            )
            .catch((thrown: unknown) => {
                reportClientError(thrown, { source: "caught" });

                if (!signal.aborted) {
                    setError(ROOM_CONNECT_FAILED);
                }
            });

        return () => {
            controller.abort();
            disconnectRoom(connected);
        };
    }, [streamId, isLive, wantsRoom, wantsMedia]);

    return { room, error };
}
