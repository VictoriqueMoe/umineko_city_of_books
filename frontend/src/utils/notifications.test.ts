import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Notification, NotificationType } from "../types/api";
import type { NotificationCategory } from "./notifications";
import {
    ensureNotificationPermission,
    formatContentEditedText,
    getCategoryLabel,
    getCategoryOrder,
    getNotificationRoute,
    getNotificationText,
    groupByCategory,
    isContentEditedNotification,
    relativeTime,
    showDesktopNotification,
} from "./notifications";

function makeNotification(overrides: Partial<Notification> = {}): Notification {
    return {
        id: 1,
        type: "theory_response",
        reference_id: "ref-1",
        reference_type: "theory",
        actor: {
            id: "00000000-0000-0000-0000-000000000001",
            username: "beatrice",
            display_name: "Beatrice",
        },
        read: false,
        created_at: "2026-01-01T00:00:00Z",
        count: 1,
        ...overrides,
    };
}

const UNKNOWN_TYPE = "some_future_type" as NotificationType;

const typeCases: { type: NotificationType; text: string; category: NotificationCategory | "dynamic" }[] = [
    { type: "theory_response", text: "responded to your theory", category: "theories" },
    { type: "response_reply", text: "replied to your response", category: "theories" },
    { type: "theory_upvote", text: "upvoted your theory", category: "theories" },
    { type: "response_upvote", text: "upvoted your response", category: "theories" },
    { type: "chat_message", text: "sent you a message", category: "social" },
    { type: "chat_room_message", text: "sent a message in a chat room", category: "social" },
    { type: "report", text: "reported content", category: "moderation" },
    { type: "report_resolved", text: "resolved your report", category: "moderation" },
    { type: "new_follower", text: "started following you", category: "social" },
    { type: "post_liked", text: "liked your post", category: "game_board" },
    { type: "post_commented", text: "commented on your post", category: "game_board" },
    { type: "post_comment_reply", text: "replied to your comment", category: "game_board" },
    { type: "mention", text: "mentioned you", category: "dynamic" },
    { type: "art_liked", text: "liked your art", category: "gallery" },
    { type: "art_commented", text: "commented on your art", category: "gallery" },
    { type: "art_comment_reply", text: "replied to your comment", category: "gallery" },
    { type: "comment_liked", text: "liked your comment", category: "dynamic" },
    { type: "content_edited", text: "edited your content", category: "dynamic" },
    { type: "mystery_attempt", text: "made an attempt on your mystery", category: "mysteries_gm" },
    { type: "mystery_reply", text: "replied in a thread on your mystery", category: "mysteries_gm" },
    { type: "mystery_attempt_vote", text: "voted on your attempt", category: "mysteries_player" },
    { type: "mystery_solved", text: "chose your attempt as the winner!", category: "mysteries_player" },
    { type: "mystery_paused_notif", text: "paused a mystery you are playing", category: "mysteries_player" },
    { type: "mystery_unpaused", text: "resumed a mystery you are playing", category: "mysteries_player" },
    {
        type: "mystery_gm_away_notif",
        text: "marked themselves as away on a mystery you are playing",
        category: "mysteries_player",
    },
    { type: "mystery_gm_back_notif", text: "is back on a mystery you are playing", category: "mysteries_player" },
    { type: "mystery_solved_all", text: "a mystery you were playing has been solved", category: "mysteries_player" },
    { type: "mystery_comment_reply", text: "replied to your comment on a mystery", category: "mysteries_player" },
    { type: "mystery_private_clue", text: "revealed a private red truth to you", category: "mysteries_player" },
    { type: "fanfic_commented", text: "commented on your fanfic", category: "social" },
    { type: "fanfic_comment_reply", text: "replied to your comment on a fanfic", category: "social" },
    { type: "fanfic_comment_liked", text: "liked your comment on a fanfic", category: "social" },
    { type: "fanfic_favourited", text: "favourited your fanfic", category: "social" },
    { type: "ship_commented", text: "commented on your ship", category: "social" },
    { type: "ship_comment_reply", text: "replied to your comment", category: "social" },
    { type: "ship_comment_liked", text: "liked your comment", category: "social" },
    { type: "oc_commented", text: "commented on your OC", category: "social" },
    { type: "oc_comment_reply", text: "replied to your comment", category: "social" },
    { type: "oc_comment_liked", text: "liked your comment", category: "social" },
    { type: "oc_favourited", text: "favourited your OC", category: "social" },
    { type: "announcement_commented", text: "commented on your announcement", category: "moderation" },
    { type: "announcement_comment_reply", text: "replied to your comment", category: "moderation" },
    { type: "announcement_comment_liked", text: "liked your comment", category: "moderation" },
    { type: "suggestion_posted", text: "posted a site suggestion", category: "site_improvements" },
    { type: "suggestion_resolved", text: "marked your suggestion as done", category: "site_improvements" },
    { type: "content_shared", text: "shared your content", category: "social" },
    { type: "journal_update", text: "posted a new update on a journal you follow", category: "social" },
    { type: "journal_commented", text: "commented on your journal", category: "social" },
    { type: "journal_comment_reply", text: "replied to your comment on a journal", category: "social" },
    { type: "journal_comment_liked", text: "liked your comment", category: "social" },
    { type: "journal_followed", text: "started following your journal", category: "social" },
    { type: "journal_archived", text: "your journal was archived after 7 days of inactivity", category: "social" },
    { type: "chat_mention", text: "mentioned you in a chat room", category: "social" },
    { type: "chat_room_invite", text: "added you to a chat room", category: "social" },
    { type: "chat_reply", text: "replied to your message", category: "social" },
    { type: "chat_room_banned", text: "banned you from a chat room", category: "moderation" },
    { type: "chat_room_kicked", text: "kicked you from a chat room", category: "moderation" },
    { type: "chat_room_unbanned", text: "unbanned you from a chat room", category: "moderation" },
    { type: "secret_comment_reply", text: "replied to your comment on a hunt", category: "social" },
    { type: "secret_commented", text: "commented on a hunt you're watching", category: "social" },
    { type: "secret_comment_liked", text: "liked your comment on a hunt", category: "social" },
    { type: "secret_solved_by_other", text: "solved a hunt before you could", category: "social" },
    { type: "game_invite", text: "invited you to a game", category: "social" },
    { type: "game_your_turn", text: "it's your move", category: "social" },
    { type: "game_finished", text: "your game has ended", category: "social" },
];

