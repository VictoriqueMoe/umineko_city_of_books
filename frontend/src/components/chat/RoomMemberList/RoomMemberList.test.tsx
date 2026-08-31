import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { MemberModContext } from "../../../domain/chat/members";
import type { MemberGroup, PresenceMap } from "../../../domain/chat/memberRoster";
import { makePublicUser, makeRoomMember } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatRoomMember, User } from "../../../types/api";
import { RoomMemberList, type RoomMemberListVariant } from "./RoomMemberList";

const VIEWER_ID = "viewer-1";
const VARIANTS: RoomMemberListVariant[] = ["desktop", "mobile"];

function makeMemberUser(overrides: Partial<User> = {}): User {
    return makePublicUser({ id: "member-1", ...overrides });
}

function makeMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return makeRoomMember({ user: makeMemberUser(), ...overrides });
}

interface RosterOptions {
    variant?: RoomMemberListVariant;
    groups?: MemberGroup[];
    members?: ChatRoomMember[];
    presence?: PresenceMap;
    onlineIds?: string[];
    viewer?: Partial<MemberModContext>;
    openMemberMenu?: string | null;
    busy?: string | null;
}

function renderRoster(options: RosterOptions = {}) {
    const setOpenMemberMenu = vi.fn();
    const actions = {
        editSelf: vi.fn(),
        openNicknameDialog: vi.fn(),
        unlockNickname: vi.fn(),
        kick: vi.fn(),
        ban: vi.fn(),
        openTimeoutDialog: vi.fn(),
        clearTimeout: vi.fn(),
    };

    const members = options.members ?? [makeMember()];
    const groups = options.groups ?? [{ label: "Members", members }];
    const onlineIds = new Set(options.onlineIds ?? []);

    const result = renderWithProviders(
        <RoomMemberList
            variant={options.variant ?? "desktop"}
            groups={groups}
            presence={options.presence ?? {}}
            onlineWeight={id => (onlineIds.has(id) ? 0 : 1)}
            viewer={{
                selfId: VIEWER_ID,
                isSystem: false,
                isSiteMod: false,
                canModerateRoom: false,
                ...options.viewer,
            }}
            openMemberMenu={options.openMemberMenu ?? null}
            setOpenMemberMenu={setOpenMemberMenu}
            busy={options.busy ?? null}
            formatTimeoutUntil={value => `until ${value ?? "never"}`}
            actions={actions}
        />,
    );

    return { ...result, setOpenMemberMenu, actions };
}

describe("RoomMemberList status headers", () => {
    it("heads an online member with Online on the desktop roster", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoster({ variant: "desktop", members, onlineIds: ["member-1"] });

        // then
        expect(screen.getByText("Online")).toBeInTheDocument();
    });

    it("heads an absent member with Offline on the desktop roster", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoster({ variant: "desktop", members });

        // then
        expect(screen.getByText("Offline")).toBeInTheDocument();
    });

    it("skips the status heading inside the voice group", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoster({ variant: "desktop", groups: [{ label: "In Voice", members }] });

        // then
        expect(screen.getByText("In Voice")).toBeInTheDocument();
        expect(screen.queryByText("Offline")).not.toBeInTheDocument();
    });

    it("starts a new heading where the roster crosses from online to offline", () => {
        // given
        const members = [
            makeMember(),
            makeMember({ user: makeMemberUser({ id: "member-2", username: "ange", display_name: "Ange" }) }),
        ];

        // when
        renderRoster({ variant: "desktop", members, onlineIds: ["member-1"] });

        // then
        expect(screen.getByText("Online")).toBeInTheDocument();
        expect(screen.getByText("Offline")).toBeInTheDocument();
    });

    it("leaves the online and offline headings off the mobile roster", () => {
        // given
        const members = [makeMember()];

        // when
        renderRoster({ variant: "mobile", members, onlineIds: ["member-1"] });

        // then
        expect(screen.getByText("Members")).toBeInTheDocument();
        expect(screen.queryByText("Online")).not.toBeInTheDocument();
        expect(screen.queryByText("Offline")).not.toBeInTheDocument();
    });
});

