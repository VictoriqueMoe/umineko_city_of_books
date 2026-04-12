import type { JournalWork } from "../types/api";

interface WorkDef {
    id: JournalWork;
    label: string;
}

export const JOURNAL_WORKS: WorkDef[] = [
    { id: "general", label: "General" },
    { id: "umineko", label: "Umineko" },
    { id: "higurashi", label: "Higurashi" },
    { id: "ciconia", label: "Ciconia" },
    { id: "higanbana", label: "Higanbana" },
    { id: "roseguns", label: "Rose Guns Days" },
];

export function workLabel(work: JournalWork): string {
    const found = JOURNAL_WORKS.find(w => w.id === work);
    return found?.label ?? "General";
}
