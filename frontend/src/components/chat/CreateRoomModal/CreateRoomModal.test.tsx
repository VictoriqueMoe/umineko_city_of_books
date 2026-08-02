import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ChatRoom, User } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { CreateRoomModal } from "./CreateRoomModal";

const mocks = vi.hoisted(() => ({
    useMutualFollowers: vi.fn(),
    useSearchUsers: vi.fn(),
    createRoom: vi.fn(),
}));

vi.mock("../../../api/queries/misc", () => ({
    useMutualFollowers: mocks.useMutualFollowers,
    useSearchUsers: mocks.useSearchUsers,
}));

vi.mock("../../../api/mutations/chat", () => ({
    useCreateGroupRoom: () => ({ mutateAsync: mocks.createRoom }),
}));

function makeChatUser(overrides: Partial<User> = {}): User {
    return {
        id: "u-1",
        username: "battler",
        display_name: "Battler",
        ...overrides,
    };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Golden Land",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 1,
        hot_score: 0,
        members: [],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

function renderModal(overrides: { onClose?: () => void; onCreated?: (room: ChatRoom) => void } = {}) {
    const onClose = overrides.onClose ?? vi.fn();
    const onCreated = overrides.onCreated ?? vi.fn();

    const result = renderWithProviders(<CreateRoomModal isOpen onClose={onClose} onCreated={onCreated} />);

    return { ...result, onClose, onCreated };
}

function nameField() {
    return screen.getByPlaceholderText("e.g. Higurashi book club");
}

function tagField() {
    return screen.getByPlaceholderText("Type a tag and press Enter or comma (max 10)");
}

function createButton() {
    return screen.getByRole("button", { name: "Create Room" });
}

beforeEach(() => {
    mocks.useMutualFollowers.mockReturnValue({ mutuals: [], loading: false });
    mocks.useSearchUsers.mockReturnValue({ users: [], loading: false });
    mocks.createRoom.mockResolvedValue(makeRoom());
});

describe("CreateRoomModal", () => {
    it("renders nothing while it is closed", () => {
        // given
        const isOpen = false;

        // when
        const { container } = renderWithProviders(
            <CreateRoomModal isOpen={isOpen} onClose={vi.fn()} onCreated={vi.fn()} />,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("keeps the create action disabled until a name is given", async () => {
        // given
        const user = userEvent.setup();
        renderModal();
        expect(createButton()).toBeDisabled();

        // when
        await user.type(nameField(), "Golden Land");

        // then
        expect(createButton()).toBeEnabled();
    });

    it("treats a whitespace only name as no name at all", async () => {
        // given
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(nameField(), "   ");

        // then
        expect(createButton()).toBeDisabled();
    });

    it("normalises a tag when it is committed with the enter key", async () => {
        // given
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(tagField(), "  Golden Truth!! {Enter}");

        // then
        expect(screen.getByRole("button", { name: /#golden-truth/ })).toBeInTheDocument();
        expect(tagField()).toHaveValue("");
    });

    it("commits the current tag when a comma is typed", async () => {
        // given
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(tagField(), "beato,battler{Enter}");

        // then
        expect(screen.getByRole("button", { name: /#beato/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /#battler/ })).toBeInTheDocument();
    });

    it("splits a pasted comma separated list into separate tags on blur", () => {
        // given
        renderModal();

        // when
        fireEvent.change(tagField(), { target: { value: "beato, battler , beato" } });
        fireEvent.blur(tagField());

        // then
        expect(screen.getByRole("button", { name: /#beato/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /#battler/ })).toBeInTheDocument();
    });

    it("discards an entry that normalises to nothing", async () => {
        // given
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(tagField(), "!!!{Enter}");

        // then
        expect(screen.queryByRole("button", { name: /#/ })).not.toBeInTheDocument();
        expect(tagField()).toHaveValue("");
    });

    it("removes a tag when its chip is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderModal();
        await user.type(tagField(), "beato{Enter}");

        // when
        await user.click(screen.getByRole("button", { name: /#beato/ }));

        // then
        expect(screen.queryByRole("button", { name: /#beato/ })).not.toBeInTheDocument();
    });

    it("removes the last tag when backspace is pressed on an empty tag field", async () => {
        // given
        const user = userEvent.setup();
        renderModal();
        await user.type(tagField(), "beato{Enter}battler{Enter}");

        // when
        await user.type(tagField(), "{Backspace}");

        // then
        expect(screen.getByRole("button", { name: /#beato/ })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /#battler/ })).not.toBeInTheDocument();
    });

    it("stops accepting tags once ten have been added", () => {
        // given
        renderModal();
        const eleven = ["t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9", "t10", "t11"].join(",");

        // when
        fireEvent.change(tagField(), { target: { value: eleven } });
        fireEvent.blur(tagField());

        // then
        expect(screen.getAllByRole("button", { name: /^#t/ })).toHaveLength(10);
        expect(tagField()).toBeDisabled();
        expect(screen.queryByRole("button", { name: /#t11/ })).not.toBeInTheDocument();
    });

    it("offers mutual followers while no search is active", () => {
        // given
        mocks.useMutualFollowers.mockReturnValue({ mutuals: [makeChatUser()], loading: false });

        // when
        renderModal();

        // then
        expect(screen.getByText("Mutual followers")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Battler/ })).toBeInTheDocument();
    });

    it("swaps the mutual followers for the search results once a search is typed", async () => {
        // given
        mocks.useMutualFollowers.mockReturnValue({ mutuals: [makeChatUser()], loading: false });
        mocks.useSearchUsers.mockReturnValue({
            users: [makeChatUser({ id: "u-2", username: "beato", display_name: "Beatrice" })],
            loading: false,
        });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "bea");

        // then
        expect(mocks.useSearchUsers).toHaveBeenLastCalledWith("bea", true);
        expect(screen.getByRole("button", { name: /Beatrice/ })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Battler/ })).not.toBeInTheDocument();
        expect(screen.queryByText("Mutual followers")).not.toBeInTheDocument();
    });

    it("shows an empty state when a search finds nobody", async () => {
        // given
        const user = userEvent.setup();
        renderModal();

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "kinzo");

        // then
        expect(screen.getByText("No users found")).toBeInTheDocument();
    });

    it("adds and removes a user from the invite list", async () => {
        // given
        mocks.useMutualFollowers.mockReturnValue({ mutuals: [makeChatUser()], loading: false });
        const user = userEvent.setup();
        renderModal();

        // when
        await user.click(screen.getByRole("button", { name: /Battler/ }));

        // then
        expect(screen.getByText("Inviting:")).toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: /Battler ✕/ }));
        expect(screen.queryByText("Inviting:")).not.toBeInTheDocument();
    });

    it("submits the trimmed form with its toggles, tags and invited members", async () => {
        // given
        mocks.useMutualFollowers.mockReturnValue({ mutuals: [makeChatUser({ id: "u-7" })], loading: false });
        const room = makeRoom({ id: "room-7" });
        mocks.createRoom.mockResolvedValue(room);
        const user = userEvent.setup();
        const { onCreated, onClose } = renderModal();
        await user.type(nameField(), "  Golden Land  ");
        await user.type(screen.getByPlaceholderText("What's the room about?"), "  tea time  ");
        await user.click(screen.getByRole("switch", { name: "Public" }));
        await user.click(screen.getByRole("switch", { name: "Roleplay (RP)" }));
        await user.type(tagField(), "beato{Enter}");
        await user.click(screen.getByRole("button", { name: /Battler/ }));

        // when
        await user.click(createButton());

        // then
        expect(mocks.createRoom).toHaveBeenCalledWith({
            name: "Golden Land",
            description: "tea time",
            is_public: false,
            is_rp: true,
            tags: ["beato"],
            member_ids: ["u-7"],
        });
        expect(onCreated).toHaveBeenCalledWith(room);
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("includes a tag that was typed but never committed", () => {
        // given
        renderModal();

        // when
        fireEvent.change(nameField(), { target: { value: "Golden Land" } });
        fireEvent.change(tagField(), { target: { value: "Trailing Tag" } });
        fireEvent.click(createButton());

        // then
        expect(mocks.createRoom).toHaveBeenCalledWith(expect.objectContaining({ tags: ["trailing-tag"] }));
    });

    it("shows the failure message and stays open when creation fails", async () => {
        // given
        mocks.createRoom.mockRejectedValue(new Error("that name is taken"));
        const user = userEvent.setup();
        const { onCreated, onClose } = renderModal();
        await user.type(nameField(), "Golden Land");

        // when
        await user.click(createButton());

        // then
        expect(await screen.findByText("that name is taken")).toBeInTheDocument();
        expect(onCreated).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
    });

    it("falls back to a generic failure message for a non error rejection", async () => {
        // given
        mocks.createRoom.mockRejectedValue("boom");
        const user = userEvent.setup();
        renderModal();
        await user.type(nameField(), "Golden Land");

        // when
        await user.click(createButton());

        // then
        expect(await screen.findByText("Failed to create room")).toBeInTheDocument();
    });

    it("shows a busy label while the room is being created", async () => {
        // given
        mocks.createRoom.mockReturnValue(new Promise(() => {}));
        const user = userEvent.setup();
        renderModal();
        await user.type(nameField(), "Golden Land");

        // when
        await user.click(createButton());

        // then
        expect(await screen.findByRole("button", { name: "Creating..." })).toBeDisabled();
    });

    it("closes without creating anything when cancel is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onClose } = renderModal();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(mocks.createRoom).not.toHaveBeenCalled();
    });

    it("clears what was typed once it is closed and opened again", async () => {
        // given
        const user = userEvent.setup();
        const onClose = vi.fn();
        const onCreated = vi.fn();
        const { rerender } = renderWithProviders(
            <CreateRoomModal isOpen={false} onClose={onClose} onCreated={onCreated} />,
        );
        rerender(<CreateRoomModal isOpen onClose={onClose} onCreated={onCreated} />);
        await user.type(nameField(), "Golden Land");
        await user.type(tagField(), "beato{Enter}");

        // when
        rerender(<CreateRoomModal isOpen={false} onClose={onClose} onCreated={onCreated} />);
        rerender(<CreateRoomModal isOpen onClose={onClose} onCreated={onCreated} />);

        // then
        expect(nameField()).toHaveValue("");
        expect(screen.queryByRole("button", { name: /#beato/ })).not.toBeInTheDocument();
    });

    it("clears what was typed when it was first mounted already open", async () => {
        // given
        const user = userEvent.setup();
        const onClose = vi.fn();
        const onCreated = vi.fn();
        const { rerender } = renderWithProviders(<CreateRoomModal isOpen onClose={onClose} onCreated={onCreated} />);
        await user.type(nameField(), "Golden Land");
        await user.type(tagField(), "beato{Enter}");

        // when
        rerender(<CreateRoomModal isOpen={false} onClose={onClose} onCreated={onCreated} />);
        rerender(<CreateRoomModal isOpen onClose={onClose} onCreated={onCreated} />);

        // then
        expect(nameField()).toHaveValue("");
        expect(screen.queryByRole("button", { name: /#beato/ })).not.toBeInTheDocument();
    });
});
