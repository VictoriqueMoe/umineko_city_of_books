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
