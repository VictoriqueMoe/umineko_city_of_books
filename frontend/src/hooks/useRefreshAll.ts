import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

export function useRefreshAll(): () => Promise<void> {
    const qc = useQueryClient();

    return useCallback(async () => {
        await qc.refetchQueries({ type: "active" });
    }, [qc]);
}
