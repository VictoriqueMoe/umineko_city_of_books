import type { SearchEntityType } from "../../../types/api";

export type SearchTypeGroup =
    | "theories"
    | "posts"
    | "art"
    | "mysteries"
    | "ships"
    | "ocs"
    | "announcements"
    | "fanfics"
    | "journals"
    | "users";

export interface SearchTypeMeta {
    type: SearchEntityType;
    label: string;
    short: string;
    color: string;
    group: SearchTypeGroup;
}

interface SearchTypeGroupDef {
    id: SearchTypeGroup;
    label: string;
}

const SEARCH_REGISTRY: SearchTypeMeta[] = [
    { type: "theory", label: "Theory", short: "Theory", color: "#a78bfa", group: "theories" },
    { type: "response", label: "Theory reply", short: "Reply", color: "#a78bfa", group: "theories" },
    { type: "post", label: "Game Board post", short: "Post", color: "#38bdf8", group: "posts" },
    { type: "post_comment", label: "Game Board comment", short: "Comment", color: "#38bdf8", group: "posts" },
    { type: "art", label: "Artwork", short: "Art", color: "#f472b6", group: "art" },
    { type: "art_comment", label: "Art comment", short: "Comment", color: "#f472b6", group: "art" },
    { type: "mystery", label: "Mystery", short: "Mystery", color: "#fb923c", group: "mysteries" },
    { type: "mystery_attempt", label: "Mystery solution", short: "Solution", color: "#fb923c", group: "mysteries" },
    { type: "mystery_comment", label: "Mystery comment", short: "Comment", color: "#fb923c", group: "mysteries" },
    { type: "ship", label: "Ship", short: "Ship", color: "#fb7185", group: "ships" },
    { type: "ship_comment", label: "Ship comment", short: "Comment", color: "#fb7185", group: "ships" },
    { type: "oc", label: "OC", short: "OC", color: "#c084fc", group: "ocs" },
    { type: "oc_comment", label: "OC comment", short: "Comment", color: "#c084fc", group: "ocs" },
    { type: "announcement", label: "Announcement", short: "News", color: "#facc15", group: "announcements" },
    {
        type: "announcement_comment",
        label: "Announcement comment",
        short: "Comment",
        color: "#facc15",
        group: "announcements",
    },
    { type: "fanfic", label: "Fanfiction", short: "Fanfic", color: "#34d399", group: "fanfics" },
    { type: "fanfic_comment", label: "Fanfic comment", short: "Comment", color: "#34d399", group: "fanfics" },
    { type: "journal", label: "Journal", short: "Journal", color: "#60a5fa", group: "journals" },
    { type: "journal_comment", label: "Journal comment", short: "Comment", color: "#60a5fa", group: "journals" },
    { type: "user", label: "User", short: "User", color: "#e89ec0", group: "users" },
];

const SEARCH_GROUP_DEFS: SearchTypeGroupDef[] = [
    { id: "theories", label: "Theories" },
    { id: "posts", label: "Game Board" },
    { id: "art", label: "Art" },
    { id: "mysteries", label: "Mysteries" },
    { id: "ships", label: "Ships" },
    { id: "ocs", label: "OCs" },
    { id: "fanfics", label: "Fanfiction" },
    { id: "journals", label: "Journals" },
    { id: "announcements", label: "Announcements" },
    { id: "users", label: "Users" },
];

export const SEARCH_TYPE_META: Record<SearchEntityType, SearchTypeMeta> = Object.fromEntries(
    SEARCH_REGISTRY.map(entry => [entry.type, entry]),
) as Record<SearchEntityType, SearchTypeMeta>;

export const SEARCH_GROUP_LABEL: Record<SearchTypeGroup, string> = Object.fromEntries(
    SEARCH_GROUP_DEFS.map(g => [g.id, g.label]),
) as Record<SearchTypeGroup, string>;

export const SEARCH_GROUP_ORDER: SearchTypeGroup[] = SEARCH_GROUP_DEFS.map(g => g.id);

export interface SearchFilterOption {
    value: string;
    label: string;
}

export const SEARCH_FILTER_OPTIONS: SearchFilterOption[] = [
    { value: "", label: "All" },
    ...SEARCH_GROUP_DEFS.map(group => ({
        value: SEARCH_REGISTRY.filter(r => r.group === group.id)
            .map(r => r.type)
            .join(","),
        label: group.label,
    })),
    { value: "comments", label: "Comments only" },
];
