import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Poll, PollOption } from "../../../types/api";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { PollDisplay } from "./PollDisplay";

const { votePoll } = vi.hoisted(() => ({ votePoll: vi.fn() }));

vi.mock("../../../api/mutations/post", () => ({
    useVotePoll: () => ({ mutateAsync: votePoll }),
}));

const NOW = new Date("2026-08-02T12:00:00Z");

function makeOption(overrides: Partial<PollOption> = {}): PollOption {
    return {
        id: 0,
        label: "Beatrice",
        vote_count: 0,
        percent: 0,
        ...overrides,
    };
}

function makePoll(overrides: Partial<Poll> = {}): Poll {
    return {
        id: "poll-1",
        options: [makeOption({ id: 0, label: "Beatrice" }), makeOption({ id: 1, label: "Battler" })],
        total_votes: 0,
        user_voted_option: null,
        expired: false,
        expires_at: "2026-08-03T12:00:00Z",
        duration_seconds: 86400,
        ...overrides,
    };
}

function setup(poll: Poll, options: { signedIn?: boolean; onVoted?: () => void } = {}) {
    const user = options.signedIn === false ? null : makeUser();
    return renderWithProviders(<PollDisplay poll={poll} postId="post-1" onVoted={options.onVoted} />, { user });
}

describe("PollDisplay", () => {
    it("hides the results while the viewer still has a vote to cast", () => {
        // given
        const poll = makePoll({
            options: [
                makeOption({ id: 0, label: "Beatrice", percent: 80 }),
                makeOption({ id: 1, label: "Battler", percent: 20 }),
            ],
        });

        // when
        setup(poll);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.queryByText("80%")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Submit Vote" })).toBeDisabled();
    });

    it("keeps the submit control away from signed out viewers", () => {
        // given
        const poll = makePoll();

        // when
        setup(poll, { signedIn: false });

        // then
        expect(screen.queryByRole("button", { name: "Submit Vote" })).not.toBeInTheDocument();
    });

    it("enables submitting once an option is picked and disables it again when it is unpicked", async () => {
        // given
        const user = userEvent.setup();
        setup(makePoll());

        // when
        await user.click(screen.getByText("Battler"));

        // then
        expect(screen.getByRole("button", { name: "Submit Vote" })).toBeEnabled();
        await user.click(screen.getByText("Battler"));
        expect(screen.getByRole("button", { name: "Submit Vote" })).toBeDisabled();
    });

    it("sends the index of the chosen option and shows the returned tally", async () => {
        // given
        const onVoted = vi.fn();
        const user = userEvent.setup();
        votePoll.mockResolvedValue(
            makePoll({
                options: [
                    makeOption({ id: 0, label: "Beatrice", vote_count: 1, percent: 50 }),
                    makeOption({ id: 1, label: "Battler", vote_count: 1, percent: 50 }),
                ],
                total_votes: 2,
                user_voted_option: 1,
            }),
        );
        setup(makePoll(), { onVoted });

        // when
        await user.click(screen.getByText("Battler"));
        await user.click(screen.getByRole("button", { name: "Submit Vote" }));

        // then
        expect(votePoll).toHaveBeenCalledWith({ postId: "post-1", optionIdx: 1 });
        await waitFor(() => expect(screen.queryByRole("button", { name: "Submit Vote" })).not.toBeInTheDocument());
        expect(screen.getAllByText("50%")).toHaveLength(2);
        expect(screen.getByText("2 votes")).toBeInTheDocument();
        expect(onVoted).toHaveBeenCalledOnce();
    });

    it("leaves the vote controls in place when the vote is rejected", async () => {
        // given
        const user = userEvent.setup();
        votePoll.mockRejectedValue(new Error("the witch denies your vote"));
        setup(makePoll());

        // when
        await user.click(screen.getByText("Battler"));
        await user.click(screen.getByRole("button", { name: "Submit Vote" }));

        // then
        await waitFor(() => expect(screen.getByRole("button", { name: "Submit Vote" })).toBeEnabled());
        expect(screen.queryByText("0%")).not.toBeInTheDocument();
    });

    it("shows the results and no submit control once the viewer has voted", () => {
        // given
        const poll = makePoll({
            options: [
                makeOption({ id: 0, label: "Beatrice", vote_count: 1, percent: 33.333 }),
                makeOption({ id: 1, label: "Battler", vote_count: 2, percent: 66.667 }),
            ],
            total_votes: 3,
            user_voted_option: 1,
        });

        // when
        setup(poll);

        // then
        expect(screen.getByText("33%")).toBeInTheDocument();
        expect(screen.getByText("67%")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Submit Vote" })).not.toBeInTheDocument();
    });

    it("offers no way to change a vote that has already been cast", async () => {
        // given
        const user = userEvent.setup();
        setup(makePoll({ total_votes: 1, user_voted_option: 0, options: [makeOption({ id: 0, percent: 100 })] }));

        // when
        await user.click(screen.getByText("Beatrice"));

        // then
        expect(screen.queryByRole("button", { name: "Submit Vote" })).not.toBeInTheDocument();
        expect(votePoll).not.toHaveBeenCalled();
    });

    it("shows the results of an expired poll to a viewer who never voted", () => {
        // given
        const poll = makePoll({
            options: [
                makeOption({ id: 0, label: "Beatrice", vote_count: 4, percent: 100 }),
                makeOption({ id: 1, label: "Battler", vote_count: 0, percent: 0 }),
            ],
            total_votes: 4,
            expired: true,
            expires_at: "2026-08-01T12:00:00Z",
        });

        // when
        setup(poll);

        // then
        expect(screen.getByText("100%")).toBeInTheDocument();
        expect(screen.getByText("0%")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Submit Vote" })).not.toBeInTheDocument();
    });

    it("shows every option at zero when an expired poll drew no votes at all", () => {
        // given
        const poll = makePoll({
            options: [
                makeOption({ id: 0, label: "Beatrice" }),
                makeOption({ id: 1, label: "Battler" }),
                makeOption({ id: 2, label: "Erika" }),
            ],
            total_votes: 0,
            expired: true,
        });

        // when
        setup(poll);

        // then
        expect(screen.getAllByText("0%")).toHaveLength(3);
        expect(screen.getByText("0 votes")).toBeInTheDocument();
    });

    it("uses the singular label for a single vote", () => {
        // given
        const poll = makePoll({ total_votes: 1, user_voted_option: 0 });

        // when
        setup(poll);

        // then
        expect(screen.getByText("1 vote")).toBeInTheDocument();
    });

    it("counts down in minutes while less than an hour is left", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(NOW);

        // when
        setup(makePoll({ expires_at: "2026-08-02T12:45:00Z" }));

        // then
        expect(screen.getByText("45m remaining")).toBeInTheDocument();
    });

    it("counts down in hours while less than a day is left", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(NOW);

        // when
        setup(makePoll({ expires_at: "2026-08-02T20:30:00Z" }));

        // then
        expect(screen.getByText("8h remaining")).toBeInTheDocument();
    });

    it("counts down in days while more than a day is left", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(NOW);

        // when
        setup(makePoll({ expires_at: "2026-08-05T13:00:00Z" }));

        // then
        expect(screen.getByText("3d remaining")).toBeInTheDocument();
    });

    it("says the poll has ended once its deadline has passed", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(NOW);

        // when
        setup(makePoll({ expired: true, expires_at: "2026-08-01T12:00:00Z" }));

        // then
        expect(screen.getByText("Poll ended")).toBeInTheDocument();
    });
});
