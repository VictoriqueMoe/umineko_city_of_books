import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createResponse,
    createTheory,
    deleteResponse,
    deleteTheory,
    refuteTheory,
    updateTheory,
    voteResponse,
    voteTheory,
} from "../../api/endpoints/theory";
import type { CreateResponsePayload, CreateTheoryPayload } from "../../types/api";
import { invalidateViews } from "../../api/cache/invalidate";

type UpdateTheoryPayload = CreateTheoryPayload;

export function useCreateTheory() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateTheoryPayload) => createTheory(payload),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useUpdateTheory(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: UpdateTheoryPayload) => updateTheory(id, payload),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useDeleteTheory() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteTheory(id),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useVoteTheory(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (value: number) => voteTheory(id, value),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useRefuteTheory(theoryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (responseId: string) => refuteTheory(theoryId, responseId),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useCreateResponse(theoryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateResponsePayload) => createResponse(theoryId, payload),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useDeleteResponse() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (responseId: string) => deleteResponse(responseId),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}

export function useVoteResponse() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ responseId, value }: { responseId: string; value: number }) => voteResponse(responseId, value),
        onSuccess: () => invalidateViews(qc, "theory"),
    });
}
