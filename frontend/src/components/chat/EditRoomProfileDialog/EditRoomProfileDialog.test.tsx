import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ChatRoomMember } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { EditRoomProfileDialog } from "./EditRoomProfileDialog";

const mocks = vi.hoisted(() => ({
    useUpdateChatRoomNickname: vi.fn(),
    useUploadChatRoomAvatar: vi.fn(),
    useClearChatRoomAvatar: vi.fn(),
    updateNickname: vi.fn(),
    uploadAvatar: vi.fn(),
    clearAvatar: vi.fn(),
}));

vi.mock("../../../api/mutations/chat", () => ({
    useUpdateChatRoomNickname: mocks.useUpdateChatRoomNickname,
    useUploadChatRoomAvatar: mocks.useUploadChatRoomAvatar,
    useClearChatRoomAvatar: mocks.useClearChatRoomAvatar,
}));

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: { id: "u-1", username: "battler", display_name: "Battler", avatar_url: "" },
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

function renderDialog(member: ChatRoomMember | null, overrides: { onClose?: () => void } = {}) {
    const onClose = overrides.onClose ?? vi.fn();
    const onSaved = vi.fn();

    const result = renderWithProviders(
        <EditRoomProfileDialog isOpen roomId="room-1" currentMember={member} onClose={onClose} onSaved={onSaved} />,
    );

    return { ...result, onClose, onSaved };
}

function nicknameField() {
    return screen.getByLabelText("Nickname");
}

function saveButton() {
    return screen.getByRole("button", { name: "Save" });
}

beforeEach(() => {
    mocks.useUpdateChatRoomNickname.mockReturnValue({ mutateAsync: mocks.updateNickname });
    mocks.useUploadChatRoomAvatar.mockReturnValue({ mutateAsync: mocks.uploadAvatar });
    mocks.useClearChatRoomAvatar.mockReturnValue({ mutateAsync: mocks.clearAvatar });
    mocks.updateNickname.mockResolvedValue(makeMember({ nickname: "Beato" }));
    mocks.uploadAvatar.mockResolvedValue(makeMember({ member_avatar_url: "https://cdn.test/new.png" }));
    mocks.clearAvatar.mockResolvedValue(makeMember());
});

