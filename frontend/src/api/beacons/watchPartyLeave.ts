import { authHeaders } from "../client";
import { apiUrl } from "../origin";
import { reportClientError } from "../telemetry";

export function sendWatchPartyLeaveBeacon(roomId: string, sessionId: string): void {
    const url = apiUrl(`/api/v1/chat/rooms/${roomId}/watch-parties/${sessionId}/participants/me`);

    try {
        fetch(url, {
            method: "DELETE",
            credentials: "include",
            keepalive: true,
            headers: authHeaders(),
        }).catch((thrown: unknown) => {
            reportClientError(thrown, { source: "caught" });
        });
    } catch (thrown) {
        reportClientError(thrown, { source: "caught" });
    }
}
