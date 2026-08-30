import { useMemo } from "react";
import { resolveViewerMeta } from "../api/livekit/viewerMeta";
import { summariseViewers, type ViewerParticipant, type ViewerRoster } from "../domain/live/viewers";

export function useViewerRoster(participants: readonly ViewerParticipant[]): ViewerRoster {
    return useMemo(() => summariseViewers(participants, resolveViewerMeta), [participants]);
}