describe("getNotificationText", () => {
    for (const testCase of typeCases) {
        it(`describes a ${testCase.type} notification as "${testCase.text}"`, () => {
            // given
            const notif = makeNotification({ type: testCase.type });

            // when
            const text = getNotificationText(notif);

            // then
            expect(text).toBe(testCase.text);
        });
    }

    it("prefers the server supplied message over the built in wording", () => {
        // given
        const notif = makeNotification({ type: "post_liked", message: "liked your post about Beatrice" });

        // when
        const text = getNotificationText(notif);

        // then
        expect(text).toBe("liked your post about Beatrice");
    });

    it("falls back to the built in wording when the message is empty", () => {
        // given
        const notif = makeNotification({ type: "post_liked", message: "" });

        // when
        const text = getNotificationText(notif);

        // then
        expect(text).toBe("liked your post");
    });

    it("ignores the server message for content edited notifications", () => {
        // given
        const notif = makeNotification({ type: "content_edited", message: "your theory was tidied up" });

        // when
        const text = getNotificationText(notif);

        // then
        expect(text).toBe("edited your content");
    });

    it("returns an empty string for an unrecognised notification type", () => {
        // given
        const notif = makeNotification({ type: UNKNOWN_TYPE });

        // when
        const text = getNotificationText(notif);

        // then
        expect(text).toBe("");
    });

    it("still shows the server message for an unrecognised notification type", () => {
        // given
        const notif = makeNotification({ type: UNKNOWN_TYPE, message: "something new happened" });

        // when
        const text = getNotificationText(notif);

        // then
        expect(text).toBe("something new happened");
    });
});

