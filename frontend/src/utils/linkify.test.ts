import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";

type LinkLikeProps = { to?: string; href?: string; target?: string };

describe("linkify origin handling (native app with a configured site origin)", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    it("treats links to the site's own origin as internal router links", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { linkify } = await import("./linkify");

        // when
        const parts = linkify("see https://whentheycry.social/mystery/123 now");
        const element = parts.find(p => typeof p === "object" && p !== null) as ReactElement;

        // then
        expect((element.props as LinkLikeProps).to).toBe("/mystery/123");
    });

    it("treats foreign links as external anchors", async () => {
        // given
        vi.stubEnv("VITE_API_BASE", "https://whentheycry.social");
        vi.resetModules();
        const { linkify } = await import("./linkify");

        // when
        const parts = linkify("see https://example.com/foo now");
        const element = parts.find(p => typeof p === "object" && p !== null) as ReactElement;

        // then
        expect(element.type).toBe("a");
        expect((element.props as LinkLikeProps).href).toBe("https://example.com/foo");
        expect((element.props as LinkLikeProps).target).toBe("_blank");
    });
});

describe("linkify mention handling", () => {
    afterEach(() => {
        vi.resetModules();
    });

    it("defers mentions to MentionLink instead of linking unconditionally", async () => {
        // given
        const { linkify } = await import("./linkify");
        const { MentionLink } = await import("../components/MentionLink/MentionLink");

        // when
        const parts = linkify("hi @foooobaaaaa there");
        const element = parts.find(p => typeof p === "object" && p !== null) as ReactElement;

        // then
        expect(element.type).toBe(MentionLink);
        expect((element.props as { username: string }).username).toBe("foooobaaaaa");
    });

    it("passes the whole token through as the visible label", async () => {
        // given
        const { linkify } = await import("./linkify");

        // when
        const parts = linkify("hi @Featherines_other_half there");
        const element = parts.find(p => typeof p === "object" && p !== null) as ReactElement;

        // then
        expect((element.props as { username: string; label: string }).username).toBe("Featherines_other_half");
        expect((element.props as { username: string; label: string }).label).toBe("@Featherines_other_half");
    });
});
