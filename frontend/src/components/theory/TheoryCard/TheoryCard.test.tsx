import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { useLocation } from "react-router";
import { renderWithProviders } from "../../../test-utils/render";
import { makeUser } from "../../../test-utils/fixtures";
import type { Theory } from "../../../types/api";
import { TheoryCard } from "./TheoryCard";

function makeTheory(overrides: Partial<Theory> = {}): Theory {
    return {
        id: "t1",
        title: "The culprit is Kanon",
        body: "Kanon cannot be seen without love.",
        episode: 5,
        series: "umineko",
        author: makeUser({ display_name: "Beatrice", username: "beatrice" }),
        vote_score: 12,
        with_love_count: 7,
        without_love_count: 3,
        credibility_score: 81,
        created_at: "2026-01-02T03:04:05Z",
        ...overrides,
    };
}

function PathProbe() {
    const location = useLocation();

    return <span>path: {location.pathname}</span>;
}

describe("TheoryCard", () => {
    it("renders the theory, its author and the tallies", () => {
        // given
        const theory = makeTheory();

        // when
        renderWithProviders(<TheoryCard theory={theory} />);

        // then
        expect(screen.getByRole("heading", { name: "The culprit is Kanon" })).toBeInTheDocument();
        expect(screen.getByText("Kanon cannot be seen without love.")).toBeInTheDocument();
        expect(screen.getByText("'s Blue Truth")).toBeInTheDocument();
        expect(screen.getByText("12 votes")).toBeInTheDocument();
        expect(screen.getByText("❤ 7")).toBeInTheDocument();
        expect(screen.getByText("✘ 3")).toBeInTheDocument();
        expect(screen.getByText("Credibility")).toBeInTheDocument();
        expect(screen.getByText("81")).toBeInTheDocument();
    });

    it("links the whole card to the theory it describes", () => {
        // given
        const theory = makeTheory({ id: "abc-123" });

        // when
        renderWithProviders(<TheoryCard theory={theory} />);

        // then
        expect(screen.getByRole("link", { name: "The culprit is Kanon" })).toHaveAttribute("href", "/theory/abc-123");
    });

    it("labels a theory that belongs to a particular episode", () => {
        // given
        const theory = makeTheory({ episode: 5 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />);

        // then
        expect(screen.getByText("Episode 5")).toBeInTheDocument();
    });

    it("leaves a general theory without an episode label", () => {
        // given
        const theory = makeTheory({ episode: 0 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />);

        // then
        expect(screen.queryByText(/^Episode/)).not.toBeInTheDocument();
    });

    it("hides a theory the reader has not caught up with behind a spoiler", () => {
        // given
        const theory = makeTheory({ episode: 5 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />, { user: makeUser({ episode_progress: 5 }) });

        // then
        expect(screen.getByText("Spoiler: Episode 5")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Show anyway" })).toBeInTheDocument();
    });

    it("does not spoil a reader who is already past the episode", () => {
        // given
        const theory = makeTheory({ episode: 5 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />, { user: makeUser({ episode_progress: 6 }) });

        // then
        expect(screen.queryByText(/^Spoiler:/)).not.toBeInTheDocument();
    });

    it("does not spoil a signed out visitor", () => {
        // given
        const theory = makeTheory({ episode: 8 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />, { user: null });

        // then
        expect(screen.queryByText(/^Spoiler:/)).not.toBeInTheDocument();
    });

    it("does not spoil a theory that is not tied to an episode", () => {
        // given
        const theory = makeTheory({ episode: 0 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />, { user: makeUser({ episode_progress: 1 }) });

        // then
        expect(screen.queryByText(/^Spoiler:/)).not.toBeInTheDocument();
    });

    it("reveals the theory when the reader asks to see it anyway", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<TheoryCard theory={makeTheory({ episode: 5 })} />, {
            user: makeUser({ episode_progress: 3 }),
        });

        // when
        await user.click(screen.getByRole("button", { name: "Show anyway" }));

        // then
        expect(screen.queryByText(/^Spoiler:/)).not.toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "The culprit is Kanon" })).toBeInTheDocument();
    });

    it("keeps the card link inert until the spoiler has been revealed", async () => {
        // given
        const clicker = userEvent.setup();
        renderWithProviders(
            <>
                <TheoryCard theory={makeTheory({ episode: 5 })} />
                <PathProbe />
            </>,
            { user: makeUser({ episode_progress: 3 }) },
        );

        // when
        await clicker.click(screen.getByRole("link", { name: "The culprit is Kanon" }));

        // then
        expect(screen.getByText("path: /")).toBeInTheDocument();
        await clicker.click(screen.getByRole("button", { name: "Show anyway" }));
        await clicker.click(screen.getByRole("link", { name: "The culprit is Kanon" }));
        expect(screen.getByText("path: /theory/t1")).toBeInTheDocument();
    });

    it("names a higurashi arc instead of numbering an episode", () => {
        // given
        const theory = makeTheory({ series: "higurashi", episode: 2 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />, { user: makeUser({ higurashi_arc_progress: 1 }) });

        // then
        expect(screen.getByText("Spoiler: Watanagashi")).toBeInTheDocument();
        expect(screen.getByText("Watanagashi")).toBeInTheDocument();
    });

    it("measures higurashi spoilers against the arc progress rather than the episode progress", () => {
        // given
        const theory = makeTheory({ series: "higurashi", episode: 2 });

        // when
        renderWithProviders(<TheoryCard theory={theory} />, {
            user: makeUser({ episode_progress: 1, higurashi_arc_progress: 3 }),
        });

        // then
        expect(screen.queryByText(/^Spoiler:/)).not.toBeInTheDocument();
    });
});
