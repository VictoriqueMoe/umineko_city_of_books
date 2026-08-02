import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { makeUser } from "../../../test-utils/fixtures";
import type { Response as TheoryResponse } from "../../../types/api";
import { ResponseList } from "./ResponseCard";

const { voteResponse, deleteResponse, createResponse } = vi.hoisted(() => ({
    voteResponse: vi.fn(() => Promise.resolve()),
    deleteResponse: vi.fn(() => Promise.resolve()),
    createResponse: vi.fn(() => Promise.resolve()),
}));

vi.mock("../../../api/mutations/theory", () => ({
    useVoteResponse: () => ({ mutateAsync: voteResponse }),
    useDeleteResponse: () => ({ mutateAsync: deleteResponse }),
    useCreateResponse: () => ({ mutateAsync: createResponse }),
}));

vi.mock("../../../hooks/useResolveQuotes", () => ({
    useResolveQuotes: () => new Map(),
}));

const author = makeUser({ id: "u-battler", username: "battler", display_name: "Battler" });
const stranger = makeUser({ id: "u-ronove", username: "ronove", display_name: "Ronove" });

function makeResponse(overrides: Partial<TheoryResponse> = {}): TheoryResponse {
    return {
        id: "r1",
        author,
        side: "with_love",
        body: "Kanon cannot be the culprit.",
        evidence: [],
        vote_score: 3,
        created_at: "2026-01-02T03:04:05Z",
        ...overrides,
    };
}

