import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { MysteryAttempt } from "../../types/api";
import { AttemptItem } from "./AttemptItem";

const { useCreateMysteryAttempt, useDeleteMysteryAttempt, useMarkMysterySolved, useVoteMysteryAttempt } = vi.hoisted(
    () => ({
        useCreateMysteryAttempt: vi.fn(),
        useDeleteMysteryAttempt: vi.fn(),
        useMarkMysterySolved: vi.fn(),
        useVoteMysteryAttempt: vi.fn(),
    }),
);

vi.mock("../../api/mutations/mystery", () => ({
    useCreateMysteryAttempt,
    useDeleteMysteryAttempt,
    useMarkMysterySolved,
    useVoteMysteryAttempt,
}));

const player = { id: "player-1", username: "battler", display_name: "Battler" };
const gameMaster = { id: "gm-1", username: "beatrice", display_name: "Beatrice" };
const playerUser = makeUser({ id: "player-1", username: "battler", display_name: "Battler" });
const gameMasterUser = makeUser({ id: "gm-1", username: "beatrice", display_name: "Beatrice" });
const bystander = makeUser({ id: "other-1", username: "ange", display_name: "Ange" });

function makeAttempt(overrides: Partial<MysteryAttempt> = {}): MysteryAttempt {
    return {
        id: "attempt-1",
        author: player,
        body: "The chain was fixed to the door after the fact.",
        is_winner: false,
        vote_score: 2,
        user_vote: 0,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    vote?: () => Promise<unknown>;
    reply?: () => Promise<unknown>;
}

function stubMutations(options: StubOptions = {}) {
    const voteAsync = vi.fn(options.vote ?? (() => Promise.resolve({})));
    const replyAsync = vi.fn(options.reply ?? (() => Promise.resolve({})));
    const deleteAsync = vi.fn(() => Promise.resolve({}));
    const markSolvedAsync = vi.fn(() => Promise.resolve({}));
    useVoteMysteryAttempt.mockReturnValue({ mutateAsync: voteAsync });
    useCreateMysteryAttempt.mockReturnValue({ mutateAsync: replyAsync });
    useDeleteMysteryAttempt.mockReturnValue({ mutateAsync: deleteAsync });
    useMarkMysterySolved.mockReturnValue({ mutateAsync: markSolvedAsync });

    return { voteAsync, replyAsync, deleteAsync, markSolvedAsync };
}

interface RenderOptions {
    attempt?: MysteryAttempt;
    viewer?: ReturnType<typeof makeUser> | null;
    isAuthor?: boolean;
    mysterySolved?: boolean;
    mysteryPaused?: boolean;
    authorAlreadyWon?: boolean;
}

function renderAttempt(options: RenderOptions = {}) {
    const onRefresh = vi.fn();
    const result = renderWithProviders(
        <AttemptItem
            attempt={options.attempt ?? makeAttempt()}
            mysteryId="mystery-1"
            isAuthor={options.isAuthor ?? false}
            onRefresh={onRefresh}
            mysterySolved={options.mysterySolved ?? false}
            mysteryPaused={options.mysteryPaused ?? false}
            authorAlreadyWon={options.authorAlreadyWon ?? false}
        />,
        { user: options.viewer ?? null, route: "/mystery/mystery-1" },
    );

    return { ...result, onRefresh };
}

describe("AttemptItem", () => {
    it("shows the attempt body and its author", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: bystander });

        // then
        expect(screen.getByText("The chain was fixed to the door after the fact.")).toBeInTheDocument();
        expect(screen.getAllByText("Battler").length).toBeGreaterThan(0);
        expect(screen.queryByText("Winner")).not.toBeInTheDocument();
    });

    it("badges the winning attempt", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ attempt: makeAttempt({ is_winner: true }), viewer: bystander });

        // then
        expect(screen.getByText("Winner")).toBeInTheDocument();
    });

    it("shows no thread controls when nobody has replied", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: bystander });

        // then
        expect(screen.queryByRole("button", { name: /repl(y|ies)$/ })).not.toBeInTheDocument();
    });

    it("flattens a nested thread and names who each reply answers", () => {
        // given
        stubMutations();
        const attempt = makeAttempt({
            replies: [
                {
                    ...makeAttempt({ id: "reply-1", author: gameMaster, body: "Denied in red." }),
                    replies: [makeAttempt({ id: "reply-2", body: "Then the window." })],
                },
            ],
        });

        // when
        renderAttempt({ attempt, viewer: bystander });

        // then
        expect(screen.getByText("Denied in red.")).toBeInTheDocument();
        expect(screen.getByText("Then the window.")).toBeInTheDocument();
        expect(screen.getByText("@Battler")).toBeInTheDocument();
        expect(screen.getByText("@Beatrice")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Hide 2 replies" })).toBeInTheDocument();
    });

    it("folds the thread away and offers to show it again", async () => {
        // given
        stubMutations();
        const attempt = makeAttempt({ replies: [makeAttempt({ id: "reply-1", body: "Denied in red." })] });
        const user = userEvent.setup();
        renderAttempt({ attempt, viewer: bystander });

        // when
        await user.click(screen.getByRole("button", { name: "Hide 1 reply" }));

        // then
        expect(screen.queryByText("Denied in red.")).not.toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Show 1 reply" }));
        expect(screen.getByText("Denied in red.")).toBeInTheDocument();
    });

    it("gives a signed out visitor nothing but the share link", () => {
        // given
        stubMutations();

        // when
        renderAttempt();

        // then
        expect(screen.getByRole("button", { name: "Copy Link" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /▲|△/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("copies a canonical link straight to the attempt", async () => {
        // given
        stubMutations();
        const writeText = vi.fn(() => Promise.resolve());
        vi.spyOn(navigator.clipboard, "writeText").mockImplementation(writeText);
        const user = userEvent.setup();
        renderAttempt({ viewer: bystander });

        // when
        await user.click(screen.getByRole("button", { name: "Copy Link" }));

        // then
        expect(writeText).toHaveBeenCalledWith("https://whentheycry.social/mystery/mystery-1#attempt-attempt-1");
    });

    it("records an upvote and moves the score straight away", async () => {
        // given
        const { voteAsync } = stubMutations();
        const user = userEvent.setup();
        renderAttempt({ viewer: bystander });

        // when
        await user.click(screen.getByRole("button", { name: /△/ }));

        // then
        expect(voteAsync).toHaveBeenCalledWith({ id: "attempt-1", value: 1 });
        expect(screen.getByRole("button", { name: /▲ 3/ })).toBeInTheDocument();
    });

    it("takes an existing vote back when the same arrow is pressed twice", async () => {
        // given
        const { voteAsync } = stubMutations();
        const user = userEvent.setup();
        renderAttempt({ attempt: makeAttempt({ user_vote: 1, vote_score: 3 }), viewer: bystander });

        // when
        await user.click(screen.getByRole("button", { name: /▲/ }));

        // then
        expect(voteAsync).toHaveBeenCalledWith({ id: "attempt-1", value: 0 });
        expect(screen.getByRole("button", { name: /△ 2/ })).toBeInTheDocument();
    });

    it("re-syncs the score when a refreshed attempt arrives from the server", async () => {
        // given
        const { voteAsync } = stubMutations();
        const user = userEvent.setup();
        const { rerender } = renderAttempt({ viewer: bystander });
        await user.click(screen.getByRole("button", { name: /△/ }));
        expect(voteAsync).toHaveBeenCalledWith({ id: "attempt-1", value: 1 });

        // when
        rerender(
            <AttemptItem
                attempt={makeAttempt({ vote_score: 9, user_vote: 1 })}
                mysteryId="mystery-1"
                isAuthor={false}
                onRefresh={vi.fn()}
                mysterySolved={false}
                mysteryPaused={false}
                authorAlreadyWon={false}
            />,
        );

        // then
        expect(screen.getByRole("button", { name: /▲ 9/ })).toBeInTheDocument();
    });

    it("puts the score back when the vote is rejected", async () => {
        // given
        stubMutations({ vote: () => Promise.reject(new Error("no")) });
        const user = userEvent.setup();
        renderAttempt({ viewer: bystander });

        // when
        await user.click(screen.getByRole("button", { name: /▽/ }));

        // then
        await waitFor(() => {
            expect(screen.getByRole("button", { name: /△ 2/ })).toBeInTheDocument();
        });
    });

    it("keeps the reply button away from a piece who is neither the owner nor the game master", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: bystander });

        // then
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
    });

    it("lets the owner of the attempt carry on the thread", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: playerUser });

        // then
        expect(screen.getByRole("button", { name: "Reply" })).toBeInTheDocument();
    });

    it("closes the thread to everyone once the mystery is solved", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: gameMasterUser, isAuthor: true, mysterySolved: true });

        // then
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
    });

    it("silences the pieces while the mystery is paused but still lets the game master speak", () => {
        // given
        stubMutations();

        // when
        const { unmount } = renderAttempt({ viewer: playerUser, mysteryPaused: true });

        // then
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
        unmount();
        renderAttempt({ viewer: gameMasterUser, isAuthor: true, mysteryPaused: true });
        expect(screen.getByRole("button", { name: "Reply" })).toBeInTheDocument();
    });

    it("posts a trimmed reply against the attempt and refreshes the board", async () => {
        // given
        const { replyAsync } = stubMutations();
        const user = userEvent.setup();
        const { onRefresh } = renderAttempt({ viewer: playerUser });
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // when
        await user.type(screen.getByPlaceholderText("Reply..."), "  The window was never shut.  ");
        await user.click(screen.getAllByRole("button", { name: "Reply" })[1]);

        // then
        expect(replyAsync).toHaveBeenCalledWith({ body: "The window was never shut.", parentId: "attempt-1" });
        await waitFor(() => {
            expect(onRefresh).toHaveBeenCalledOnce();
        });
        expect(screen.queryByPlaceholderText("Reply...")).not.toBeInTheDocument();
    });

    it("refuses to send an empty reply", async () => {
        // given
        const { replyAsync } = stubMutations();
        const user = userEvent.setup();
        renderAttempt({ viewer: playerUser });

        // when
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // then
        expect(screen.getAllByRole("button", { name: "Reply" })[1]).toBeDisabled();
        expect(replyAsync).not.toHaveBeenCalled();
    });

    it("abandons the composer when the reply is cancelled", async () => {
        // given
        stubMutations();
        const user = userEvent.setup();
        renderAttempt({ viewer: playerUser });

        // when
        await user.click(screen.getByRole("button", { name: "Reply" }));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(screen.queryByPlaceholderText("Reply...")).not.toBeInTheDocument();
    });

    it("offers the winner's laurels only to the game master judging somebody else", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: gameMasterUser, isAuthor: true });

        // then
        expect(screen.getByRole("button", { name: "Select Winner" })).toBeInTheDocument();
    });

    it("withholds the winner's laurels from a piece and from an already crowned author", () => {
        // given
        stubMutations();

        // when
        const { unmount } = renderAttempt({ viewer: playerUser });

        // then
        expect(screen.queryByRole("button", { name: "Select Winner" })).not.toBeInTheDocument();
        unmount();
        renderAttempt({ viewer: gameMasterUser, isAuthor: true, authorAlreadyWon: true });
        expect(screen.queryByRole("button", { name: "Select Winner" })).not.toBeInTheDocument();
    });

    it("withholds the winner's laurels once the mystery is solved", () => {
        // given
        stubMutations();

        // when
        renderAttempt({ viewer: gameMasterUser, isAuthor: true, mysterySolved: true });

        // then
        expect(screen.queryByRole("button", { name: "Select Winner" })).not.toBeInTheDocument();
    });

    it("asks the game master to confirm before crowning a winner", async () => {
        // given
        const { markSolvedAsync } = stubMutations();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderAttempt({ viewer: gameMasterUser, isAuthor: true });

        // when
        await user.click(screen.getByRole("button", { name: "Select Winner" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Select this attempt by Battler as the winner?");
        expect(markSolvedAsync).not.toHaveBeenCalled();
    });

    it("crowns the attempt and refreshes the board once confirmed", async () => {
        // given
        const { markSolvedAsync } = stubMutations();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        const { onRefresh } = renderAttempt({ viewer: gameMasterUser, isAuthor: true });

        // when
        await user.click(screen.getByRole("button", { name: "Select Winner" }));

        // then
        expect(markSolvedAsync).toHaveBeenCalledWith("attempt-1");
        await waitFor(() => {
            expect(onRefresh).toHaveBeenCalledOnce();
        });
    });

    it("lets the owner and a moderator delete the attempt but nobody else", () => {
        // given
        stubMutations();

        // when
        const { unmount } = renderAttempt({ viewer: playerUser });

        // then
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
        unmount();
        const moderator = renderAttempt({ viewer: makeUser({ id: "mod-1", role: "moderator" }) });
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
        moderator.unmount();
        renderAttempt({ viewer: bystander });
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("asks before deleting and refreshes the board afterwards", async () => {
        // given
        const { deleteAsync } = stubMutations();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        const { onRefresh } = renderAttempt({ viewer: playerUser });

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Delete this attempt?");
        expect(deleteAsync).toHaveBeenCalledWith("attempt-1");
        await waitFor(() => {
            expect(onRefresh).toHaveBeenCalledOnce();
        });
    });

    it("keeps the attempt when the deletion is waved away", async () => {
        // given
        const { deleteAsync } = stubMutations();
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderAttempt({ viewer: playerUser });

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(deleteAsync).not.toHaveBeenCalled();
    });

    it("lets a piece report somebody else's attempt but not their own", () => {
        // given
        stubMutations();

        // when
        const { unmount } = renderAttempt({ viewer: bystander });

        // then
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
        unmount();
        renderAttempt({ viewer: playerUser });
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
    });
});