describe("getNotificationRoute", () => {
    const referenceRoutes: { reference_type: string; expected: string }[] = [
        { reference_type: "chat", expected: "/chat/ref-1" },
        { reference_type: "post", expected: "/game-board/ref-1" },
        { reference_type: "post_comment:c9", expected: "/game-board/ref-1#comment-c9" },
        { reference_type: "art", expected: "/gallery/art/ref-1" },
        { reference_type: "art_comment:c9", expected: "/gallery/art/ref-1#comment-c9" },
        { reference_type: "mystery", expected: "/mystery/ref-1" },
        { reference_type: "mystery_attempt:a9", expected: "/mystery/ref-1#attempt-a9" },
        { reference_type: "mystery_comment:c9", expected: "/mystery/ref-1#comment-c9" },
        { reference_type: "fanfic", expected: "/fanfiction/ref-1" },
        { reference_type: "fanfic_comment:c9", expected: "/fanfiction/ref-1#comment-c9" },
        { reference_type: "ship", expected: "/ships/ref-1" },
        { reference_type: "ship_comment:c9", expected: "/ships/ref-1#comment-c9" },
        { reference_type: "oc", expected: "/oc/ref-1" },
        { reference_type: "oc_comment:c9", expected: "/oc/ref-1#comment-c9" },
        { reference_type: "announcement", expected: "/announcements/ref-1" },
        { reference_type: "announcement_comment:c9", expected: "/announcements/ref-1#comment-c9" },
        { reference_type: "journal", expected: "/journals/ref-1" },
        { reference_type: "journal_comment:c9", expected: "/journals/ref-1#comment-c9" },
        { reference_type: "journal_entry:e9", expected: "/journals/ref-1/entry/e9" },
        { reference_type: "journal_entry_comment:e9:c9", expected: "/journals/ref-1/entry/e9#comment-c9" },
        { reference_type: "secret:s9", expected: "/secrets/s9" },
        { reference_type: "secret_comment:s9:c9", expected: "/secrets/s9#comment-c9" },
        { reference_type: "theory", expected: "/theory/ref-1" },
        { reference_type: "response", expected: "/theory/ref-1" },
    ];

    for (const testCase of referenceRoutes) {
        it(`links a "${testCase.reference_type}" reference to ${testCase.expected}`, () => {
            // given
            const notif = makeNotification({ type: "mention", reference_type: testCase.reference_type });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe(testCase.expected);
        });
    }

    const malformedReferences: { reference_type: string; expected: string }[] = [
        { reference_type: "journal_entry_comment:e9", expected: "/journals/ref-1" },
        { reference_type: "journal_entry_comment:e9:", expected: "/journals/ref-1" },
        { reference_type: "journal_entry_comment::c9", expected: "/journals/ref-1" },
        { reference_type: "journal_entry:", expected: "/journals/ref-1" },
        { reference_type: "secret_comment:s9", expected: "/secrets" },
        { reference_type: "secret_comment:", expected: "/secrets" },
        { reference_type: "secret:s9:extra", expected: "/secrets" },
        { reference_type: "secret:", expected: "/secrets" },
    ];

    for (const testCase of malformedReferences) {
        it(`falls back to ${testCase.expected} for the malformed reference "${testCase.reference_type}"`, () => {
            // given
            const notif = makeNotification({ type: "mention", reference_type: testCase.reference_type });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe(testCase.expected);
        });
    }

    it("falls back to the theory route for an unrecognised reference type", () => {
        // given
        const unknownRef = makeNotification({ type: "mention", reference_type: "wardrobe" });
        const emptyRef = makeNotification({ type: "mention", reference_type: "" });

        // when
        const unknownRoute = getNotificationRoute(unknownRef);
        const emptyRoute = getNotificationRoute(emptyRef);

        // then
        expect(unknownRoute).toBe("/theory/ref-1");
        expect(emptyRoute).toBe("/theory/ref-1");
    });

    it("falls back to the theory route for an unrecognised notification type", () => {
        // given
        const notif = makeNotification({ type: UNKNOWN_TYPE, reference_id: "t9" });

        // when
        const route = getNotificationRoute(notif);

        // then
        expect(route).toBe("/theory/t9");
    });

    it("sends reports to the moderation queue rather than the reported content", () => {
        // given
        const notif = makeNotification({ type: "report", reference_type: "post", reference_id: "p1" });

        // when
        const route = getNotificationRoute(notif);

        // then
        expect(route).toBe("/admin/reports");
    });

    it("sends a new follower to the follower's profile", () => {
        // given
        const notif = makeNotification({
            type: "new_follower",
            actor: { id: "u2", username: "lambdadelta", display_name: "Lambdadelta" },
        });

        // when
        const route = getNotificationRoute(notif);

        // then
        expect(route).toBe("/user/lambdadelta");
    });

    it("sends both suggestion notifications to the suggestion itself", () => {
        // given
        const posted = makeNotification({ type: "suggestion_posted", reference_id: "s1" });
        const resolved = makeNotification({ type: "suggestion_resolved", reference_id: "s1" });

        // when
        const postedRoute = getNotificationRoute(posted);
        const resolvedRoute = getNotificationRoute(resolved);

        // then
        expect(postedRoute).toBe("/suggestions/s1");
        expect(resolvedRoute).toBe("/suggestions/s1");
    });

    const roomTypes: NotificationType[] = [
        "chat_room_invite",
        "chat_room_banned",
        "chat_room_kicked",
        "chat_room_unbanned",
    ];

    for (const type of roomTypes) {
        it(`sends a ${type} notification to the room`, () => {
            // given
            const notif = makeNotification({ type, reference_id: "room-7", reference_type: "chat_room" });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe("/rooms/room-7");
        });
    }

    const chatMessageTypes: NotificationType[] = ["chat_room_message", "chat_mention", "chat_reply"];

    for (const type of chatMessageTypes) {
        it(`anchors a ${type} notification on the message it refers to`, () => {
            // given
            const notif = makeNotification({ type, reference_id: "room-7", reference_type: "chat_message:m9" });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe("/rooms/room-7#msg-m9");
        });

        it(`sends a ${type} notification to the room when no message is referenced`, () => {
            // given
            const notif = makeNotification({ type, reference_id: "room-7", reference_type: "chat_room" });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe("/rooms/room-7");
        });
    }

    const gameTypes: NotificationType[] = ["game_invite", "game_your_turn", "game_finished"];

    for (const type of gameTypes) {
        it(`sends a ${type} notification to the game named by the reference type`, () => {
            // given
            const notif = makeNotification({ type, reference_id: "g5", reference_type: "othello" });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe("/games/othello/g5");
        });

        it(`defaults a ${type} notification with no game named to chess`, () => {
            // given
            const notif = makeNotification({ type, reference_id: "g5", reference_type: "" });

            // when
            const route = getNotificationRoute(notif);

            // then
            expect(route).toBe("/games/chess/g5");
        });
    }
});