for (const variant of VARIANTS) {
    describe(`RoomMemberList ${variant} roster`, () => {
        it("labels each group the way the caller named it", () => {
            // given
            const groups = [
                { label: "In Voice", members: [makeMember()] },
                { label: "Members", members: [makeMember({ user: makeMemberUser({ id: "member-2" }) })] },
            ];

            // when
            renderRoster({ variant, groups });

            // then
            expect(screen.getByText("In Voice")).toBeInTheDocument();
            expect(screen.getByText("Members")).toBeInTheDocument();
        });

        it("says whether a member is watching the room right now", () => {
            // given
            const presence: PresenceMap = { "member-1": "active" };

            // when
            renderRoster({ variant, presence });

            // then
            expect(screen.getByLabelText("Active in this room")).toBeInTheDocument();
        });

        it("says when a member has the tab in the background", () => {
            // given
            const presence: PresenceMap = { "member-1": "idle" };

            // when
            renderRoster({ variant, presence });

            // then
            expect(screen.getByLabelText("Idle or tab in background")).toBeInTheDocument();
        });

        it("says when a member is not viewing the room at all", () => {
            // given
            const presence: PresenceMap = {};

            // when
            renderRoster({ variant, presence });

            // then
            expect(screen.getByLabelText("Not currently viewing")).toBeInTheDocument();
        });

        it("badges the host", () => {
            // given
            const members = [makeMember({ role: "host" })];

            // when
            renderRoster({ variant, members });

            // then
            expect(screen.getByText("Host")).toBeInTheDocument();
        });

        it("badges a ghost member", () => {
            // given
            const members = [makeMember({ ghost: true })];

            // when
            renderRoster({ variant, members });

            // then
            expect(screen.getByTitle(/Ghost member/)).toBeInTheDocument();
        });

        it("marks a member who is timed out", () => {
            // given
            const members = [makeMember({ timeout_until: "2026-02-01T13:00:00Z" })];

            // when
            renderRoster({ variant, members });

            // then
            expect(screen.getByLabelText("Timed out until until 2026-02-01T13:00:00Z")).toBeInTheDocument();
        });

        it("shows a member's nickname in place of their display name", () => {
            // given
            const members = [makeMember({ nickname: "The Golden Witch" })];

            // when
            renderRoster({ variant, members });

            // then
            expect(screen.getByText("The Golden Witch")).toBeInTheDocument();
            expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
        });

        it("puts the room profile control on the viewer's own row alone", () => {
            // given
            const members = [makeMember({ user: makeMemberUser({ id: VIEWER_ID }) }), makeMember()];

            // when
            renderRoster({ variant, members });

            // then
            expect(screen.getAllByLabelText("Edit profile in this room")).toHaveLength(1);
        });

        it("lets the viewer edit their own profile in the room", async () => {
            // given
            const user = userEvent.setup();
            const members = [makeMember({ user: makeMemberUser({ id: VIEWER_ID }) })];
            const { actions } = renderRoster({ variant, members });

            // when
            await user.click(screen.getByRole("button", { name: "Edit profile in this room" }));

            // then
            expect(actions.editSelf).toHaveBeenCalledTimes(1);
        });

        it("gives an ordinary member no moderator actions", () => {
            // given
            const viewer = { canModerateRoom: false };

            // when
            renderRoster({ variant, viewer });

            // then
            expect(screen.queryByRole("button", { name: "Moderator actions" })).not.toBeInTheDocument();
        });

        it("gives a room moderator the actions over an ordinary member", () => {
            // given
            const viewer = { canModerateRoom: true };

            // when
            renderRoster({ variant, viewer });

            // then
            expect(screen.getByRole("button", { name: "Moderator actions" })).toBeInTheDocument();
        });

        it("leaves a system room unmoderated", () => {
            // given
            const viewer = { canModerateRoom: true, isSystem: true };

            // when
            renderRoster({ variant, viewer });

            // then
            expect(screen.queryByRole("button", { name: "Moderator actions" })).not.toBeInTheDocument();
        });

        it("toggles the moderator menu open and shut from the same control", async () => {
            // given
            const user = userEvent.setup();
            const { setOpenMemberMenu } = renderRoster({ variant, viewer: { canModerateRoom: true } });

            // when
            await user.click(screen.getByRole("button", { name: "Moderator actions" }));

            // then
            expect(setOpenMemberMenu).toHaveBeenCalledTimes(1);
            const toggle = setOpenMemberMenu.mock.calls[0][0] as (previous: string | null) => string | null;
            expect(toggle(null)).toBe("member-1");
            expect(toggle("member-1")).toBeNull();
        });

        it("offers a room moderator kick and ban but no nickname change", () => {
            // given
            const viewer = { canModerateRoom: true };

            // when
            renderRoster({ variant, viewer, openMemberMenu: "member-1" });

            // then
            expect(screen.getByRole("button", { name: "Kick member" })).toBeInTheDocument();
            expect(screen.getByRole("button", { name: "Ban from room" })).toBeInTheDocument();
            expect(screen.queryByRole("button", { name: "Change nickname" })).not.toBeInTheDocument();
        });

        it("offers site staff the nickname controls too", () => {
            // given
            const members = [makeMember({ nickname_locked: true })];

            // when
            renderRoster({ variant, members, viewer: { isSiteMod: true }, openMemberMenu: "member-1" });

            // then
            expect(screen.getByRole("button", { name: "Change nickname" })).toBeInTheDocument();
            expect(screen.getByRole("button", { name: "Reset/unlock nickname" })).toBeInTheDocument();
        });

        it("hides the unlock control while the nickname is not locked", () => {
            // given
            const members = [makeMember({ nickname_locked: false })];

            // when
            renderRoster({ variant, members, viewer: { isSiteMod: true }, openMemberMenu: "member-1" });

            // then
            expect(screen.queryByRole("button", { name: "Reset/unlock nickname" })).not.toBeInTheDocument();
        });

        it("opens the nickname dialog on the member the moderator chose", async () => {
            // given
            const user = userEvent.setup();
            const target = makeMember();
            const { actions } = renderRoster({
                variant,
                members: [target],
                viewer: { isSiteMod: true },
                openMemberMenu: "member-1",
            });

            // when
            await user.click(screen.getByRole("button", { name: "Change nickname" }));

            // then
            expect(actions.openNicknameDialog).toHaveBeenCalledWith(target);
        });

        it("unlocks a locked nickname", async () => {
            // given
            const user = userEvent.setup();
            const { actions } = renderRoster({
                variant,
                members: [makeMember({ nickname_locked: true })],
                viewer: { isSiteMod: true },
                openMemberMenu: "member-1",
            });

            // when
            await user.click(screen.getByRole("button", { name: "Reset/unlock nickname" }));

            // then
            expect(actions.unlockNickname).toHaveBeenCalledWith("member-1");
        });

        it("kicks a member and closes the menu behind it", async () => {
            // given
            const user = userEvent.setup();
            const { actions, setOpenMemberMenu } = renderRoster({
                variant,
                viewer: { canModerateRoom: true },
                openMemberMenu: "member-1",
            });

            // when
            await user.click(screen.getByRole("button", { name: "Kick member" }));

            // then
            expect(setOpenMemberMenu).toHaveBeenCalledWith(null);
            expect(actions.kick).toHaveBeenCalledWith("member-1");
        });

        it("bans a member and closes the menu behind it", async () => {
            // given
            const user = userEvent.setup();
            const { actions, setOpenMemberMenu } = renderRoster({
                variant,
                viewer: { canModerateRoom: true },
                openMemberMenu: "member-1",
            });

            // when
            await user.click(screen.getByRole("button", { name: "Ban from room" }));

            // then
            expect(setOpenMemberMenu).toHaveBeenCalledWith(null);
            expect(actions.ban).toHaveBeenCalledWith("member-1");
        });

        it("opens the timeout dialog on the member the moderator chose", async () => {
            // given
            const user = userEvent.setup();
            const target = makeMember();
            const { actions } = renderRoster({
                variant,
                members: [target],
                viewer: { canModerateRoom: true },
                openMemberMenu: "member-1",
            });

            // when
            await user.click(screen.getByRole("button", { name: "Set timeout" }));

            // then
            expect(actions.openTimeoutDialog).toHaveBeenCalledWith(target);
        });

        it("lifts an existing timeout", async () => {
            // given
            const user = userEvent.setup();
            const { actions } = renderRoster({
                variant,
                members: [makeMember({ timeout_until: "2026-09-01T00:00:00Z" })],
                viewer: { canModerateRoom: true },
                openMemberMenu: "member-1",
            });

            // when
            await user.click(screen.getByRole("button", { name: "Remove timeout" }));

            // then
            expect(actions.clearTimeout).toHaveBeenCalledWith("member-1");
        });

        it("holds the kick and ban controls shut while that member's request is in flight", () => {
            // given
            const busy = "member-1";

            // when
            renderRoster({ variant, viewer: { canModerateRoom: true }, openMemberMenu: "member-1", busy });

            // then
            expect(screen.getByRole("button", { name: "Kick member" })).toBeDisabled();
            expect(screen.getByRole("button", { name: "Ban from room" })).toBeDisabled();
        });

        it("holds the remove timeout control shut while the clearing is in flight", () => {
            // given
            const busy = "timeout:member-1";

            // when
            renderRoster({
                variant,
                members: [makeMember({ timeout_until: "2026-09-01T00:00:00Z" })],
                viewer: { canModerateRoom: true },
                openMemberMenu: "member-1",
                busy,
            });

            // then
            expect(screen.getByRole("button", { name: "Remove timeout" })).toBeDisabled();
        });

        it("keeps the moderator menu shut for every member but the open one", () => {
            // given
            const members = [
                makeMember(),
                makeMember({ user: makeMemberUser({ id: "member-2", display_name: "Ange" }) }),
            ];

            // when
            renderRoster({ variant, members, viewer: { canModerateRoom: true }, openMemberMenu: "member-1" });

            // then
            expect(screen.getAllByRole("button", { name: "Moderator actions" })).toHaveLength(2);
            expect(screen.getAllByRole("button", { name: "Kick member" })).toHaveLength(1);
        });
    });
}
