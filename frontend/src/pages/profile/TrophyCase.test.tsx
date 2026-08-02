import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { SiteInfo, SiteInfoSecret, VanityRoleDefinition } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { UserProfile } from "../../types/api";
import { TrophyCase } from "./TrophyCase";

vi.mock("../../features/easterEgg", () => ({
    HuntPanel: ({ secretId, onClose }: { secretId: string; isOpen: boolean; onClose: () => void }) => (
        <div data-testid="hunt-panel">
            <span>hunt panel for {secretId}</span>
            <button type="button" onClick={onClose}>
                close hunt panel
            </button>
        </div>
    ),
}));

const profileUserId = "profile-1";

function makeSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return {
        id: "epitaph",
        title: "The Witch's Epitaph",
        description: "Seek the key that opens the golden land.",
        solved: false,
        pieces: [],
        ...overrides,
    };
}

function makeVanityRole(overrides: Partial<VanityRoleDefinition> = {}): VanityRoleDefinition {
    return {
        id: "role-1",
        label: "Witch Hunter",
        color: "#ff0000",
        is_system: false,
        sort_order: 0,
        ...overrides,
    };
}

interface SetupOptions {
    secrets?: SiteInfoSecret[];
    vanityRoles?: VanityRoleDefinition[];
    profileSecrets?: string[];
    localSecrets?: string[];
    viewer?: UserProfile | null;
}

function setup(options: SetupOptions = {}) {
    const localSecrets = options.localSecrets ?? [];
    const siteInfo: Partial<SiteInfo> = {
        listed_secrets: options.secrets ?? [makeSecret()],
        vanity_roles: options.vanityRoles ?? [],
    };

    const user = userEvent.setup();
    const result = renderWithProviders(
        <TrophyCase profileUserId={profileUserId} profileSecrets={options.profileSecrets} />,
        {
            user: options.viewer === undefined ? makeUser({ id: profileUserId }) : options.viewer,
            siteInfo,
            theme: { hasSecret: (id: string) => localSecrets.includes(id) },
        },
    );

    return { ...result, user };
}

describe("TrophyCase", () => {
    it("stays hidden while the player has solved nothing", () => {
        // given
        const options = { profileSecrets: [] };

        // when
        const { container } = setup(options);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows off the achievements the profile has solved", () => {
        // given
        const options = { profileSecrets: ["epitaph"] };

        // when
        setup(options);

        // then
        expect(screen.getByRole("heading", { name: "Achievements" })).toBeInTheDocument();
        expect(screen.getByText("The Witch's Epitaph")).toBeInTheDocument();
    });

    it("leaves out the secrets this profile has not solved", () => {
        // given
        const options = {
            secrets: [makeSecret(), makeSecret({ id: "goldsmith", title: "The Goldsmith" })],
            profileSecrets: ["epitaph"],
        };

        // when
        setup(options);

        // then
        expect(screen.getByText("The Witch's Epitaph")).toBeInTheDocument();
        expect(screen.queryByText("The Goldsmith")).not.toBeInTheDocument();
    });

    it("counts an owner's locally remembered secret as solved", () => {
        // given
        const options = { profileSecrets: [], localSecrets: ["epitaph"] };

        // when
        setup(options);

        // then
        expect(screen.getByText("The Witch's Epitaph")).toBeInTheDocument();
    });

    it("ignores a visitor's own locally remembered secrets", () => {
        // given
        const options = {
            profileSecrets: [],
            localSecrets: ["epitaph"],
            viewer: makeUser({ id: "someone-else" }),
        };

        // when
        const { container } = setup(options);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("lets the owner press a trophy to reopen its hunt", async () => {
        // given
        const { user } = setup({ profileSecrets: ["epitaph"] });

        // when
        await user.click(screen.getByRole("button", { name: /The Witch's Epitaph/ }));

        // then
        expect(screen.getByText("hunt panel for epitaph")).toBeInTheDocument();
    });

    it("closes the hunt again when the owner dismisses it", async () => {
        // given
        const { user } = setup({ profileSecrets: ["epitaph"] });
        await user.click(screen.getByRole("button", { name: /The Witch's Epitaph/ }));

        // when
        await user.click(screen.getByRole("button", { name: "close hunt panel" }));

        // then
        expect(screen.queryByTestId("hunt-panel")).not.toBeInTheDocument();
    });

    it("shows a visitor the trophies without anything to press", () => {
        // given
        const options = { profileSecrets: ["epitaph"], viewer: makeUser({ id: "someone-else" }) };

        // when
        setup(options);

        // then
        expect(screen.getByText("The Witch's Epitaph")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /The Witch's Epitaph/ })).not.toBeInTheDocument();
    });

    it("shows a signed out visitor the trophies without anything to press", () => {
        // given
        const options = { profileSecrets: ["epitaph"], viewer: null };

        // when
        setup(options);

        // then
        expect(screen.getByText("The Witch's Epitaph")).toBeInTheDocument();
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("borrows the colour of the vanity role the secret grants", () => {
        // given
        const options = {
            secrets: [makeSecret({ vanity_role_id: "role-1" })],
            vanityRoles: [makeVanityRole()],
            profileSecrets: ["epitaph"],
        };

        // when
        setup(options);

        // then
        expect(screen.getByRole("button", { name: /The Witch's Epitaph/ })).toHaveStyle({ borderColor: "#ff0000" });
    });

    it("falls back to gold when the secret grants no vanity role", () => {
        // given
        const options = { profileSecrets: ["epitaph"] };

        // when
        setup(options);

        // then
        expect(screen.getByRole("button", { name: /The Witch's Epitaph/ })).toHaveStyle({ borderColor: "#d4a84b" });
    });

    it("falls back to a star when the secret has no icon of its own", () => {
        // given
        const options = { profileSecrets: ["epitaph"] };

        // when
        setup(options);

        // then
        expect(screen.getByText("★")).toBeInTheDocument();
    });

    it("uses the icon the secret carries", () => {
        // given
        const options = { secrets: [makeSecret({ icon: "♛" })], profileSecrets: ["epitaph"] };

        // when
        setup(options);

        // then
        expect(screen.getByText("♛")).toBeInTheDocument();
    });

    it("explains the secret through the trophy's tooltip", () => {
        // given
        const options = { profileSecrets: ["epitaph"] };

        // when
        setup(options);

        // then
        expect(screen.getByRole("button", { name: /The Witch's Epitaph/ })).toHaveAttribute(
            "title",
            "Seek the key that opens the golden land.",
        );
    });
});
