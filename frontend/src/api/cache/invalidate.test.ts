import type { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createTestQueryClient } from "../../test-utils/render";
import { queryKeys } from "../queryKeys";
import { invalidateViews, type ViewDomain } from "./invalidate";

function seededClient() {
    const qc = createTestQueryClient();
    const keys = [
        queryKeys.post.feed({ corner: "general" }),
        queryKeys.post.detail("p-1"),
        queryKeys.user.posts("u-1", { limit: 20 }),
        queryKeys.user.art("u-1", {}),
        queryKeys.user.galleries("u-1", {}),
        queryKeys.user.mysteries("u-1", {}),
        queryKeys.user.journals("u-1", {}),
        queryKeys.art.feed({}),
        queryKeys.gallery.detail("g-1", {}),
        queryKeys.gallery.list(""),
        queryKeys.theory.feed({}),
        queryKeys.mystery.list({}),
        queryKeys.profile.byUsername("ange"),
        queryKeys.chat.userRooms(),
        queryKeys.chat.roomsListJoined(),
        queryKeys.gameRoom.scoreboard("chess"),
        queryKeys.gameRoom.finished("", { limit: 20, offset: 0 }),
        queryKeys.gameRoom.detail("r-1"),
        queryKeys.notifications.unreadCount(),
    ];
    for (const key of keys) {
        qc.setQueryDefaults(key, { gcTime: Infinity });
        qc.setQueryData(key, {});
    }

    return qc;
}

function invalidatedKeys(qc: QueryClient): string[] {
    return qc
        .getQueryCache()
        .findAll()
        .filter(query => query.state.isInvalidated)
        .map(query => JSON.stringify(query.queryKey));
}

describe("invalidateViews", () => {
    it.each<{ domain: ViewDomain; want: readonly (readonly unknown[])[] }>([
        {
            domain: "post",
            want: [
                queryKeys.post.feed({ corner: "general" }),
                queryKeys.post.detail("p-1"),
                queryKeys.user.posts("u-1", { limit: 20 }),
            ],
        },
        {
            domain: "art",
            want: [
                queryKeys.user.art("u-1", {}),
                queryKeys.user.galleries("u-1", {}),
                queryKeys.art.feed({}),
                queryKeys.gallery.detail("g-1", {}),
                queryKeys.gallery.list(""),
            ],
        },
        {
            domain: "mystery",
            want: [
                queryKeys.user.mysteries("u-1", {}),
                queryKeys.mystery.list({}),
                queryKeys.profile.byUsername("ange"),
            ],
        },
        { domain: "theory", want: [queryKeys.theory.feed({}), queryKeys.profile.byUsername("ange")] },
        { domain: "journal", want: [queryKeys.user.journals("u-1", {})] },
        { domain: "rooms", want: [queryKeys.chat.userRooms(), queryKeys.chat.roomsListJoined()] },
        {
            domain: "gameRooms",
            want: [queryKeys.gameRoom.scoreboard("chess"), queryKeys.gameRoom.finished("", { limit: 20, offset: 0 })],
        },
    ])("refreshes every view of $domain and nothing else", ({ domain, want }) => {
        // given a cache holding views of every domain
        const qc = seededClient();

        // when
        invalidateViews(qc, domain);

        // then
        expect(invalidatedKeys(qc).sort()).toEqual(want.map(key => JSON.stringify(key)).sort());
    });
});
