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
        cornerCounts: () => ["post", "corner-counts"] as const,
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
    oc: {
        all: ["oc"] as const,
        detail: (id: string) => ["oc", "detail", id] as const,
        feed: (params: Record<string, unknown> = {}) => ["oc", "feed", params] as const,
        userList: (userId: string) => ["oc", "userList", userId] as const,
        userSummaries: (userId: string) => ["oc", "userSummaries", userId] as const,
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
        list: (params: Record<string, unknown> = {}) => ["mystery", "list", params] as const,
        leaderboard: (limit: number | null = null) => ["mystery", "leaderboard", limit] as const,
        gmLeaderboard: (limit: number | null = null) => ["mystery", "gm-leaderboard", limit] as const,
    },
    gameRoom: {
        all: ["gameRoom"] as const,
        detail: (id: string) => ["gameRoom", "detail", id] as const,
        list: (filters: Record<string, unknown> = {}) => ["gameRoom", "list", filters] as const,
        live: (gameType?: string) =>
            gameType ? (["gameRoom", "live", gameType] as const) : (["gameRoom", "live"] as const),
        finished: (gameType: string, page: Record<string, unknown>) =>
            ["gameRoom", "finished", gameType, page] as const,
        scoreboard: (gameType: string) => ["gameRoom", "scoreboard", gameType] as const,
    },
    chat: {
        room: (id: string) => ["chat", "room", id] as const,
        roomMembers: (id: string) => ["chat", "room", id, "members"] as const,
        pinned: (id: string) => ["chat", "room", id, "pinned"] as const,
        userRooms: () => ["chat", "rooms", "user"] as const,
    },
    profile: {
        all: ["profile"] as const,
        byUsername: (username: string) => ["profile", "username", username] as const,
        blockedUsers: (userID: string) => ["profile", id(userID), "blocked"] as const,
    },
    notifications: {
        all: ["notifications"] as const,
        list: (params: Record<string, unknown> = {}) => ["notifications", "list", params] as const,
        unreadCount: () => ["notifications", "unread-count"] as const,
    },
    chatbots: {
        all: ["chatbots"] as const,
        list: () => ["chatbots", "list"] as const,
    },
    admin: {
        announcements: () => ["admin", "announcements"] as const,
        users: (params: Record<string, unknown> = {}) => ["admin", "users", params] as const,
        invites: () => ["admin", "invites"] as const,
        reports: (params: Record<string, unknown> = {}) => ["admin", "reports", params] as const,
        auditLog: (params: Record<string, unknown> = {}) => ["admin", "audit-log", params] as const,
        bannedGifs: () => ["admin", "banned-gifs"] as const,
        bannedWords: (scope: string) => ["admin", "banned-words", scope] as const,
        vanityRoles: () => ["admin", "vanity-roles"] as const,
        permissions: () => ["admin", "permissions"] as const,
        chatbots: () => ["admin", "chatbots"] as const,
        chatbotUsage: (days: number) => ["admin", "chatbots", "usage", days] as const,
        chatbotModels: () => ["admin", "chatbots", "models"] as const,
    },
} as const;

function id(value: string): string {
    return value;
}
