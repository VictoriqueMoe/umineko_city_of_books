import type { Notification, NotificationType } from "../types/api";

export type NotificationCategory =
    | "game_board"
    | "gallery"
    | "theories"
    | "mysteries_gm"
    | "mysteries_player"
    | "social"
    | "site_improvements"
    | "moderation";

interface NotificationConfig {
    text: string;
    category: NotificationCategory | "dynamic";
    route: (notif: Notification) => string;
}

const roleDisplayNames: Record<string, string> = {
    super_admin: "Reality Author",
    admin: "Voyager Witch",
    moderator: "Witch",
};

const categoryLabels: Record<NotificationCategory, string> = {
    game_board: "Game Board",
    gallery: "Gallery",
    theories: "Theories",
    mysteries_gm: "Mysteries (as Game Master)",
    mysteries_player: "Mysteries (as Player)",
    social: "Social",
    site_improvements: "Site Improvements",
    moderation: "Moderation",
};

const categoryOrder: NotificationCategory[] = [
    "game_board",
    "gallery",
    "theories",
    "mysteries_gm",
    "mysteries_player",
    "social",
    "site_improvements",
    "moderation",
];

function routeByReferenceType(notif: Notification): string {
    const refType = notif.reference_type;
    if (refType === "chat") {
        return `/chat/${notif.reference_id}`;
    }
    if (refType.startsWith("post_comment:")) {
        const commentId = refType.split(":")[1];
        return `/game-board/${notif.reference_id}#comment-${commentId}`;
    }
    if (refType.startsWith("art_comment:")) {
        const commentId = refType.split(":")[1];
        return `/gallery/art/${notif.reference_id}#comment-${commentId}`;
    }
    if (refType === "post") {
        return `/game-board/${notif.reference_id}`;
    }
    if (refType === "art") {
        return `/gallery/art/${notif.reference_id}`;
    }
    if (refType === "mystery") {
        return `/mystery/${notif.reference_id}`;
    }
    if (refType.startsWith("mystery_attempt:")) {
        const attemptId = refType.split(":")[1];
        return `/mystery/${notif.reference_id}#attempt-${attemptId}`;
    }
    if (refType.startsWith("mystery_comment:")) {
        const commentId = refType.split(":")[1];
        return `/mystery/${notif.reference_id}#comment-${commentId}`;
    }
    if (refType === "fanfic") {
        return `/fanfiction/${notif.reference_id}`;
    }
    if (refType.startsWith("fanfic_comment:")) {
        const commentId = refType.split(":")[1];
        return `/fanfiction/${notif.reference_id}#comment-${commentId}`;
    }
    if (refType === "ship" || refType.startsWith("ship_comment:")) {
        const parts = refType.split(":");
        if (parts.length === 2) {
            return `/ships/${notif.reference_id}#comment-${parts[1]}`;
        }
        return `/ships/${notif.reference_id}`;
    }
    if (refType === "announcement" || refType.startsWith("announcement_comment:")) {
        const parts = refType.split(":");
        if (parts.length === 2) {
            return `/announcements/${notif.reference_id}#comment-${parts[1]}`;
        }
        return `/announcements/${notif.reference_id}`;
    }
    return `/theory/${notif.reference_id}`;
}

function categoryFromReferenceType(notif: Notification): NotificationCategory {
    const refType = notif.reference_type;
    if (refType === "post" || refType.startsWith("post_comment:")) {
        return "game_board";
    }
    if (refType === "art" || refType.startsWith("art_comment:")) {
        return "gallery";
    }
    if (refType === "theory" || refType === "response") {
        return "theories";
    }
    if (refType === "mystery") {
        return "game_board";
    }
    if (refType === "ship" || refType.startsWith("ship_comment:")) {
        return "social";
    }
    return "social";
}