describe("groupByCategory", () => {
    const staticCategories: NotificationCategory[] = [
        "game_board",
        "gallery",
        "theories",
        "mysteries_gm",
        "mysteries_player",
        "social",
        "site_improvements",
        "moderation",
    ];

    for (const category of staticCategories) {
        it(`files every ${category} notification type under that category`, () => {
            // given
            const types = typeCases.filter(testCase => testCase.category === category).map(testCase => testCase.type);

            // when / then
            expect(types.length).toBeGreaterThan(0);
            for (const type of types) {
                const groups = groupByCategory([makeNotification({ type })]);
                expect([...groups.keys()]).toEqual([category]);
            }
        });
    }

    const dynamicCategories: { reference_type: string; category: NotificationCategory }[] = [
        { reference_type: "post", category: "game_board" },
        { reference_type: "post_comment:c9", category: "game_board" },
        { reference_type: "art", category: "gallery" },
        { reference_type: "art_comment:c9", category: "gallery" },
        { reference_type: "theory", category: "theories" },
        { reference_type: "response", category: "theories" },
        { reference_type: "mystery", category: "game_board" },
        { reference_type: "ship", category: "social" },
        { reference_type: "ship_comment:c9", category: "social" },
        { reference_type: "oc", category: "social" },
        { reference_type: "oc_comment:c9", category: "social" },
        { reference_type: "fanfic", category: "social" },
        { reference_type: "", category: "social" },
    ];

    const dynamicTypes: NotificationType[] = ["mention", "comment_liked", "content_edited"];

    for (const type of dynamicTypes) {
        it(`files a ${type} notification under the category of the content it points at`, () => {
            // given
            const notifs = dynamicCategories.map(testCase =>
                makeNotification({ type, reference_type: testCase.reference_type }),
            );

            // when / then
            for (const [index, notif] of notifs.entries()) {
                const groups = groupByCategory([notif]);
                expect([...groups.keys()]).toEqual([dynamicCategories[index].category]);
            }
        });
    }

    it("returns no groups for an empty list", () => {
        // given
        const notifs: Notification[] = [];

        // when
        const groups = groupByCategory(notifs);

        // then
        expect(groups.size).toBe(0);
    });

    it("files an unrecognised notification type under social", () => {
        // given
        const notif = makeNotification({ type: UNKNOWN_TYPE });

        // when
        const groups = groupByCategory([notif]);

        // then
        expect([...groups.keys()]).toEqual(["social"]);
    });

    it("keeps notifications in the order they were given within a group", () => {
        // given
        const first = makeNotification({ id: 1, type: "post_liked" });
        const second = makeNotification({ id: 2, type: "post_commented" });
        const third = makeNotification({ id: 3, type: "post_comment_reply" });

        // when
        const groups = groupByCategory([first, second, third]);

        // then
        expect(groups.get("game_board")).toEqual([first, second, third]);
    });

    it("keys the groups in the order the categories are first seen", () => {
        // given
        const notifs = [
            makeNotification({ id: 1, type: "new_follower" }),
            makeNotification({ id: 2, type: "post_liked" }),
            makeNotification({ id: 3, type: "art_liked" }),
            makeNotification({ id: 4, type: "chat_message" }),
        ];

        // when
        const groups = groupByCategory(notifs);

        // then
        expect([...groups.keys()]).toEqual(["social", "game_board", "gallery"]);
        expect(groups.get("social")).toHaveLength(2);
    });
});

