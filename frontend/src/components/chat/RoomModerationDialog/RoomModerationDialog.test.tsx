import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { BannedWordRule, ChatRoomBan, CreateBannedWordRequest, User } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { RoomModerationDialog } from "./RoomModerationDialog";

const { useChatRoomBans, useChatRoomBannedWords } = vi.hoisted(() => ({
    useChatRoomBans: vi.fn(),
    useChatRoomBannedWords: vi.fn(),
}));

const {
    useUnbanChatRoomMember,
    useCreateChatRoomBannedWord,
    useUpdateChatRoomBannedWord,
    useDeleteChatRoomBannedWord,
} = vi.hoisted(() => ({
    useUnbanChatRoomMember: vi.fn(),
    useCreateChatRoomBannedWord: vi.fn(),
    useUpdateChatRoomBannedWord: vi.fn(),
    useDeleteChatRoomBannedWord: vi.fn(),
}));

vi.mock("../../../api/queries/chat", () => ({ useChatRoomBans, useChatRoomBannedWords }));
vi.mock("../../../api/mutations/chat", () => ({
    useUnbanChatRoomMember,
    useCreateChatRoomBannedWord,
    useUpdateChatRoomBannedWord,
    useDeleteChatRoomBannedWord,
}));

const roomId = "room-1";
const patternPlaceholder = "Word or regex to block";

interface StubOptions {
    bans?: ChatRoomBan[];
    rules?: BannedWordRule[];
    loading?: boolean;
    unban?: (userId: string) => Promise<unknown>;
    create?: (req: CreateBannedWordRequest) => Promise<unknown>;
    update?: (vars: { ruleId: string; req: CreateBannedWordRequest }) => Promise<unknown>;
    remove?: (ruleId: string) => Promise<unknown>;
}

function makeMember(overrides: Partial<User> = {}): User {
    return {
        id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        ...overrides,
    };
}

function makeBan(overrides: Partial<ChatRoomBan> = {}): ChatRoomBan {
    return {
        user: makeMember(),
        reason: "",
        created_at: "2026-07-01T12:00:00Z",
        ...overrides,
    };
}

function makeRule(overrides: Partial<BannedWordRule> = {}): BannedWordRule {
    return {
        id: "rule-1",
        scope: "room",
        room_id: roomId,
        pattern: "kihihi",
        match_mode: "substring",
        case_sensitive: false,
        action: "delete",
        created_at: "2026-07-01T12:00:00Z",
        ...overrides,
    };
}

function stubModeration(options: StubOptions = {}) {
    const refreshBans = vi.fn(() => Promise.resolve());
    const refreshRules = vi.fn(() => Promise.resolve());
    useChatRoomBans.mockReturnValue({
        bans: options.bans ?? [],
        loading: options.loading ?? false,
        refresh: refreshBans,
    });
    useChatRoomBannedWords.mockReturnValue({
        rules: options.rules ?? [],
        loading: false,
        refresh: refreshRules,
    });

    const unban = vi.fn(options.unban ?? (() => Promise.resolve()));
    const create = vi.fn(options.create ?? (() => Promise.resolve()));
    const update = vi.fn(options.update ?? (() => Promise.resolve()));
    const remove = vi.fn(options.remove ?? (() => Promise.resolve()));
    useUnbanChatRoomMember.mockReturnValue({ mutateAsync: unban });
    useCreateChatRoomBannedWord.mockReturnValue({ mutateAsync: create });
    useUpdateChatRoomBannedWord.mockReturnValue({ mutateAsync: update });
    useDeleteChatRoomBannedWord.mockReturnValue({ mutateAsync: remove });

    return { refreshBans, refreshRules, unban, create, update, remove };
}

function renderDialog(isOpen = true) {
    const onClose = vi.fn();
    const result = renderWithProviders(<RoomModerationDialog isOpen={isOpen} roomId={roomId} onClose={onClose} />);

    return { ...result, onClose };
}

async function openWordsTab(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: /^Banned words/ }));
}

afterEach(() => {
    vi.restoreAllMocks();
});