const notificationConfigs: Record<NotificationType, NotificationConfig> = {
    theory_response: {
        text: "responded to your theory",
        category: "theories",
        route: routeByReferenceType,
    },
    response_reply: {
        text: "replied to your response",
        category: "theories",
        route: routeByReferenceType,
    },
    theory_upvote: {
        text: "upvoted your theory",
        category: "theories",
        route: routeByReferenceType,
    },
    response_upvote: {
        text: "upvoted your response",
        category: "theories",
        route: routeByReferenceType,
    },
    chat_message: {
        text: "sent you a message",
        category: "social",
        route: routeByReferenceType,
    },
    report: {
        text: "reported content",
        category: "moderation",
        route: () => "/admin/reports",
    },
    report_resolved: {
        text: "resolved your report",
        category: "moderation",
        route: routeByReferenceType,
    },
    new_follower: {
        text: "started following you",
        category: "social",
        route: notif => `/user/${notif.actor.username}`,
    },
    post_liked: {
        text: "liked your post",
        category: "game_board",
        route: routeByReferenceType,
    },
    post_commented: {
        text: "commented on your post",
        category: "game_board",
        route: routeByReferenceType,
    },
    post_comment_reply: {
        text: "replied to your comment",
        category: "game_board",
        route: routeByReferenceType,
    },
    mention: {
        text: "mentioned you",
        category: "dynamic",
        route: routeByReferenceType,
    },
    art_liked: {
        text: "liked your art",
        category: "gallery",
        route: routeByReferenceType,
    },
    art_commented: {
        text: "commented on your art",
        category: "gallery",
        route: routeByReferenceType,
    },
    art_comment_reply: {
        text: "replied to your comment",
        category: "gallery",
        route: routeByReferenceType,
    },
    comment_liked: {
        text: "liked your comment",
        category: "dynamic",
        route: routeByReferenceType,
    },
    content_edited: {
        text: "edited your content",
        category: "dynamic",
        route: routeByReferenceType,
    },
    mystery_attempt: {
        text: "made an attempt on your mystery",
        category: "mysteries_gm",
        route: routeByReferenceType,
    },
    mystery_reply: {
        text: "replied in a thread on your mystery",
        category: "mysteries_gm",
        route: routeByReferenceType,
    },
    mystery_attempt_vote: {
        text: "voted on your attempt",
        category: "mysteries_player",
        route: routeByReferenceType,
    },
    mystery_solved: {
        text: "chose your attempt as the winner!",
        category: "mysteries_player",
        route: routeByReferenceType,
    },
    mystery_comment_reply: {
        text: "replied to your comment on a mystery",
        category: "mysteries_player",
        route: routeByReferenceType,
    },
    fanfic_commented: {
        text: "commented on your fanfic",
        category: "social",
        route: routeByReferenceType,
    },
    fanfic_comment_reply: {
        text: "replied to your comment on a fanfic",
        category: "social",
        route: routeByReferenceType,
    },
    fanfic_comment_liked: {
        text: "liked your comment on a fanfic",
        category: "social",
        route: routeByReferenceType,
    },
    fanfic_favourited: {
        text: "favourited your fanfic",
        category: "social",
        route: routeByReferenceType,
    },
    ship_commented: {
        text: "commented on your ship",
        category: "social",
        route: routeByReferenceType,
    },
    ship_comment_reply: {
        text: "replied to your comment",
        category: "social",
        route: routeByReferenceType,
    },
    ship_comment_liked: {
        text: "liked your comment",
        category: "social",
        route: routeByReferenceType,
    },
    announcement_commented: {
        text: "commented on your announcement",
        category: "moderation",
        route: routeByReferenceType,
    },
    announcement_comment_reply: {
        text: "replied to your comment",
        category: "moderation",
        route: routeByReferenceType,
    },
    announcement_comment_liked: {
        text: "liked your comment",
        category: "moderation",
        route: routeByReferenceType,
    },
    suggestion_posted: {
        text: "posted a site suggestion",
        category: "site_improvements",
        route: notif => `/suggestions/${notif.reference_id}`,
    },
    suggestion_resolved: {
        text: "marked your suggestion as done",
        category: "site_improvements",
        route: notif => `/suggestions/${notif.reference_id}`,
    },
    content_shared: {
        text: "shared your content",
        category: "social",
        route: routeByReferenceType,
    },
};

export function getNotificationText(notif: Notification): string {
    if (notif.message && notif.type !== "content_edited") {
        return notif.message;
    }
    return notificationConfigs[notif.type]?.text ?? "";
}

export function getNotificationRoute(notif: Notification): string {
    const config = notificationConfigs[notif.type];
    if (config) {
        return config.route(notif);
    }
    return `/theory/${notif.reference_id}`;
}

export function getNotificationCategory(notif: Notification): NotificationCategory {
    const config = notificationConfigs[notif.type];
    if (!config) {
        return "social";
    }
    if (config.category === "dynamic") {
        return categoryFromReferenceType(notif);
    }
    return config.category;
}

export function getCategoryLabel(category: NotificationCategory): string {
    return categoryLabels[category];
}

export function getCategoryOrder(): NotificationCategory[] {
    return categoryOrder;
}

export function groupByCategory(notifications: Notification[]): Map<NotificationCategory, Notification[]> {
    const groups = new Map<NotificationCategory, Notification[]>();
    for (const notif of notifications) {
        const cat = getNotificationCategory(notif);
        const list = groups.get(cat);
        if (list) {
            list.push(notif);
        } else {
            groups.set(cat, [notif]);
        }
    }
    return groups;
}

export function isContentEditedNotification(notif: Notification): boolean {
    return notif.type === "content_edited";
}

export function formatContentEditedText(notif: Notification): { message: string; role: string; actorName: string } {
    const message = notif.message || "your content has been edited";
    const role = notif.actor.role ? (roleDisplayNames[notif.actor.role] ?? "") : "";
    return { message, role, actorName: notif.actor.display_name };
}

export function relativeTime(dateStr: string): string {
    const now = Date.now();
    const then = new Date(dateStr).getTime();
    const diffSeconds = Math.floor((now - then) / 1000);

    if (diffSeconds < 60) {
        return "just now";
    }

    const diffMinutes = Math.floor(diffSeconds / 60);
    if (diffMinutes < 60) {
        return `${diffMinutes}m ago`;
    }

    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) {
        return `${diffHours}h ago`;
    }

    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 30) {
        return `${diffDays}d ago`;
    }

    const diffMonths = Math.floor(diffDays / 30);
    return `${diffMonths}mo ago`;
}
