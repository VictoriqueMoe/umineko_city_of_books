import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { fetchOverlayConnectorSEF, resetOverlayToken, testOverlay } from "../../api/endpoints/overlay";
import { queryKeys } from "../../api/queryKeys";
import type { OverlayConnection } from "../../types/api";

const DROP_SECRET_ON_SETTLE = {
    gcTime: 0,
} as const;

export const OVERLAY_CONNECTOR_FILENAME = "overlay-connector.sef";

function markConnected(qc: QueryClient, connected: boolean): void {
    qc.setQueryData<OverlayConnection>(queryKeys.overlay.connection(), prev => (prev ? { ...prev, connected } : prev));
}

export function useResetOverlayToken() {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: () => resetOverlayToken(),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.overlay.connection() });
        },
        ...DROP_SECRET_ON_SETTLE,
    });
}

export function useTestOverlay() {
    const qc = useQueryClient();

    return useMutation({
        mutationFn: () => testOverlay(),
        onSuccess: () => markConnected(qc, true),
        onError: () => markConnected(qc, false),
    });
}

export function useOverlayConnectorFile() {
    return useMutation({
        mutationFn: async () => {
            const sef = await fetchOverlayConnectorSEF();

            return new Blob([sef], { type: "text/plain" });
        },
        ...DROP_SECRET_ON_SETTLE,
    });
}
