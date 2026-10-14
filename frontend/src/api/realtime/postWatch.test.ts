import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as OutboundModule from "./outbound";
import type * as PostWatchModule from "./postWatch";

const holder = vi.hoisted(() => ({
    sendRealtime: vi.fn(),
    epochListeners: new Set<() => void>(),
}));

vi.mock("./outbound", async importOriginal => {
    const actual = await importOriginal<typeof OutboundModule>();

    return { ...actual, sendRealtime: holder.sendRealtime };
});

vi.mock("./pipeline", () => ({
    subscribeRealtimeEpoch: (listener: () => void) => {
        holder.epochListeners.add(listener);

        return () => {
            holder.epochListeners.delete(listener);
        };
    },
}));

let postWatch: typeof PostWatchModule;

function reconnect(): void {
    for (const listener of Array.from(holder.epochListeners)) {
        listener();
    }
}

function settle(): void {
    vi.advanceTimersByTime(postWatch.POST_WATCH_DEBOUNCE_MS);
}

function sentPostIds(): string[][] {
    return holder.sendRealtime.mock.calls.map(call => {
        const command = call[0] as OutboundModule.RealtimeCommand;
        if (command.type !== "watch_posts") {
            throw new Error(`unexpected command ${command.type}`);
        }

        return command.data.post_ids;
    });
}

beforeEach(async () => {
    vi.useFakeTimers();
    holder.sendRealtime.mockReset();
    holder.epochListeners.clear();
    vi.resetModules();
    postWatch = await import("./postWatch");
});

afterEach(() => {
    vi.useRealTimers();
});

describe("post watch debounce", () => {
    it("sends one watch_posts carrying every post a feed mounts at once", () => {
        // given
        postWatch.watchPost("post-1");
        postWatch.watchPost("post-2");
        postWatch.watchPost("post-3");

        // when
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1", "post-2", "post-3"]]);
    });

    it("sends nothing until the debounce settles", () => {
        // given
        postWatch.watchPost("post-1");

        // when
        vi.advanceTimersByTime(postWatch.POST_WATCH_DEBOUNCE_MS - 1);

        // then
        expect(holder.sendRealtime).not.toHaveBeenCalled();
    });

    it("folds a post mounted mid-window into the same command", () => {
        // given
        postWatch.watchPost("post-1");
        vi.advanceTimersByTime(postWatch.POST_WATCH_DEBOUNCE_MS - 1);

        // when
        postWatch.watchPost("post-2");
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1", "post-2"]]);
    });

    it("sends nothing while the set is empty and was already empty", () => {
        // given
        const release = postWatch.watchPost("post-1");

        // when
        release();
        settle();

        // then
        expect(holder.sendRealtime).not.toHaveBeenCalled();
    });

    it("caps what it sends at the most recently watched posts", () => {
        // given
        const total = postWatch.MAX_WATCHED_POSTS + 50;
        for (let index = 0; index < total; index += 1) {
            postWatch.watchPost(`post-${index}`);
        }

        // when
        settle();

        // then
        const [sent] = sentPostIds();
        expect(sent).toHaveLength(postWatch.MAX_WATCHED_POSTS);
        expect(sent[0]).toBe("post-50");
        expect(sent[sent.length - 1]).toBe(`post-${total - 1}`);
    });
});

describe("post watch refcount", () => {
    it("keeps a post watched while another holder still has it", () => {
        // given
        const first = postWatch.watchPost("post-1");
        postWatch.watchPost("post-1");
        settle();

        // when
        first();
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1"]]);
    });

    it("drops a post once its last holder lets go", () => {
        // given
        const first = postWatch.watchPost("post-1");
        const second = postWatch.watchPost("post-1");
        postWatch.watchPost("post-2");
        settle();

        // when
        first();
        second();
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1", "post-2"], ["post-2"]]);
    });

    it("clears the server set when the last post is unwatched", () => {
        // given
        const release = postWatch.watchPost("post-1");
        settle();

        // when
        release();
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1"], []]);
    });

    it("ignores a second call to the same unwatch", () => {
        // given
        const first = postWatch.watchPost("post-1");
        postWatch.watchPost("post-1");
        settle();

        // when
        first();
        first();
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1"]]);
    });
});

describe("post watch reconnect", () => {
    it("re-sends the full set when the socket reconnects", () => {
        // given
        postWatch.watchPost("post-1");
        postWatch.watchPost("post-2");
        settle();

        // when
        reconnect();

        // then
        expect(sentPostIds()).toEqual([
            ["post-1", "post-2"],
            ["post-1", "post-2"],
        ]);
    });

    it("sends a pending change straight away on reconnect and not again when the timer would have fired", () => {
        // given
        postWatch.watchPost("post-1");

        // when
        reconnect();
        settle();

        // then
        expect(sentPostIds()).toEqual([["post-1"]]);
    });

    it("sends nothing on reconnect when nothing is watched", () => {
        // given
        const release = postWatch.watchPost("post-1");
        settle();
        release();
        settle();
        holder.sendRealtime.mockClear();

        // when
        reconnect();

        // then
        expect(holder.sendRealtime).not.toHaveBeenCalled();
    });
});
