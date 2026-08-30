import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createReport, resolveReport } from "../../api/endpoints/report";
import { queryKeys } from "../../api/queryKeys";

export function useResolveReport() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, comment }: { id: number; comment: string }) => resolveReport(id, comment),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.reportsAll() }),
    });
}

export function useCreateReport() {
    return useMutation({
        mutationFn: ({
            targetType,
            targetId,
            reason,
            contextId,
        }: {
            targetType: string;
            targetId: string;
            reason: string;
            contextId?: string;
        }) => createReport(targetType, targetId, reason, contextId),
    });
}