describe("getCategoryLabel", () => {
    it("gives each category a human readable label", () => {
        // given / when / then
        expect(getCategoryLabel("game_board")).toBe("Game Board");
        expect(getCategoryLabel("gallery")).toBe("Gallery");
        expect(getCategoryLabel("theories")).toBe("Theories");
        expect(getCategoryLabel("mysteries_gm")).toBe("Mysteries (as Game Master)");
        expect(getCategoryLabel("mysteries_player")).toBe("Mysteries (as Player)");
        expect(getCategoryLabel("social")).toBe("Social");
        expect(getCategoryLabel("site_improvements")).toBe("Site Improvements");
        expect(getCategoryLabel("moderation")).toBe("Moderation");
    });
});

describe("getCategoryOrder", () => {
    it("lists every category in display order", () => {
        // given / when
        const order = getCategoryOrder();

        // then
        expect(order).toEqual([
            "game_board",
            "gallery",
            "theories",
            "mysteries_gm",
            "mysteries_player",
            "social",
            "site_improvements",
            "moderation",
        ]);
    });

    it("hands out a fresh list so a caller cannot reorder it for everybody else", () => {
        // given
        const order = getCategoryOrder();

        // when
        order.reverse();

        // then
        expect(getCategoryOrder()[0]).toBe("game_board");
    });

    it("gives every ordered category a label", () => {
        // given
        const order = getCategoryOrder();

        // when
        const labels = order.map(getCategoryLabel);

        // then
        expect(labels.filter(label => !label)).toEqual([]);
    });
});

describe("isContentEditedNotification", () => {
    it("recognises a content edited notification", () => {
        // given
        const notif = makeNotification({ type: "content_edited" });

        // when
        const result = isContentEditedNotification(notif);

        // then
        expect(result).toBe(true);
    });

    it("rejects every other notification type", () => {
        // given
        const others = typeCases.filter(testCase => testCase.type !== "content_edited");

        // when / then
        for (const testCase of others) {
            expect(isContentEditedNotification(makeNotification({ type: testCase.type }))).toBe(false);
        }
    });
});