describe("ResponseList", () => {
    beforeEach(() => {
        voteResponse.mockResolvedValue(undefined);
        deleteResponse.mockResolvedValue(undefined);
    });

    it("renders every response it is given", () => {
        // given
        const responses = [makeResponse(), makeResponse({ id: "r2", body: "The stakes did it." })];

        // when
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />);

        // then
        expect(screen.getByText("Kanon cannot be the culprit.")).toBeInTheDocument();
        expect(screen.getByText("The stakes did it.")).toBeInTheDocument();
    });

    it("offers no controls at all to a signed out visitor", () => {
        // given
        const responses = [makeResponse()];

        // when
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />, { user: null });

        // then
        expect(screen.queryByRole("button", { name: "Reply" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("lets a signed in member reply to and report someone else's response", () => {
        // given
        const responses = [makeResponse()];

        // when
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />, { user: stranger });

        // then
        expect(screen.getByRole("button", { name: "Reply" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Report" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    });

    it("lets the author delete their own response but not report it", () => {
        // given
        const responses = [makeResponse()];

        // when
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />, { user: author });

        // then
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Report" })).not.toBeInTheDocument();
    });

    it("lets a moderator delete a response written by somebody else", () => {
        // given
        const moderator = makeUser({ id: "u-mod", username: "virgilia", display_name: "Virgilia", role: "moderator" });

        // when
        renderWithProviders(<ResponseList responses={[makeResponse()]} theoryId="t1" />, { user: moderator });

        // then
        expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    });

    it("keeps the response when the deletion is not confirmed", async () => {
        // given
        const onDeleted = vi.fn();
        const confirmed = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={[makeResponse()]} theoryId="t1" onDeleted={onDeleted} />, {
            user: author,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(confirmed).toHaveBeenCalledOnce();
        expect(deleteResponse).not.toHaveBeenCalled();
        expect(onDeleted).not.toHaveBeenCalled();
    });

    it("deletes the response and refreshes the thread once the deletion is confirmed", async () => {
        // given
        const onDeleted = vi.fn();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={[makeResponse()]} theoryId="t1" onDeleted={onDeleted} />, {
            user: author,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Delete" }));

        // then
        expect(deleteResponse).toHaveBeenCalledExactlyOnceWith("r1");
        expect(onDeleted).toHaveBeenCalledOnce();
    });

    it("sends an upvote for the response and shows the new score straight away", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={[makeResponse({ vote_score: 3 })]} theoryId="t1" />, {
            user: stranger,
        });

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));

        // then
        expect(voteResponse).toHaveBeenCalledExactlyOnceWith({ responseId: "r1", value: 1 });
        expect(screen.getByText("4")).toBeInTheDocument();
    });

    it("starts from the vote the reader has already cast", async () => {
        // given
        const responses = [makeResponse({ vote_score: 3, user_vote: 1 })];
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />, { user: stranger });

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));

        // then
        expect(voteResponse).toHaveBeenCalledExactlyOnceWith({ responseId: "r1", value: 0 });
        expect(screen.getByText("2")).toBeInTheDocument();
    });

    it("collapses a single reply behind a singular count", () => {
        // given
        const responses = [makeResponse({ replies: [makeResponse({ id: "c1", body: "But he was seen." })] })];

        // when
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />);

        // then
        expect(screen.getByRole("button", { name: "Show 1 reply" })).toBeInTheDocument();
        expect(screen.queryByText("But he was seen.")).not.toBeInTheDocument();
    });

    it("counts nested replies in the collapsed thread as well", () => {
        // given
        const nested = makeResponse({ id: "g1", body: "Only by Battler.", author: stranger });
        const responses = [
            makeResponse({
                replies: [makeResponse({ id: "c1", body: "But he was seen.", author: stranger, replies: [nested] })],
            }),
        ];

        // when
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />);

        // then
        expect(screen.getByRole("button", { name: "Show 2 replies" })).toBeInTheDocument();
    });

    it("shows the nested replies and who each one answers once the thread is expanded", async () => {
        // given
        const nested = makeResponse({ id: "g1", body: "Only by Battler.", author });
        const responses = [
            makeResponse({
                replies: [makeResponse({ id: "c1", body: "But he was seen.", author: stranger, replies: [nested] })],
            }),
        ];
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />);

        // when
        await user.click(screen.getByRole("button", { name: "Show 2 replies" }));

        // then
        expect(screen.getByText("But he was seen.")).toBeInTheDocument();
        expect(screen.getByText("Only by Battler.")).toBeInTheDocument();
        expect(screen.getByText("@Ronove")).toBeInTheDocument();
    });

    it("hides the thread again when it is collapsed", async () => {
        // given
        const responses = [makeResponse({ replies: [makeResponse({ id: "c1", body: "But he was seen." })] })];
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />);
        await user.click(screen.getByRole("button", { name: "Show 1 reply" }));

        // when
        await user.click(screen.getByRole("button", { name: "Hide replies" }));

        // then
        expect(screen.queryByText("But he was seen.")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Show 1 reply" })).toBeInTheDocument();
    });

    it("opens a reply editor under the response and closes it when reply is pressed again", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={[makeResponse()]} theoryId="t1" />, { user: stranger });

        // when
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // then
        expect(screen.getByRole("heading", { name: "Reply" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Write your reply...")).toBeInTheDocument();
        await user.click(screen.getAllByRole("button", { name: "Reply" })[0]);
        expect(screen.queryByRole("heading", { name: "Reply" })).not.toBeInTheDocument();
    });

    it("posts a reply to a thread reply on that reply's own side", async () => {
        // given
        const nested = makeResponse({ id: "c1", body: "But he was seen.", side: "without_love", author: stranger });
        const responses = [makeResponse({ side: "with_love", replies: [nested] })];
        const user = userEvent.setup();
        const { container } = renderWithProviders(<ResponseList responses={responses} theoryId="t1" />, {
            user: author,
        });
        await user.click(screen.getByRole("button", { name: "Show 1 reply" }));

        // when
        await user.click(screen.getAllByRole("button", { name: "Reply" })[1]);
        await user.type(screen.getByPlaceholderText("Write your reply..."), "Then who was it?");
        const submit = container.querySelector<HTMLButtonElement>('form button[type="submit"]');
        await user.click(submit as HTMLButtonElement);

        // then
        expect(createResponse).toHaveBeenCalledWith({
            parent_id: "c1",
            side: "without_love",
            body: "Then who was it?",
            evidence: [],
        });
    });

    it("opens only one reply editor at a time", async () => {
        // given
        const responses = [makeResponse(), makeResponse({ id: "r2", body: "The stakes did it." })];
        const user = userEvent.setup();
        renderWithProviders(<ResponseList responses={responses} theoryId="t1" />, { user: stranger });

        // when
        await user.click(screen.getAllByRole("button", { name: "Reply" })[1]);
        await user.click(screen.getAllByRole("button", { name: "Reply" })[0]);

        // then
        expect(screen.getAllByRole("heading", { name: "Reply" })).toHaveLength(1);
    });
});
