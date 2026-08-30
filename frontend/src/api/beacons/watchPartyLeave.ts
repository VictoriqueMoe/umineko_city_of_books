import { apiUrl, authHeaders } from "../client";
import { nonFatal } from "../../utils/nonFatal";

export function sendWatchPartyLeaveBeacon(roomId: string, sessionId: string): void {
    const url = apiUrl(`/api/v1/chat/rooms/${roomId}/watch-parties/${sessionId}/participants/me`);

    try {
        fetch(url, {
            method: "DELETE",
            credentials: "include",
            keepalive: true,
            headers: authHeaders(),
        }).catch(nonFatal);
    } catch {
        nonFatal();
    }
}
