import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createResponse,
    createTheory,
    deleteResponse,
    deleteTheory,
    updateTheory,
    voteResponse,
    voteTheory,
} from "../endpoints";
import type { CreateResponsePayload, CreateTheoryPayload } from "../../types/api";

type UpdateTheoryPayload = CreateTheoryPayload;
import { queryKeys } from "../queryKeys";

export function useCreateTheory() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateTheoryPayload) => createTheory(payload),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.all });
        },
    });
}

export function useUpdateTheory(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: UpdateTheoryPayload) => updateTheory(id, payload),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.detail(id) });
            void qc.invalidateQueries({ queryKey: queryKeys.theory.all });
        },
    });
}

export function useDeleteTheory() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteTheory(id),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.all });
        },
    });
}

export function useVoteTheory(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (value: number) => voteTheory(id, value),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.detail(id) });
        },
    });
}

export function useCreateResponse(theoryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateResponsePayload) => createResponse(theoryId, payload),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.detail(theoryId) });
        },
    });
}

export function useDeleteResponse(theoryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (responseId: string) => deleteResponse(responseId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.detail(theoryId) });
        },
    });
}

export function useVoteResponse(theoryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ responseId, value }: { responseId: string; value: number }) => voteResponse(responseId, value),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.theory.detail(theoryId) });
        },
    });
}
