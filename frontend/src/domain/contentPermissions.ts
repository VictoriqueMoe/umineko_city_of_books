import { can, type Permission, type PermissionSubject } from "./permissions";

export type ContentFamily =
    | "post"
    | "fanfic"
    | "art"
    | "ship"
    | "theory"
    | "mystery"
    | "mystery_attempt"
    | "journal"
    | "journal_entry"
    | "comment"
    | "response"
    | "oc"
    | "gallery";

export type ContentActionRule = Permission | "owner_only" | "no_surface";

export interface ContentFamilyRules {
    edit: ContentActionRule;
    delete: ContentActionRule;
}

export interface ContentViewer extends PermissionSubject {
    id?: string | null;
}

export interface ContentSubject {
    family: ContentFamily;
    authorId: string | null | undefined;
}

export interface ContentPermissions {
    canEdit: boolean;
    canDelete: boolean;
}

export const CONTENT_RULES: Record<ContentFamily, ContentFamilyRules> = {
    post: { edit: "edit_any_post", delete: "delete_any_post" },
    fanfic: { edit: "edit_any_post", delete: "delete_any_post" },
    art: { edit: "edit_any_post", delete: "delete_any_post" },
    ship: { edit: "edit_any_post", delete: "delete_any_post" },
    theory: { edit: "edit_any_theory", delete: "delete_any_theory" },
    mystery: { edit: "edit_any_theory", delete: "delete_any_theory" },
    mystery_attempt: { edit: "no_surface", delete: "delete_any_comment" },
    journal: { edit: "edit_any_journal", delete: "delete_any_journal" },
    journal_entry: { edit: "edit_any_journal", delete: "delete_any_journal" },
    comment: { edit: "edit_any_comment", delete: "delete_any_comment" },
    response: { edit: "no_surface", delete: "delete_any_response" },
    oc: { edit: "owner_only", delete: "no_surface" },
    gallery: { edit: "owner_only", delete: "owner_only" },
};

function allows(rule: ContentActionRule, viewer: ContentViewer | null | undefined, isOwner: boolean): boolean {
    if (rule === "no_surface") {
        return false;
    }

    if (rule === "owner_only") {
        return isOwner;
    }

    return isOwner || can(viewer, rule);
}

export function isContentOwner(viewer: ContentViewer | null | undefined, item: ContentSubject): boolean {
    return !!viewer?.id && viewer.id === item.authorId;
}

export function contentPermissions(viewer: ContentViewer | null | undefined, item: ContentSubject): ContentPermissions {
    const rules = CONTENT_RULES[item.family];
    const isOwner = isContentOwner(viewer, item);

    return {
        canEdit: allows(rules.edit, viewer, isOwner),
        canDelete: allows(rules.delete, viewer, isOwner),
    };
}