describe("EditRoomProfileDialog", () => {
    it("renders nothing while it is closed", () => {
        // given
        const isOpen = false;

        // when
        const { container } = renderWithProviders(
            <EditRoomProfileDialog
                isOpen={isOpen}
                roomId="room-1"
                currentMember={makeMember()}
                onClose={vi.fn()}
                onSaved={vi.fn()}
            />,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing when the viewer is not a member of the room", () => {
        // given
        const member = null;

        // when
        const { container } = renderDialog(member);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("scopes its mutations to the room it was given", () => {
        // given
        const member = makeMember();

        // when
        renderDialog(member);

        // then
        expect(mocks.useUpdateChatRoomNickname).toHaveBeenCalledWith("room-1");
        expect(mocks.useUploadChatRoomAvatar).toHaveBeenCalledWith("room-1");
        expect(mocks.useClearChatRoomAvatar).toHaveBeenCalledWith("room-1");
    });

    it("prefills the existing nickname and the characters left", () => {
        // given
        const member = makeMember({ nickname: "Beato" });

        // when
        renderDialog(member);

        // then
        expect(nicknameField()).toHaveValue("Beato");
        expect(screen.getByText("27 characters remaining")).toBeInTheDocument();
    });

    it("counts down the remaining characters as the nickname is typed", async () => {
        // given
        const user = userEvent.setup();
        renderDialog(makeMember());
        expect(screen.getByText("32 characters remaining")).toBeInTheDocument();

        // when
        await user.type(nicknameField(), "Beato");

        // then
        expect(screen.getByText("27 characters remaining")).toBeInTheDocument();
    });

    it("falls back to the display name as the nickname placeholder", () => {
        // given
        const member = makeMember();

        // when
        renderDialog(member);

        // then
        expect(nicknameField()).toHaveAttribute("placeholder", "Battler");
    });

    it("shows the initial of the nickname when no avatar has been set", () => {
        // given
        const member = makeMember({ nickname: "Zepar" });

        // when
        renderDialog(member);

        // then
        expect(screen.getByText("Z")).toBeInTheDocument();
    });

    it("shows the member avatar in preference to the account avatar", () => {
        // given
        const member = makeMember({
            member_avatar_url: "https://cdn.test/member.png",
            user: { id: "u-1", username: "battler", display_name: "Battler", avatar_url: "https://cdn.test/user.png" },
        });

        // when
        const { container } = renderDialog(member);

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "https://cdn.test/member.png");
    });

    it("locks every control and explains why when a moderator has locked the profile", () => {
        // given
        const member = makeMember({ nickname_locked: true });

        // when
        renderDialog(member);

        // then
        expect(
            screen.getByText(
                "Your profile in this room has been locked by a moderator. Contact a moderator to unlock.",
            ),
        ).toBeInTheDocument();
        expect(nicknameField()).toBeDisabled();
        expect(screen.getByRole("button", { name: "Choose avatar" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Clear" })).toBeDisabled();
        expect(saveButton()).toBeDisabled();
    });

    it("saves a trimmed nickname and hands the updated member back", async () => {
        // given
        const updated = makeMember({ nickname: "Beato" });
        mocks.updateNickname.mockResolvedValue(updated);
        const user = userEvent.setup();
        const { onSaved, onClose } = renderDialog(makeMember());
        await user.type(nicknameField(), "  Beato  ");

        // when
        await user.click(saveButton());

        // then
        expect(mocks.updateNickname).toHaveBeenCalledWith("Beato");
        expect(onSaved).toHaveBeenCalledWith(updated);
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("skips the nickname request when the nickname was left alone", async () => {
        // given
        const member = makeMember({ nickname: "Beato" });
        const user = userEvent.setup();
        const { onSaved } = renderDialog(member);

        // when
        await user.click(saveButton());

        // then
        expect(mocks.updateNickname).not.toHaveBeenCalled();
        expect(onSaved).toHaveBeenCalledWith(member);
    });

    it("holds a chosen avatar back until the save is confirmed", async () => {
        // given
        const file = new File(["gold"], "beato.png", { type: "image/png" });
        const user = userEvent.setup();
        const { container } = renderDialog(makeMember());
        const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;

        // when
        await user.upload(fileInput, file);

        // then
        expect(screen.getByText("Saved when you click Save.")).toBeInTheDocument();
        expect(mocks.uploadAvatar).not.toHaveBeenCalled();
    });

    it("uploads the chosen avatar and reports the member the upload returned", async () => {
        // given
        const uploaded = makeMember({ member_avatar_url: "https://cdn.test/new.png" });
        mocks.uploadAvatar.mockResolvedValue(uploaded);
        const file = new File(["gold"], "beato.png", { type: "image/png" });
        const user = userEvent.setup();
        const { container, onSaved } = renderDialog(makeMember());
        const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
        await user.upload(fileInput, file);

        // when
        await user.click(saveButton());

        // then
        expect(mocks.uploadAvatar).toHaveBeenCalledWith(file);
        expect(onSaved).toHaveBeenCalledWith(uploaded);
    });

    it("keeps the dialog open and shows why the save failed", async () => {
        // given
        mocks.updateNickname.mockRejectedValue(new Error("nickname already taken"));
        const user = userEvent.setup();
        const { onSaved, onClose } = renderDialog(makeMember());
        await user.type(nicknameField(), "Beato");

        // when
        await user.click(saveButton());

        // then
        expect(await screen.findByText("nickname already taken")).toBeInTheDocument();
        expect(onSaved).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
    });

    it("falls back to a generic message when the save fails without an error", async () => {
        // given
        mocks.updateNickname.mockRejectedValue("boom");
        const user = userEvent.setup();
        renderDialog(makeMember());
        await user.type(nicknameField(), "Beato");

        // when
        await user.click(saveButton());

        // then
        expect(await screen.findByText("Failed to save profile")).toBeInTheDocument();
    });

    it("shows a busy label and blocks cancelling while the save is in flight", async () => {
        // given
        mocks.updateNickname.mockReturnValue(new Promise(() => {}));
        const user = userEvent.setup();
        renderDialog(makeMember());
        await user.type(nicknameField(), "Beato");

        // when
        await user.click(saveButton());

        // then
        expect(await screen.findByRole("button", { name: "Saving..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    });

    it("clears both the nickname and the avatar", async () => {
        // given
        const cleared = makeMember();
        mocks.clearAvatar.mockResolvedValue(cleared);
        const user = userEvent.setup();
        const { onSaved, onClose } = renderDialog(makeMember({ nickname: "Beato" }));

        // when
        await user.click(screen.getByRole("button", { name: "Clear" }));

        // then
        expect(mocks.updateNickname).toHaveBeenCalledWith("");
        expect(mocks.clearAvatar).toHaveBeenCalledOnce();
        expect(onSaved).toHaveBeenCalledWith(cleared);
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("still clears the nickname when there was no avatar to remove", async () => {
        // given
        const withoutNickname = makeMember();
        mocks.updateNickname.mockResolvedValue(withoutNickname);
        mocks.clearAvatar.mockRejectedValue(new Error("no avatar"));
        const user = userEvent.setup();
        const { onSaved, onClose } = renderDialog(makeMember({ nickname: "Beato" }));

        // when
        await user.click(screen.getByRole("button", { name: "Clear" }));

        // then
        expect(onSaved).toHaveBeenCalledWith(withoutNickname);
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("reports a failure to clear the nickname", async () => {
        // given
        mocks.updateNickname.mockRejectedValue(new Error("the room refuses"));
        const user = userEvent.setup();
        const { onSaved } = renderDialog(makeMember({ nickname: "Beato" }));

        // when
        await user.click(screen.getByRole("button", { name: "Clear" }));

        // then
        expect(await screen.findByText("the room refuses")).toBeInTheDocument();
        expect(onSaved).not.toHaveBeenCalled();
    });

    it("closes without saving anything when cancel is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onClose } = renderDialog(makeMember());

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
        expect(mocks.updateNickname).not.toHaveBeenCalled();
        expect(mocks.uploadAvatar).not.toHaveBeenCalled();
    });
});
