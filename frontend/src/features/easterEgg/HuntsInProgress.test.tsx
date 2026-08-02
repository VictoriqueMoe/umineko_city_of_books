import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { SiteInfoSecret } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { HuntsInProgress } from "./HuntsInProgress";

const mocks = vi.hoisted(() => ({ unlock: vi.fn() }));

vi.mock("../../api/mutations/secret", () => ({
    useUnlockSecret: () => ({ mutateAsync: mocks.unlock }),
}));

const ownerId = "11111111-1111-1111-1111-111111111111";

const epitaph: SiteInfoSecret = {
    id: "epitaph",
    title: "Witch's Epitaph",
    description: "Seek the key that opens the golden land.",
    icon: "\u{1f52e}",
    solved: false,
    pieces: [
        { id: "epitaph-1", letter: "B", tile: 1 },
        { id: "epitaph-2", letter: "E", tile: 2 },
    ],
};

const catbox: SiteInfoSecret = {
    id: "catbox",
    title: "Schrodinger's Catbox",
    description: "The cat is neither here nor there.",
    icon: "\u{1f431}",
    solved: false,
    pieces: [
        { id: "catbox-1", letter: "C", tile: 1 },
        { id: "catbox-2", letter: "A", tile: 2 },
    ],
};

interface SetupOptions {
    hunts?: SiteInfoSecret[];
    collected?: string[];
    profileUserId?: string;
}

function setup(options: SetupOptions = {}) {
    const collected = new Set(options.collected ?? ["epitaph-1", "catbox-1"]);

    return renderWithProviders(<HuntsInProgress profileUserId={options.profileUserId ?? ownerId} />, {
        user: makeUser({ id: ownerId }),
        siteInfo: { listed_secrets: options.hunts ?? [epitaph, catbox] },
        theme: { hasSecret: id => collected.has(id) },
    });
}

describe("HuntsInProgress", () => {
    it("shows an icon for every hunt the visitor has started", () => {
        // given
        const hunts = [epitaph, catbox];

        // when
        setup({ hunts });

        // then
        expect(screen.getByRole("button", { name: "Witch's Epitaph: 1 of 2 pieces" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Schrodinger's Catbox: 1 of 2 pieces" })).toBeInTheDocument();
        expect(screen.getAllByRole("button")).toHaveLength(2);
    });

    it("leaves out a hunt that has not been started yet", () => {
        // given
        const collected = ["epitaph-1"];

        // when
        setup({ collected });

        // then
        expect(screen.getByRole("button", { name: "Witch's Epitaph: 1 of 2 pieces" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Schrodinger's Catbox/ })).not.toBeInTheDocument();
    });

    it("leaves out a hunt whose reward has already been claimed", () => {
        // given
        const collected = ["epitaph-1", "catbox-1", "catbox-2", "catbox"];

        // when
        setup({ collected });

        // then
        expect(screen.getByRole("button", { name: "Witch's Epitaph: 1 of 2 pieces" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Schrodinger's Catbox/ })).not.toBeInTheDocument();
    });

    it("shows nothing when the site lists no hunts", () => {
        // given
        const hunts: SiteInfoSecret[] = [];

        // when
        const { container } = setup({ hunts });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows nothing when the site info carries no hunts at all", () => {
        // given
        const hunts = undefined;

        // when
        const { container } = renderWithProviders(<HuntsInProgress profileUserId={ownerId} />, {
            user: makeUser({ id: ownerId }),
            siteInfo: { listed_secrets: hunts },
            theme: { hasSecret: () => true },
        });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows nothing on somebody else's profile", () => {
        // given
        const profileUserId = "22222222-2222-2222-2222-222222222222";

        // when
        const { container } = setup({ profileUserId });

        // then
        expect(container).toBeEmptyDOMElement();
    });
});
