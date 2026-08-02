import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { User } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { InviteMembersModal } from "./InviteMembersModal";

const { useMutualFollowers, useSearchUsers } = vi.hoisted(() => ({
    useMutualFollowers: vi.fn(),
    useSearchUsers: vi.fn(),
}));

const { useInviteChatRoomMembers } = vi.hoisted(() => ({ useInviteChatRoomMembers: vi.fn() }));

vi.mock("../../../api/queries/misc", () => ({ useMutualFollowers, useSearchUsers }));
vi.mock("../../../api/mutations/chat", () => ({ useInviteChatRoomMembers }));

const roomId = "room-1";

interface InviteResult {
    invited_count: number;
    skipped_count: number;
}

interface StubOptions {
    mutuals?: User[];
    results?: User[];
    invite?: (userIds: string[]) => Promise<InviteResult>;
}

function makeCandidate(overrides: Partial<User> = {}): User {
    return {
        id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "/avatar.png",
        ...overrides,
    };
}

const beatrice = makeCandidate();
const battler = makeCandidate({ id: "user-2", username: "battler", display_name: "Battler" });

function stubInvites(options: StubOptions = {}) {
    useMutualFollowers.mockReturnValue({ mutuals: options.mutuals ?? [], loading: false });
    useSearchUsers.mockReturnValue({ users: options.results ?? [], loading: false });
    const mutateAsync = vi.fn(options.invite ?? (() => Promise.resolve({ invited_count: 1, skipped_count: 0 })));
    useInviteChatRoomMembers.mockReturnValue({ mutateAsync });

    return { mutateAsync };
}

function renderModal(existingMemberIds: Set<string> = new Set(), isOpen = true) {
    const onClose = vi.fn();
    const onInvited = vi.fn();
    const result = renderWithProviders(
        <InviteMembersModal
            isOpen={isOpen}
            roomId={roomId}
            existingMemberIds={existingMemberIds}
            onClose={onClose}
            onInvited={onInvited}
        />,
    );

    return { ...result, onClose, onInvited };
}

