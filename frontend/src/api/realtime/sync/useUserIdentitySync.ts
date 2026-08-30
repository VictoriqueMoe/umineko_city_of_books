import { useQueryClient } from "@tanstack/react-query";
import { userIdentityPatch } from "../../../domain/user/identityPatch";
import type { UserProfile } from "../../../types/api";
import { patchUserInCache } from "../../cache/patchUser";
import { queryKeys } from "../../queryKeys";
import { REALTIME_EVENTS } from "../events";
import { useRealtimeEvent } from "../useRealtime";

const USER_IDENTITY_EVENTS = [
    REALTIME_EVENTS.ROLE_CHANGED,
    REALTIME_EVENTS.BAN_CHANGED,
    REALTIME_EVENTS.LOCK_CHANGED,
    REALTIME_EVENTS.PROFILE_CHANGED,
] as const;

export function useUserIdentitySync(): void {
    const qc = useQueryClient();

    useRealtimeEvent(USER_IDENTITY_EVENTS, event => {
        const update = userIdentityPatch(event);
        if (!update) {
            return;
        }

        const me = qc.getQueryData<UserProfile | null>(queryKeys.auth.me());

        patchUserInCache(qc, update.userId, update.patch);

        if (me && me.id === update.userId) {
            qc.setQueryData<UserProfile | null>(queryKeys.auth.me(), { ...me, ...update.patch });
        }
    });
}
