import { describe, expect, it, vi } from "vitest";
import { resolveViewerMeta } from "./viewerMeta";

vi.mock("../client", () => ({
    absolutizeMedia: (meta: { avatarUrl?: string }) => ({
        ...meta,
        avatarUrl: meta.avatarUrl ? `https://cdn.test${meta.avatarUrl}` : meta.avatarUrl,
    }),
}));

describe("resolveViewerMeta", () => {
    it("puts the media origin back on a viewer's avatar", () => {
        // given
        const meta = { userId: "user-1", username: "beatrice", avatarUrl: "/uploads/beato.png" };

        // when
        const resolved = resolveViewerMeta(meta);

        // then
        expect(resolved.avatarUrl).toBe("https://cdn.test/uploads/beato.png");
        expect(resolved.username).toBe("beatrice");
    });

    it("leaves a viewer with no avatar alone", () => {
        // given
        const meta = { userId: "user-2" };

        // when
        const resolved = resolveViewerMeta(meta);

        // then
        expect(resolved).toEqual({ userId: "user-2", avatarUrl: undefined });
    });
});