describe("formatContentEditedText", () => {
    it("uses the server message when there is one", () => {
        // given
        const notif = makeNotification({ type: "content_edited", message: "your theory was reworded" });

        // when
        const result = formatContentEditedText(notif);

        // then
        expect(result.message).toBe("your theory was reworded");
    });

    it("falls back to a generic message when there is none", () => {
        // given
        const missing = makeNotification({ type: "content_edited" });
        const empty = makeNotification({ type: "content_edited", message: "" });

        // when / then
        expect(formatContentEditedText(missing).message).toBe("your content has been edited");
        expect(formatContentEditedText(empty).message).toBe("your content has been edited");
    });

    const roleCases: { role: "super_admin" | "admin" | "moderator"; label: string }[] = [
        { role: "super_admin", label: "Reality Author" },
        { role: "admin", label: "Voyager Witch" },
        { role: "moderator", label: "Witch" },
    ];

    for (const testCase of roleCases) {
        it(`shows a ${testCase.role} editor as a ${testCase.label}`, () => {
            // given
            const notif = makeNotification({
                type: "content_edited",
                actor: { id: "u2", username: "lambdadelta", display_name: "Lambdadelta", role: testCase.role },
            });

            // when
            const result = formatContentEditedText(notif);

            // then
            expect(result.role).toBe(testCase.label);
            expect(result.actorName).toBe("Lambdadelta");
        });
    }

    it("leaves the role blank when the actor has none", () => {
        // given
        const notif = makeNotification({ type: "content_edited" });

        // when
        const result = formatContentEditedText(notif);

        // then
        expect(result.role).toBe("");
    });

    it("leaves the role blank when the actor has an unrecognised role", () => {
        // given
        const notif = makeNotification({
            type: "content_edited",
            actor: {
                id: "u2",
                username: "lambdadelta",
                display_name: "Lambdadelta",
                role: "archivist" as "super_admin",
            },
        });

        // when
        const result = formatContentEditedText(notif);

        // then
        expect(result.role).toBe("");
    });
});

class StubNotification {
    static permission = "default";
    static requestPermission: () => Promise<string> = () => Promise.resolve("default");
    static instances: StubNotification[] = [];

    readonly close = vi.fn();
    onclick: (() => void) | null = null;

    constructor(
        readonly title: string,
        readonly options: NotificationOptions = {},
    ) {
        StubNotification.instances.push(this);
    }
}