describe("InviteMembersModal", () => {
    it("renders nothing while the modal is closed", () => {
        // given
        stubInvites({ mutuals: [beatrice] });

        // when
        const { container } = renderModal(new Set(), false);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("leaves the mutual followers query disabled while the modal is closed", () => {
        // given
        stubInvites();

        // when
        renderModal(new Set(), false);

        // then
        expect(useMutualFollowers).toHaveBeenLastCalledWith(false);
        expect(useSearchUsers).toHaveBeenLastCalledWith("", false);
    });

    it("offers the mutual followers as the default candidates", () => {
        // given
        stubInvites({ mutuals: [beatrice, battler] });

        // when
        renderModal();

        // then
        expect(screen.getByText("Mutual followers")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Beatrice" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Battler" })).toBeInTheDocument();
    });

    it("leaves out candidates who are already in the room", () => {
        // given
        stubInvites({ mutuals: [beatrice, battler] });

        // when
        renderModal(new Set(["user-2"]));

        // then
        expect(screen.getByRole("button", { name: "Beatrice" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Battler" })).not.toBeInTheDocument();
    });

    it("shows an empty state when every mutual follower is already a member", () => {
        // given
        stubInvites({ mutuals: [beatrice] });

        // when
        renderModal(new Set(["user-1"]));

        // then
        expect(screen.getByText("No mutual followers to invite")).toBeInTheDocument();
        expect(screen.queryByText("Mutual followers")).not.toBeInTheDocument();
    });

    it("searches for users once the typing settles", async () => {
        // given
        stubInvites({ mutuals: [beatrice], results: [battler] });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "batt");

        // then
        await waitFor(() => {
            expect(useSearchUsers).toHaveBeenLastCalledWith("batt", true);
        });
    });

    it("replaces the mutual followers with the search results as soon as a term is typed", async () => {
        // given
        stubInvites({ mutuals: [beatrice], results: [battler] });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "batt");

        // then
        expect(screen.getByRole("button", { name: "Battler" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Beatrice" })).not.toBeInTheDocument();
        expect(screen.queryByText("Mutual followers")).not.toBeInTheDocument();
    });

    it("says no users were found when a search matches nobody", async () => {
        // given
        stubInvites({ mutuals: [beatrice], results: [] });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "lambda");

        // then
        expect(screen.getByText("No users found")).toBeInTheDocument();
    });

    it("marks a candidate as chosen and lets the same click undo it", async () => {
        // given
        stubInvites({ mutuals: [beatrice] });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Beatrice" }));

        // then
        expect(screen.getByRole("button", { name: "Beatrice✓" })).toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Beatrice✓" }));
        expect(screen.getByRole("button", { name: "Beatrice" })).toBeInTheDocument();
    });

    it("drops a selection when its chip is dismissed", async () => {
        // given
        stubInvites({ mutuals: [beatrice] });
        const user = userEvent.setup();
        renderModal();
        await user.click(screen.getByRole("button", { name: "Beatrice" }));

        // when
        await user.click(screen.getByRole("button", { name: "Beatrice ✕" }));

        // then
        expect(screen.queryByRole("button", { name: "Beatrice ✕" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Invite" })).toBeDisabled();
    });

    it("keeps the invite control disabled until somebody is chosen", () => {
        // given
        stubInvites({ mutuals: [beatrice] });

        // when
        renderModal();

        // then
        expect(screen.getByRole("button", { name: "Invite" })).toBeDisabled();
    });

    it("counts the chosen users on the invite control", async () => {
        // given
        stubInvites({ mutuals: [beatrice, battler] });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Beatrice" }));
        await user.click(screen.getByRole("button", { name: "Battler" }));

        // then
        expect(screen.getByRole("button", { name: "Invite 2" })).toBeEnabled();
    });

    it("sends every chosen id, reports the outcome and closes", async () => {
        // given
        const { mutateAsync } = stubInvites({
            mutuals: [beatrice, battler],
            invite: () => Promise.resolve({ invited_count: 2, skipped_count: 0 }),
        });
        const user = userEvent.setup();
        const { onClose, onInvited } = renderModal();
        await user.click(screen.getByRole("button", { name: "Beatrice" }));
        await user.click(screen.getByRole("button", { name: "Battler" }));

        // when
        await user.click(screen.getByRole("button", { name: "Invite 2" }));

        // then
        expect(mutateAsync).toHaveBeenCalledWith(["user-1", "user-2"]);
        expect(onInvited).toHaveBeenCalledWith({ invited_count: 2, skipped_count: 0 });
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("reports the reason an invite was refused and stays open", async () => {
        // given
        stubInvites({ mutuals: [beatrice], invite: () => Promise.reject(new Error("the room is sealed")) });
        const user = userEvent.setup();
        const { onClose, onInvited } = renderModal();
        await user.click(screen.getByRole("button", { name: "Beatrice" }));

        // when
        await user.click(screen.getByRole("button", { name: "Invite 1" }));

        // then
        expect(await screen.findByText("the room is sealed")).toBeInTheDocument();
        expect(onInvited).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
    });

    it("shows a busy label while the invite is in flight", async () => {
        // given
        let release: (value: InviteResult) => void = () => {};
        stubInvites({
            mutuals: [beatrice],
            invite: () =>
                new Promise<InviteResult>(resolve => {
                    release = resolve;
                }),
        });
        const user = userEvent.setup();
        renderModal();
        await user.click(screen.getByRole("button", { name: "Beatrice" }));

        // when
        await user.click(screen.getByRole("button", { name: "Invite 1" }));

        // then
        expect(screen.getByRole("button", { name: "Inviting..." })).toBeDisabled();
        await act(async () => {
            release({ invited_count: 1, skipped_count: 0 });
        });
    });

    it("forgets the previous selection when the modal is opened again", async () => {
        // given
        stubInvites({ mutuals: [beatrice] });
        const user = userEvent.setup();
        const element = (isOpen: boolean) => (
            <InviteMembersModal
                isOpen={isOpen}
                roomId={roomId}
                existingMemberIds={new Set<string>()}
                onClose={() => {}}
            />
        );
        const { rerender } = renderWithProviders(element(false));
        rerender(element(true));
        await user.click(screen.getByRole("button", { name: "Beatrice" }));
        expect(screen.getByRole("button", { name: "Beatrice ✕" })).toBeInTheDocument();

        // when
        rerender(element(false));
        rerender(element(true));

        // then
        expect(screen.queryByRole("button", { name: "Beatrice ✕" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Invite" })).toBeDisabled();
    });

    it("forgets the previous selection when it was first mounted already open", async () => {
        // given
        stubInvites({ mutuals: [beatrice] });
        const user = userEvent.setup();
        const element = (isOpen: boolean) => (
            <InviteMembersModal
                isOpen={isOpen}
                roomId={roomId}
                existingMemberIds={new Set<string>()}
                onClose={() => {}}
            />
        );
        const { rerender } = renderWithProviders(element(true));
        await user.click(screen.getByRole("button", { name: "Beatrice" }));
        expect(screen.getByRole("button", { name: "Beatrice ✕" })).toBeInTheDocument();

        // when
        rerender(element(false));
        rerender(element(true));

        // then
        expect(screen.queryByRole("button", { name: "Beatrice ✕" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Invite" })).toBeDisabled();
    });

    it("closes without inviting anyone when cancel is pressed", async () => {
        // given
        const { mutateAsync } = stubInvites({ mutuals: [beatrice] });
        const user = userEvent.setup();
        const { onClose } = renderModal();
        await user.click(screen.getByRole("button", { name: "Beatrice" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(mutateAsync).not.toHaveBeenCalled();
    });
});
