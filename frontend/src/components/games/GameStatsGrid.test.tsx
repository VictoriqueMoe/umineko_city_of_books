import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { GameStatsGrid, type StatsRow } from "./GameStatsGrid";

const rows: StatsRow[] = [
    { slot0: 12, label: "Moves", slot1: 11 },
    { slot0: 3, label: "Captures", slot1: 5 },
];

describe("GameStatsGrid", () => {
    it("names both players in the header", () => {
        // given
        const slot0Name = "Battler";
        const slot1Name = "Beatrice";

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name={slot0Name}
                slot1Name={slot1Name}
                isOver={false}
                rows={rows}
                totalLabel="Total moves"
                totalValue={23}
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("calls the figures live while the game is still going", () => {
        // given
        const isOver = false;

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver={isOver}
                rows={rows}
                totalLabel="Total moves"
                totalValue={23}
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
    });

    it("drops the live label once the game is over", () => {
        // given
        const isOver = true;

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver={isOver}
                rows={rows}
                totalLabel="Total moves"
                totalValue={23}
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.queryByText("Live stats")).not.toBeInTheDocument();
    });

    it("shows a labelled row for every statistic with a value on each side", () => {
        // given
        const supplied = rows;

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver={false}
                rows={supplied}
                totalLabel="Total moves"
                totalValue={23}
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.getByText("Moves")).toBeInTheDocument();
        expect(screen.getByText("Captures")).toBeInTheDocument();
        expect(screen.getByText("12")).toBeInTheDocument();
        expect(screen.getByText("11")).toBeInTheDocument();
        expect(screen.getByText("3")).toBeInTheDocument();
        expect(screen.getByText("5")).toBeInTheDocument();
    });

    it("renders text values as happily as numbers", () => {
        // given
        const supplied: StatsRow[] = [{ slot0: "e4", label: "Opening", slot1: "c5" }];

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver
                rows={supplied}
                totalLabel="Result"
                totalValue="Sicilian"
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.getByText("e4")).toBeInTheDocument();
        expect(screen.getByText("c5")).toBeInTheDocument();
        expect(screen.getByText("Result: Sicilian")).toBeInTheDocument();
    });

    it("summarises the total in the footer", () => {
        // given
        const totalValue = 23;

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver
                rows={rows}
                totalLabel="Total moves"
                totalValue={totalValue}
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.getByText("Total moves: 23")).toBeInTheDocument();
    });

    it("formats the duration in the footer", () => {
        // given
        const durationSeconds = 3725;

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver
                rows={rows}
                totalLabel="Total moves"
                totalValue={23}
                durationSeconds={durationSeconds}
            />,
        );

        // then
        expect(screen.getByText("Duration: 1h 2m")).toBeInTheDocument();
    });

    it("shows a dash for a duration that has not been measured", () => {
        // given
        const durationSeconds = 0;

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver
                rows={rows}
                totalLabel="Total moves"
                totalValue={23}
                durationSeconds={durationSeconds}
            />,
        );

        // then
        expect(screen.getByText("Duration: -")).toBeInTheDocument();
    });

    it("still renders the header and the footer when there is nothing to report", () => {
        // given
        const supplied: StatsRow[] = [];

        // when
        renderWithProviders(
            <GameStatsGrid
                slot0Name="Battler"
                slot1Name="Beatrice"
                isOver={false}
                rows={supplied}
                totalLabel="Total moves"
                totalValue={0}
                durationSeconds={0}
            />,
        );

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
        expect(screen.getByText("Total moves: 0")).toBeInTheDocument();
        expect(screen.queryByText("Moves")).not.toBeInTheDocument();
    });
});
