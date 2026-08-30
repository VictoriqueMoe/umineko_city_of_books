import { useMutation } from "@tanstack/react-query";
import { getVoiceToken } from "../../api/endpoints/chat";

const DROP_SECRET_ON_SETTLE = {
    gcTime: 0,
} as const;

export function useVoiceToken() {
    return useMutation({
        mutationFn: (roomId: string) => getVoiceToken(roomId),
        ...DROP_SECRET_ON_SETTLE,
    });
}