describe("showDesktopNotification", () => {
    beforeEach(() => {
        StubNotification.instances = [];
        StubNotification.permission = "granted";
        vi.stubGlobal("Notification", StubNotification);
    });

    afterEach(() => {
        vi.restoreAllMocks();
        window.history.replaceState({}, "", "/");
    });

    it("does nothing when the browser has no notification api", () => {
        // given
        vi.unstubAllGlobals();
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(StubNotification.instances).toHaveLength(0);
    });

    it("does nothing when permission has not been granted", () => {
        // given
        StubNotification.permission = "default";
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(StubNotification.instances).toHaveLength(0);
    });

    it("does nothing when permission has been denied", () => {
        // given
        StubNotification.permission = "denied";
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(StubNotification.instances).toHaveLength(0);
    });

    it("does nothing when the tab is already visible and focused", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(true);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(document.visibilityState).toBe("visible");
        expect(StubNotification.instances).toHaveLength(0);
    });

    it("shows a notification when the tab is visible but not focused", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(StubNotification.instances).toHaveLength(1);
    });

    it("titles the notification with the actor name and the notification text", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const notif = makeNotification({
            id: 42,
            type: "post_liked",
            actor: { id: "u2", username: "lambdadelta", display_name: "Lambdadelta", avatar_url: "/uploads/l.png" },
        });

        // when
        showDesktopNotification(notif);

        // then
        const osNotif = StubNotification.instances[0];
        expect(osNotif.title).toBe("Lambdadelta liked your post");
        expect(osNotif.options.icon).toBe("/uploads/l.png");
        expect(osNotif.options.badge).toBe("/favicon/favicon-32x32.png");
        expect(osNotif.options.tag).toBe("notif-42");
    });

    it("carries the server message across as the notification body", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const notif = makeNotification({ type: "post_liked", message: "liked your post about Beatrice" });

        // when
        showDesktopNotification(notif);

        // then
        expect(StubNotification.instances[0].options.body).toBe("liked your post about Beatrice");
    });

    it("uses an empty body when there is no server message", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(StubNotification.instances[0].options.body).toBe("");
    });

    it("falls back to the site icon when the actor has no avatar", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // when
        showDesktopNotification(makeNotification({ type: "post_liked" }));

        // then
        expect(StubNotification.instances[0].options.icon).toBe("/favicon/android-chrome-192x192.png");
    });

    it("titles the notification with the text alone when the actor is missing", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const notif = makeNotification({ type: "post_liked" });
        const malformed = { ...notif, actor: undefined } as unknown as Notification;

        // when
        showDesktopNotification(malformed);

        // then
        expect(StubNotification.instances[0].title).toBe("liked your post");
        expect(StubNotification.instances[0].options.icon).toBe("/favicon/android-chrome-192x192.png");
    });

    it("navigates to the notification route and closes itself when clicked", () => {
        // given
        window.history.replaceState({}, "", "/game-board/p1");
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const focus = vi.spyOn(window, "focus").mockImplementation(() => {});
        showDesktopNotification(
            makeNotification({ type: "post_comment_reply", reference_id: "p1", reference_type: "post_comment:c9" }),
        );
        const osNotif = StubNotification.instances[0];

        // when
        osNotif.onclick?.();

        // then
        expect(window.location.hash).toBe("#comment-c9");
        expect(focus).toHaveBeenCalledOnce();
        expect(osNotif.close).toHaveBeenCalledOnce();
    });
});

describe("ensureNotificationPermission", () => {
    beforeEach(() => {
        StubNotification.instances = [];
        StubNotification.permission = "default";
        StubNotification.requestPermission = () => Promise.resolve("default");
        vi.stubGlobal("Notification", StubNotification);
    });

    it("reports no permission when the browser has no notification api", async () => {
        // given
        vi.unstubAllGlobals();

        // when
        const granted = await ensureNotificationPermission();

        // then
        expect(granted).toBe(false);
    });

    it("reports permission without asking again when it is already granted", async () => {
        // given
        StubNotification.permission = "granted";
        const request = vi.fn(() => Promise.resolve("granted"));
        StubNotification.requestPermission = request;

        // when
        const granted = await ensureNotificationPermission();

        // then
        expect(granted).toBe(true);
        expect(request).not.toHaveBeenCalled();
    });

    it("does not ask again once permission has been denied", async () => {
        // given
        StubNotification.permission = "denied";
        const request = vi.fn(() => Promise.resolve("granted"));
        StubNotification.requestPermission = request;

        // when
        const granted = await ensureNotificationPermission();

        // then
        expect(granted).toBe(false);
        expect(request).not.toHaveBeenCalled();
    });

    it("asks for permission when it has not been decided yet", async () => {
        // given
        const request = vi.fn(() => Promise.resolve("granted"));
        StubNotification.requestPermission = request;

        // when
        const granted = await ensureNotificationPermission();

        // then
        expect(granted).toBe(true);
        expect(request).toHaveBeenCalledOnce();
    });

    it("reports no permission when the user dismisses the prompt", async () => {
        // given
        StubNotification.requestPermission = () => Promise.resolve("default");

        // when
        const granted = await ensureNotificationPermission();

        // then
        expect(granted).toBe(false);
    });

    it("reports no permission when the request fails", async () => {
        // given
        StubNotification.requestPermission = () => Promise.reject(new Error("blocked"));

        // when
        const granted = await ensureNotificationPermission();

        // then
        expect(granted).toBe(false);
    });
});

describe("relativeTime", () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    it("is re-exported so notification lists can stamp their entries", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-01-01T12:00:00Z"));

        // when
        const stamp = relativeTime("2026-01-01T11:30:00Z");

        // then
        expect(stamp).toBe("30m ago");
    });
});
