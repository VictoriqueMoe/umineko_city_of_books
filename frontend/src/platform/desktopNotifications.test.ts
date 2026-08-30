import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { TurnNotice } from "../domain/games/turnNotice";
import type { Notification } from "../types/api";
import {
    ensureNotificationPermission,
    isTabHidden,
    showDesktopNotification,
    showGameTurnNotification,
} from "./desktopNotifications";

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

describe("isTabHidden", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("calls a visible and focused tab present", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(true);

        // then
        expect(document.visibilityState).toBe("visible");
        expect(isTabHidden()).toBe(false);
    });

    it("calls a visible but unfocused tab hidden, because the player is looking elsewhere", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);

        // then
        expect(isTabHidden()).toBe(true);
    });
});

describe("showGameTurnNotification", () => {
    const notice: TurnNotice = {
        title: "Your move in chess",
        body: "It's your turn.",
        tag: "game-turn-r1",
        route: "/games/chess/r1",
    };

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

        // when
        showGameTurnNotification(notice);

        // then
        expect(StubNotification.instances).toHaveLength(0);
    });

    it("does nothing when permission has not been granted", () => {
        // given
        StubNotification.permission = "default";

        // when
        showGameTurnNotification(notice);

        // then
        expect(StubNotification.instances).toHaveLength(0);
    });

    it("carries the title, body and tag of the notice across", () => {
        // when
        showGameTurnNotification(notice);

        // then
        const osNotif = StubNotification.instances[0];
        expect(osNotif.title).toBe("Your move in chess");
        expect(osNotif.options.body).toBe("It's your turn.");
        expect(osNotif.options.tag).toBe("game-turn-r1");
        expect(osNotif.options.icon).toBe("/favicon/android-chrome-192x192.png");
    });

    it("focuses the window and closes itself when clicked", () => {
        // given
        const focus = vi.spyOn(window, "focus").mockImplementation(() => {});
        showGameTurnNotification(notice);
        const osNotif = StubNotification.instances[0];

        // when
        osNotif.onclick?.();

        // then
        expect(focus).toHaveBeenCalledOnce();
        expect(osNotif.close).toHaveBeenCalledOnce();
    });

    it("shows the notice whether or not the tab is in front, because the caller has already decided", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(true);

        // when
        showGameTurnNotification(notice);

        // then
        expect(StubNotification.instances).toHaveLength(1);
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
