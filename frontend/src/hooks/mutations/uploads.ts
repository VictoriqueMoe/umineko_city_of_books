import { useMutation } from "@tanstack/react-query";
import type { UploadProgress } from "../../api/client";
import { beginUpload, getUploadStatuses, removeUpload } from "../../api/endpoints/uploads";
import type { UploadKind } from "../../types/api";

type BeginUploadInput = {
    file: File;
    onProgress: UploadProgress;
    signal: AbortSignal;
};

export function useBeginUpload(kind: UploadKind) {
    return useMutation({
        mutationFn: ({ file, onProgress, signal }: BeginUploadInput) => beginUpload(kind, file, onProgress, signal),
    });
}

export function useUploadStatuses() {
    return useMutation({
        mutationFn: getUploadStatuses,
    });
}

export function useRemoveUpload() {
    return useMutation({
        mutationFn: removeUpload,
    });
}
