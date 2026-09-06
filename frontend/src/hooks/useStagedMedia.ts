import { useCallback, useMemo, useState } from "react";

export interface StagedMedia {
    file: File;
    isSpoiler: boolean;
}

export interface UseStagedMediaResult {
    staged: StagedMedia[];
    files: File[];
    spoilers: boolean[];
    add: (files: File[]) => void;
    remove: (index: number) => void;
    reorder: (from: number, to: number) => void;
    toggleSpoiler: (index: number) => void;
    keepOnly: (files: File[]) => void;
    reset: () => void;
}

export function stageMedia(files: File[]): StagedMedia[] {
    return files.map(file => ({ file, isSpoiler: false }));
}

export function useStagedMedia(): UseStagedMediaResult {
    const [staged, setStaged] = useState<StagedMedia[]>([]);

    const files = useMemo(() => staged.map(s => s.file), [staged]);
    const spoilers = useMemo(() => staged.map(s => s.isSpoiler), [staged]);

    const add = useCallback((incoming: File[]) => {
        setStaged(prev => [...prev, ...stageMedia(incoming)]);
    }, []);

    const remove = useCallback((index: number) => {
        setStaged(prev => prev.filter((_, i) => i !== index));
    }, []);

    const reorder = useCallback((from: number, to: number) => {
        setStaged(prev => {
            const next = prev.slice();
            const [moved] = next.splice(from, 1);
            next.splice(to, 0, moved);

            return next;
        });
    }, []);

    const toggleSpoiler = useCallback((index: number) => {
        setStaged(prev => prev.map((s, i) => (i === index ? { ...s, isSpoiler: !s.isSpoiler } : s)));
    }, []);

    const keepOnly = useCallback((kept: File[]) => {
        setStaged(prev => prev.filter(s => kept.includes(s.file)));
    }, []);

    const reset = useCallback(() => {
        setStaged([]);
    }, []);

    return { staged, files, spoilers, add, remove, reorder, toggleSpoiler, keepOnly, reset };
}
