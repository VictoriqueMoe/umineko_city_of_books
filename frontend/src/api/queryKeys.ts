export const queryKeys = {
    theory: {
        all: ["theory"] as const,
        detail: (id: string) => ["theory", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["theory", "feed", params] as const,
    },
    post: {
        all: ["post"] as const,
        detail: (id: string) => ["post", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["post", "feed", params] as const,
    },
    art: {
        all: ["art"] as const,
        detail: (id: string) => ["art", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["art", "feed", params] as const,
    },
    ship: {
        all: ["ship"] as const,
        detail: (id: string) => ["ship", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["ship", "feed", params] as const,
    },
    journal: {
        all: ["journal"] as const,
        detail: (id: string) => ["journal", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["journal", "feed", params] as const,
    },
    fanfic: {
        all: ["fanfic"] as const,
        detail: (id: string) => ["fanfic", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["fanfic", "feed", params] as const,
    },
    mystery: {
        all: ["mystery"] as const,
        detail: (id: string) => ["mystery", "detail", id] as const,
    },
    gameRoom: {
        all: ["gameRoom"] as const,
        detail: (id: string) => ["gameRoom", "detail", id] as const,
        list: (filters: Record<string, unknown> = {}) => ["gameRoom", "list", filters] as const,
    },
    chat: {
        all: ["chat"] as const,
        room: (id: string) => ["chat", "room", id] as const,
        roomMembers: (id: string) => ["chat", "room", id, "members"] as const,
        roomMessages: (id: string) => ["chat", "room", id, "messages"] as const,
        roomList: (params: Record<string, unknown> = {}) => ["chat", "rooms", params] as const,
        systemRooms: () => ["chat", "system-rooms"] as const,
        pinned: (id: string) => ["chat", "room", id, "pinned"] as const,
    },
    profile: {
        all: ["profile"] as const,
        byUsername: (username: string) => ["profile", "username", username] as const,
        byID: (id: string) => ["profile", "id", id] as const,
        blockedUsers: (userID: string) => ["profile", id(userID), "blocked"] as const,
    },
    notifications: {
        all: ["notifications"] as const,
        list: (params: Record<string, unknown> = {}) => ["notifications", "list", params] as const,
        unreadCount: () => ["notifications", "unread-count"] as const,
    },
    admin: {
        all: ["admin"] as const,
        announcements: () => ["admin", "announcements"] as const,
        users: (params: Record<string, unknown> = {}) => ["admin", "users", params] as const,
        invites: () => ["admin", "invites"] as const,
        reports: (params: Record<string, unknown> = {}) => ["admin", "reports", params] as const,
        auditLog: (params: Record<string, unknown> = {}) => ["admin", "audit-log", params] as const,
        bannedGifs: () => ["admin", "banned-gifs"] as const,
        bannedWords: (scope: string) => ["admin", "banned-words", scope] as const,
        vanityRoles: () => ["admin", "vanity-roles"] as const,
    },
    quotes: {
        all: ["quotes"] as const,
        list: (params: Record<string, unknown> = {}) => ["quotes", "list", params] as const,
    },
    giphy: {
        favourites: () => ["giphy", "favourites"] as const,
        trending: () => ["giphy", "trending"] as const,
    },
    siteInfo: () => ["site-info"] as const,
    settings: () => ["settings"] as const,
    theme: () => ["theme"] as const,
} as const;

function id(value: string): string {
    return value;
}
