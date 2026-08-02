import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { useVote } from "../../../hooks/useVote";
import { VoteButton } from "./VoteButton";

function VoteHarness({ initialScore, voteFn }: { initialScore: number; voteFn: (value: number) => Promise<void> }) {
    const { score, userVote, vote } = useVote(initialScore, 0, voteFn);

    return <VoteButton score={score} userVote={userVote} onVote={vote} />;
}

describe("VoteButton", () => {
    it("shows the current score between the two arrows", () => {
        // given
        const score = 42;

        // when
        renderWithProviders(<VoteButton score={score} userVote={0} onVote={vi.fn()} />);

        // then
        expect(screen.getByText("42")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Upvote" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Downvote" })).toBeInTheDocument();
    });

    it("shows a negative score without any special handling", () => {
        // given
        const score = -3;

        // when
        renderWithProviders(<VoteButton score={score} userVote={-1} onVote={vi.fn()} />);

        // then
        expect(screen.getByText("-3")).toBeInTheDocument();
    });

    it("reports an upvote when the up arrow is pressed", async () => {
        // given
        const onVote = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<VoteButton score={1} userVote={0} onVote={onVote} />);

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));

        // then
        expect(onVote).toHaveBeenCalledExactlyOnceWith(1);
    });

    it("reports a downvote when the down arrow is pressed", async () => {
        // given
        const onVote = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<VoteButton score={1} userVote={0} onVote={onVote} />);

        // when
        await user.click(screen.getByRole("button", { name: "Downvote" }));

        // then
        expect(onVote).toHaveBeenCalledExactlyOnceWith(-1);
    });

    it("reports the same direction again when the arrow already chosen is pressed", async () => {
        // given
        const onVote = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<VoteButton score={1} userVote={1} onVote={onVote} />);

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));

        // then
        expect(onVote).toHaveBeenCalledExactlyOnceWith(1);
    });

    it("marks only the arrow the reader has chosen as active", () => {
        // given
        const userVote = 1;

        // when
        renderWithProviders(<VoteButton score={1} userVote={userVote} onVote={vi.fn()} />);

        // then
        const up = screen.getByRole("button", { name: "Upvote" }).getAttribute("class") ?? "";
        const down = screen.getByRole("button", { name: "Downvote" }).getAttribute("class") ?? "";
        expect(up).not.toBe(down);
    });

    it("raises the score as soon as an upvote is sent", async () => {
        // given
        const voteFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<VoteHarness initialScore={5} voteFn={voteFn} />);

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));

        // then
        expect(screen.getByText("6")).toBeInTheDocument();
    });

    it("cycles from up, to cleared, to down as the reader keeps pressing", async () => {
        // given
        const voteFn = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderWithProviders(<VoteHarness initialScore={5} voteFn={voteFn} />);

        // when
        await user.click(screen.getByRole("button", { name: "Upvote" }));
        await user.click(screen.getByRole("button", { name: "Upvote" }));
        await user.click(screen.getByRole("button", { name: "Downvote" }));

        // then
        expect(voteFn.mock.calls).toEqual([[1], [0], [-1]]);
        expect(screen.getByText("4")).toBeInTheDocument();
    });
});
