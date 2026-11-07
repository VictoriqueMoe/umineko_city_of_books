import { useParams } from "react-router";
import { type ContentFamily, contentPermissions } from "../domain/contentPermissions";
import { useAuth } from "./useAuth";
import { useFanfic } from "./queries/fanfic";
import { useJournal } from "./queries/journal";
import { useMystery } from "./queries/mystery";
import { useOC } from "./queries/oc";
import { useTheory } from "./queries/theory";

export type EditableResource = Extract<ContentFamily, "mystery" | "theory" | "oc" | "fanfic" | "journal">;

export type Editable = {
    loading: boolean;
    allowed: boolean;
};

export function useEditable(resource: EditableResource): Editable {
    const { id = "" } = useParams();
    const { user } = useAuth();
    const mystery = useMystery(resource === "mystery" ? id : "");
    const theory = useTheory(resource === "theory" ? id : "");
    const oc = useOC(resource === "oc" ? id : "");
    const fanfic = useFanfic(resource === "fanfic" ? id : "");
    const journal = useJournal(resource === "journal" ? id : "");

    const sources: Record<EditableResource, { loading: boolean; authorId: string | undefined }> = {
        mystery: { loading: mystery.loading, authorId: mystery.mystery?.author.id },
        theory: { loading: theory.loading, authorId: theory.theory?.author.id },
        oc: { loading: oc.loading, authorId: oc.oc?.author.id },
        fanfic: { loading: fanfic.loading, authorId: fanfic.fanfic?.author.id },
        journal: { loading: journal.loading, authorId: journal.journal?.author.id },
    };
    const { loading, authorId } = sources[resource];

    return {
        loading,
        allowed: !!authorId && contentPermissions(user, { family: resource, authorId }).canEdit,
    };
}