describe("RoomModerationDialog", () => {
    it("renders nothing while the dialog is closed", () => {
        // given
        stubModeration();

        // when
        const { container } = renderDialog(false);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("leaves both moderation queries disabled while the dialog is closed", () => {
        // given
        stubModeration();

        // when
        renderDialog(false);

        // then
        expect(useChatRoomBans).toHaveBeenLastCalledWith(roomId, false);
        expect(useChatRoomBannedWords).toHaveBeenLastCalledWith(roomId, false);
    });

    it("counts the bans and the rules on their tabs", () => {
        // given
        stubModeration({
            bans: [makeBan({ user: makeMember({ id: "user-1" }) }), makeBan({ user: makeMember({ id: "user-2" }) })],
            rules: [makeRule()],
        });

        // when
        renderDialog();

        // then
        expect(screen.getByRole("button", { name: "Bans (2)" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Banned words (1)" })).toBeInTheDocument();
    });

    it("holds back both lists while either of them is still loading", () => {
        // given
        stubModeration({ bans: [makeBan()], loading: true });

        // when
        renderDialog();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Unban" })).not.toBeInTheDocument();
    });

    it("shows an empty state when nobody is banned from the room", () => {
        // given
        stubModeration({ bans: [] });

        // when
        renderDialog();

        // then
        expect(screen.getByText("No bans in this room.")).toBeInTheDocument();
    });

    it("shows the reason for a ban and who issued it", () => {
        // given
        stubModeration({
            bans: [
                makeBan({
                    reason: "endless spam",
                    banned_by: makeMember({ id: "user-9", username: "ronove", display_name: "Ronove" }),
                }),
            ],
        });

        // when
        renderDialog();

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText(/Reason: endless spam/)).toBeInTheDocument();
        expect(screen.getByText("Ronove")).toBeInTheDocument();
    });

    it("omits the reason line for a ban that was given without one", () => {
        // given
        stubModeration({ bans: [makeBan({ reason: "" })] });

        // when
        renderDialog();

        // then
        expect(screen.queryByText(/Reason:/)).not.toBeInTheDocument();
    });

    it("lifts a ban and then refreshes the ban list", async () => {
        // given
        const { unban, refreshBans } = stubModeration({ bans: [makeBan()] });
        const user = userEvent.setup();
        renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Unban" }));

        // then
        expect(unban).toHaveBeenCalledWith("user-1");
        await waitFor(() => {
            expect(refreshBans).toHaveBeenCalledOnce();
        });
    });

    it("reports the reason an unban was refused", async () => {
        // given
        stubModeration({ bans: [makeBan()], unban: () => Promise.reject(new Error("the ban is eternal")) });
        const user = userEvent.setup();
        renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "Unban" }));

        // then
        expect(await screen.findByText("the ban is eternal")).toBeInTheDocument();
    });

    it("marks only the row being unbanned as busy", async () => {
        // given
        let release: () => void = () => {};
        stubModeration({
            bans: [makeBan({ user: makeMember({ id: "user-1" }) }), makeBan({ user: makeMember({ id: "user-2" }) })],
            unban: () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        });
        const user = userEvent.setup();
        renderDialog();

        // when
        await user.click(screen.getAllByRole("button", { name: "Unban" })[0]);

        // then
        expect(screen.getByRole("button", { name: "..." })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Unban" })).toBeEnabled();
        await act(async () => {
            release();
        });
        expect(screen.getAllByRole("button", { name: "Unban" })).toHaveLength(2);
    });

    it("explains the scope of local and global rules on the banned words tab", async () => {
        // given
        stubModeration({ rules: [] });
        const user = userEvent.setup();
        renderDialog();

        // when
        await openWordsTab(user);

        // then
        expect(screen.getByText(/Hosts, site moderators, and admins are immune from all rules/)).toBeInTheDocument();
        expect(screen.getByText("No rules apply in this room.")).toBeInTheDocument();
    });

    it("keeps the add control unusable until a pattern is entered", async () => {
        // given
        stubModeration();
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);
        const addButton = () => screen.getByRole("button", { name: "Add rule" });
        const disabledWhileEmpty = (addButton() as HTMLButtonElement).disabled;

        // when
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "kihihi");

        // then
        expect(disabledWhileEmpty).toBe(true);
        expect(addButton()).toBeEnabled();
    });

    it("treats a pattern of only whitespace as no pattern at all", async () => {
        // given
        stubModeration();
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "   ");

        // then
        expect(screen.getByRole("button", { name: "Add rule" })).toBeDisabled();
    });

    it("refuses to save a pattern that is not a valid regular expression", async () => {
        // given
        stubModeration();
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.selectOptions(screen.getByLabelText("Mode"), "regex");
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "(unclosed");

        // then
        expect(screen.getByText(/^Regex error:/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Add rule" })).toBeDisabled();
    });

    it("accepts the same pattern as a plain substring rule", async () => {
        // given
        stubModeration();
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "(unclosed");

        // then
        expect(screen.queryByText(/^Regex error:/)).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Add rule" })).toBeEnabled();
    });

    it("creates a rule from the form and trims the pattern before sending it", async () => {
        // given
        const { create, refreshRules } = stubModeration();
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "  kihihi  ");
        await user.selectOptions(screen.getByLabelText("Mode"), "whole_word");
        await user.selectOptions(screen.getByLabelText("Action"), "kick");
        await user.click(screen.getByLabelText("Case sensitive"));
        await user.click(screen.getByRole("button", { name: "Add rule" }));

        // then
        expect(create).toHaveBeenCalledWith({
            pattern: "kihihi",
            match_mode: "whole_word",
            case_sensitive: true,
            action: "kick",
        });
        await waitFor(() => {
            expect(refreshRules).toHaveBeenCalledOnce();
        });
    });

    it("clears the form back to its defaults after a rule is created", async () => {
        // given
        stubModeration();
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "kihihi");
        await user.selectOptions(screen.getByLabelText("Action"), "kick");

        // when
        await user.click(screen.getByRole("button", { name: "Add rule" }));

        // then
        await waitFor(() => {
            expect(screen.getByPlaceholderText(patternPlaceholder)).toHaveValue("");
        });
        expect(screen.getByLabelText("Action")).toHaveValue("delete");
        expect(screen.getByLabelText("Mode")).toHaveValue("substring");
    });

    it("reports the reason a rule could not be saved", async () => {
        // given
        stubModeration({ create: () => Promise.reject(new Error("that word is sacred")) });
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "kihihi");

        // when
        await user.click(screen.getByRole("button", { name: "Add rule" }));

        // then
        expect(await screen.findByText("that word is sacred")).toBeInTheDocument();
        expect(screen.getByPlaceholderText(patternPlaceholder)).toHaveValue("kihihi");
    });

    it("shows global rules for awareness without any way to change them", async () => {
        // given
        stubModeration({
            rules: [
                makeRule({ id: "rule-1", scope: "room", pattern: "kihihi" }),
                makeRule({ id: "rule-2", scope: "global", room_id: undefined, pattern: "ushiromiya" }),
            ],
        });
        const user = userEvent.setup();
        renderDialog();

        // when
        await openWordsTab(user);

        // then
        expect(screen.getByText("ushiromiya")).toBeInTheDocument();
        expect(screen.getAllByRole("button", { name: "Edit" })).toHaveLength(1);
        expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(1);
    });

    it("shows the case sensitivity of a rule only when it is set", async () => {
        // given
        stubModeration({
            rules: [
                makeRule({ id: "rule-1", pattern: "kihihi", case_sensitive: true }),
                makeRule({ id: "rule-2", pattern: "uuu", case_sensitive: false }),
            ],
        });
        const user = userEvent.setup();
        renderDialog();

        // when
        await openWordsTab(user);

        // then
        expect(screen.getAllByText("case-sensitive")).toHaveLength(1);
    });

    it("loads a rule into the form when it is edited", async () => {
        // given
        stubModeration({
            rules: [makeRule({ pattern: "kihihi", match_mode: "whole_word", case_sensitive: true, action: "kick" })],
        });
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // then
        expect(screen.getByPlaceholderText(patternPlaceholder)).toHaveValue("kihihi");
        expect(screen.getByLabelText("Mode")).toHaveValue("whole_word");
        expect(screen.getByLabelText("Action")).toHaveValue("kick");
        expect(screen.getByLabelText("Case sensitive")).toBeChecked();
        expect(screen.getByRole("button", { name: "Save changes" })).toBeInTheDocument();
    });

    it("updates the edited rule rather than creating another one", async () => {
        // given
        const { create, update, refreshRules } = stubModeration({ rules: [makeRule({ id: "rule-7" })] });
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // when
        await user.clear(screen.getByPlaceholderText(patternPlaceholder));
        await user.type(screen.getByPlaceholderText(patternPlaceholder), "uuu");
        await user.click(screen.getByRole("button", { name: "Save changes" }));

        // then
        expect(update).toHaveBeenCalledWith({
            ruleId: "rule-7",
            req: { pattern: "uuu", match_mode: "substring", case_sensitive: false, action: "delete" },
        });
        expect(create).not.toHaveBeenCalled();
        await waitFor(() => {
            expect(refreshRules).toHaveBeenCalledOnce();
        });
    });

    it("abandons an edit and returns the form to adding a new rule", async () => {
        // given
        stubModeration({ rules: [makeRule({ pattern: "kihihi", action: "kick" })] });
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);
        await user.click(screen.getByRole("button", { name: "Edit" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.getByPlaceholderText(patternPlaceholder)).toHaveValue("");
        expect(screen.getByLabelText("Action")).toHaveValue("delete");
        expect(screen.getByRole("button", { name: "Add rule" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();
    });

    it("asks before removing a rule and leaves it alone when the prompt is dismissed", async () => {
        // given
        const { remove } = stubModeration({ rules: [makeRule()] });
        const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(confirmSpy).toHaveBeenCalledWith('Remove local rule for "kihihi"?');
        expect(remove).not.toHaveBeenCalled();
    });

    it("removes a local rule once the prompt is confirmed", async () => {
        // given
        const { remove, refreshRules } = stubModeration({ rules: [makeRule({ id: "rule-7" })] });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(remove).toHaveBeenCalledWith("rule-7");
        await waitFor(() => {
            expect(refreshRules).toHaveBeenCalledOnce();
        });
    });

    it("reports the reason a rule could not be removed", async () => {
        // given
        stubModeration({ rules: [makeRule()], remove: () => Promise.reject(new Error("global policy wins")) });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderDialog();
        await openWordsTab(user);

        // when
        await user.click(screen.getByRole("button", { name: "Remove" }));

        // then
        expect(await screen.findByText("global policy wins")).toBeInTheDocument();
    });

    it("closes when the dialog close control is pressed", async () => {
        // given
        stubModeration();
        const user = userEvent.setup();
        const { onClose } = renderDialog();

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });
});
