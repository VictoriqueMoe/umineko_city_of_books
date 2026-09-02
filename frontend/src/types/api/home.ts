export interface HomeActivityAuthor {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
}

export interface HomeActivityEntry {
    kind: "theory" | "post" | "journal" | "art";
    id: string;
    title: string;
    excerpt: string;
    corner: string;
    url: string;
    created_at: string;
    author: HomeActivityAuthor;
}

export interface HomeEcho {
    kind: "theory" | "post" | "journal" | "art";
    id: string;
    title: string;
    excerpt: string;
    corner: string;
    episode: number;
    is_spoiler: boolean;
    url: string;
    age: string;
    created_at: string;
    author: HomeActivityAuthor;
}

export interface HomeMember {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    created_at: string;
}

export interface HomePublicRoom {
    id: string;
    name: string;
    description: string;
    member_count: number;
    last_message_at: string | null;
}

export interface HomeCornerActivity {
    corner: string;
    post_count: number;
    unique_posters: number;
    last_post_at: string | null;
}

export interface HomeActivityResponse {
    online_count: number;
    recent_activity: HomeActivityEntry[];
    echoes: HomeEcho[];
    recent_members: HomeMember[];
    public_rooms: HomePublicRoom[];
    corner_activity: HomeCornerActivity[];
}

export interface SidebarActivityResponse {
    activity: Record<string, string>;
}

export interface SidebarLastVisitedResponse {
    visited: Record<string, string>;
}

export interface MarkSidebarVisitedRequest {
    key: string;
}
