import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfo } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { CharacterOptInSection } from "./CharacterOptInSection";

const mocks = vi.hoisted(() => ({
    useChatbotOptIn: vi.fn(),
    mutate: vi.fn(),
    pending: false,
}));

vi.mock("../../api/queries/auth", () => ({ useChatbotOptIn: mocks.useChatbotOptIn }));
vi.mock("../../api/mutations/auth", () => ({
    useUpdateChatbotOptIn: () => ({ mutate: mocks.mutate, isPending: mocks.pending }),
}));

interface SetupOptions {
    siteInfo?: Partial<SiteInfo>;
    optedIn?: boolean;
    loading?: boolean;
}

function setup(options: SetupOptions = {}) {
    mocks.useChatbotOptIn.mockReturnValue({ optedIn: options.optedIn ?? false, loading: options.loading ?? false });

    const user = userEvent.setup();
    const result = renderWithProviders(<CharacterOptInSection />, { siteInfo: options.siteInfo });

    return { ...result, user };
}

function toggle(): HTMLElement {
    return screen.getByRole("switch", { name: "Talk To Characters" });
}

beforeEach(() => {
    mocks.pending = false;
    mocks.mutate.mockReset();
});

describe("CharacterOptInSection visibility", () => {
    it("says nothing at all while anyone may summon a character", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: false } };

        // when
        setup(options);

        // then
        expect(screen.queryByRole("switch", { name: "Talk To Characters" })).not.toBeInTheDocument();
        expect(screen.queryByText("Characters")).not.toBeInTheDocument();
    });

    it("does not even ask the server for a state it will never show", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: false } };

        // when
        setup(options);

        // then
        expect(mocks.useChatbotOptIn).toHaveBeenCalledWith(false);
    });

    it("offers the choice once characters are restricted to a permission", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: true } };

        // when
        setup(options);

        // then
        expect(toggle()).toBeEnabled();
        expect(
            screen.queryByText("Characters are switched off across the site at the moment."),
        ).not.toBeInTheDocument();
    });

    it("seals the choice and explains itself while characters are switched off site wide", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: false, chatbot_require_permission: true } };

        // when
        setup(options);

        // then
        expect(toggle()).toBeDisabled();
        expect(screen.getByText("Characters are switched off across the site at the moment.")).toBeInTheDocument();
    });

    it("still shows the true answer while the choice is sealed", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: false, chatbot_require_permission: true }, optedIn: true };

        // when
        setup(options);

        // then
        expect(toggle()).toHaveAttribute("aria-checked", "true");
    });

    it("warns that opting in hands over the whole role", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: true } };

        // when
        setup(options);

        // then
        expect(screen.getByText(/Opting in gives you the role that carries this/)).toBeInTheDocument();
    });
});

describe("CharacterOptInSection opting in and out", () => {
    it("starts switched off for a member who has not opted in", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: true }, optedIn: false };

        // when
        setup(options);

        // then
        expect(toggle()).toHaveAttribute("aria-checked", "false");
    });

    it("opts the member in when the switch is pressed while off", async () => {
        // given
        const { user } = setup({
            siteInfo: { chatbot_enabled: true, chatbot_require_permission: true },
            optedIn: false,
        });

        // when
        await user.click(toggle());

        // then
        expect(mocks.mutate).toHaveBeenCalledWith(true, expect.anything());
    });

    it("opts the member back out when the switch is pressed while on", async () => {
        // given
        const { user } = setup({
            siteInfo: { chatbot_enabled: true, chatbot_require_permission: true },
            optedIn: true,
        });

        // when
        await user.click(toggle());

        // then
        expect(mocks.mutate).toHaveBeenCalledWith(false, expect.anything());
    });

    it("holds the switch still until the current state has arrived", () => {
        // given
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: true }, loading: true };

        // when
        setup(options);

        // then
        expect(toggle()).toBeDisabled();
    });

    it("holds the switch still while the change is on its way", () => {
        // given
        mocks.pending = true;
        const options = { siteInfo: { chatbot_enabled: true, chatbot_require_permission: true } };

        // when
        setup(options);

        // then
        expect(toggle()).toBeDisabled();
    });

    it("repeats whatever the server said when the change was refused", async () => {
        // given
        mocks.mutate.mockImplementation((_optedIn: boolean, handlers: { onError: (e: Error) => void }) => {
            handlers.onError(new Error("character opt-in is not available right now"));
        });
        const { user } = setup({ siteInfo: { chatbot_enabled: true, chatbot_require_permission: true } });

        // when
        await user.click(toggle());

        // then
        expect(screen.getByText("character opt-in is not available right now")).toBeInTheDocument();
    });

    it("clears an earlier refusal when the member tries again", async () => {
        // given
        mocks.mutate.mockImplementationOnce((_optedIn: boolean, handlers: { onError: (e: Error) => void }) => {
            handlers.onError(new Error("character opt-in is not available right now"));
        });
        const { user } = setup({ siteInfo: { chatbot_enabled: true, chatbot_require_permission: true } });
        await user.click(toggle());

        // when
        await user.click(toggle());

        // then
        expect(screen.queryByText("character opt-in is not available right now")).not.toBeInTheDocument();
    });
});
