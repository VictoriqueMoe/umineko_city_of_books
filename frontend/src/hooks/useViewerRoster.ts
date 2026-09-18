import { useMemo } from "react";
import { absolutizeMedia } from "../api/origin";
import { summariseViewers, type ViewerParticipant, type ViewerRoster } from "../domain/live/viewers";

export function useViewerRoster(participants: readonly ViewerParticipant[]): ViewerRoster {
    return useMemo(() => summariseViewers(participants, absolutizeMedia), [participants]);
}
