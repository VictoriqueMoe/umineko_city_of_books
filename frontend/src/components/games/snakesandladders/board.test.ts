import { describe, expect, it } from "vitest";
import {
    buildLadder,
    buildSnake,
    CELL,
    cellCenter,
    cellTopLeft,
    GRID,
    LADDERS,
    type Point,
    SNAKES,
    VIEW,
} from "./board";

function distance(a: Point, b: Point): number {
    return Math.hypot(b.x - a.x, b.y - a.y);
}

function midpoint(a: Point, b: Point): Point {
    return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
}

function expectPoint(actual: Point, x: number, y: number): void {
    expect(actual.x).toBeCloseTo(x, 6);
    expect(actual.y).toBeCloseTo(y, 6);
}

function numbersIn(path: string): number[] {
    const matches = path.match(/-?\d+\.\d+/g) ?? [];
    return matches.map(Number);
}

describe("snakes and ladders board metrics", () => {
    it("measures the board as ten squares of a hundred units", () => {
        // given
        const expectedView = 1000;

        // when
        const view = VIEW;

        // then
        expect(GRID).toBe(10);
        expect(CELL).toBe(100);
        expect(view).toBe(expectedView);
    });
});

describe("LADDERS", () => {
    it("holds every ladder on the board", () => {
        // given
        const board = LADDERS;

        // when
        const feet = Object.keys(board).map(Number);

        // then
        expect(board).toEqual({ 1: 38, 4: 14, 9: 31, 21: 42, 28: 84, 36: 44, 51: 67, 71: 91, 80: 100 });
        expect(feet).toHaveLength(9);
    });

    it("only ever climbs upwards and stays on the board", () => {
        // given
        const entries = Object.entries(LADDERS);

        // when
        const climbs = entries.map(([from, to]) => to - Number(from));

        // then
        for (const climb of climbs) {
            expect(climb).toBeGreaterThan(0);
        }
        for (const [from, to] of entries) {
            expect(Number(from)).toBeGreaterThanOrEqual(1);
            expect(to).toBeLessThanOrEqual(GRID * GRID);
        }
    });
});

describe("SNAKES", () => {
    it("holds every snake on the board", () => {
        // given
        const board = SNAKES;

        // when
        const heads = Object.keys(board).map(Number);

        // then
        expect(board).toEqual({ 16: 6, 47: 26, 49: 11, 56: 53, 62: 19, 64: 60, 87: 24, 93: 73, 95: 75, 98: 78 });
        expect(heads).toHaveLength(10);
    });

    it("only ever slides downwards and stays on the board", () => {
        // given
        const entries = Object.entries(SNAKES);

        // when
        const drops = entries.map(([from, to]) => Number(from) - to);

        // then
        for (const drop of drops) {
            expect(drop).toBeGreaterThan(0);
        }
        for (const [, to] of entries) {
            expect(to).toBeGreaterThanOrEqual(1);
        }
    });

    it("never shares a square with a ladder", () => {
        // given
        const ladderFeet = new Set(Object.keys(LADDERS));

        // when
        const shared = Object.keys(SNAKES).filter(head => ladderFeet.has(head));

        // then
        expect(shared).toEqual([]);
    });

    it("never chains a ladder straight into a snake or the other way round", () => {
        // given
        const snakeHeads = new Set(Object.keys(SNAKES).map(Number));
        const ladderFeet = new Set(Object.keys(LADDERS).map(Number));

        // when
        const ladderTops = Object.values(LADDERS);
        const snakeTails = Object.values(SNAKES);

        // then
        for (const top of ladderTops) {
            expect(snakeHeads.has(top)).toBe(false);
        }
        for (const tail of snakeTails) {
            expect(ladderFeet.has(tail)).toBe(false);
        }
    });
});

describe("cellCenter", () => {
    it("puts the first square in the bottom left corner", () => {
        // given
        const square = 1;

        // when
        const centre = cellCenter(square);

        // then
        expectPoint(centre, 50, 950);
    });

    it("runs the bottom row from left to right", () => {
        // given
        const row = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

        // when
        const centres = row.map(cellCenter);

        // then
        expectPoint(centres[0], 50, 950);
        expectPoint(centres[9], 950, 950);
        for (let i = 1; i < centres.length; i++) {
            expect(centres[i].x).toBeCloseTo(centres[i - 1].x + CELL, 6);
            expect(centres[i].y).toBeCloseTo(centres[i - 1].y, 6);
        }
    });

    it("turns back on itself and runs the second row from right to left", () => {
        // given
        const row = [11, 12, 13, 14, 15, 16, 17, 18, 19, 20];

        // when
        const centres = row.map(cellCenter);

        // then
        expectPoint(centres[0], 950, 850);
        expectPoint(centres[9], 50, 850);
        for (let i = 1; i < centres.length; i++) {
            expect(centres[i].x).toBeCloseTo(centres[i - 1].x - CELL, 6);
        }
    });

    it("keeps the tenth and eleventh squares in the same column so the path is continuous", () => {
        // given
        const turn = [10, 11];

        // when
        const [ten, eleven] = turn.map(cellCenter);

        // then
        expect(eleven.x).toBeCloseTo(ten.x, 6);
        expect(ten.y - eleven.y).toBeCloseTo(CELL, 6);
    });

    it("puts the hundredth square in the top left corner", () => {
        // given
        const square = 100;

        // when
        const centre = cellCenter(square);

        // then
        expectPoint(centre, 50, 50);
    });

    it("puts the ninety first square in the top right corner", () => {
        // given
        const square = 91;

        // when
        const centre = cellCenter(square);

        // then
        expectPoint(centre, 950, 50);
    });

    it("climbs exactly one row for every ten squares", () => {
        // given
        const squares: number[] = [];
        for (let n = 1; n <= 90; n++) {
            squares.push(n);
        }

        // when
        const rises = squares.map(n => cellCenter(n).y - cellCenter(n + GRID).y);

        // then
        for (const rise of rises) {
            expect(rise).toBeCloseTo(CELL, 6);
        }
    });

    it("gives every square its own place inside the board", () => {
        // given
        const seen = new Set<string>();

        // when
        for (let n = 1; n <= GRID * GRID; n++) {
            const centre = cellCenter(n);
            seen.add(`${centre.x},${centre.y}`);
            expect(centre.x).toBeGreaterThan(0);
            expect(centre.x).toBeLessThan(VIEW);
            expect(centre.y).toBeGreaterThan(0);
            expect(centre.y).toBeLessThan(VIEW);
        }

        // then
        expect(seen.size).toBe(GRID * GRID);
    });
});

describe("cellTopLeft", () => {
    it("anchors the first square at the bottom left of the board", () => {
        // given
        const square = 1;

        // when
        const corner = cellTopLeft(square);

        // then
        expectPoint(corner, 0, 900);
    });

    it("anchors the hundredth square at the very top left", () => {
        // given
        const square = 100;

        // when
        const corner = cellTopLeft(square);

        // then
        expectPoint(corner, 0, 0);
    });

    it("sits half a square above and to the left of the centre for every square", () => {
        // given
        const half = CELL / 2;

        // when
        for (let n = 1; n <= GRID * GRID; n++) {
            const corner = cellTopLeft(n);
            const centre = cellCenter(n);

            // then
            expect(centre.x - corner.x).toBeCloseTo(half, 6);
            expect(centre.y - corner.y).toBeCloseTo(half, 6);
        }
    });
});

describe("buildLadder", () => {
    it("draws two rails the same width apart at both ends", () => {
        // given
        const from = 1;
        const to = 38;

        // when
        const ladder = buildLadder(from, to);

        // then
        const [railA, railB] = ladder.rails;
        expect(ladder.rails).toHaveLength(2);
        expect(distance(railA[0], railB[0])).toBeCloseTo(34, 6);
        expect(distance(railA[1], railB[1])).toBeCloseTo(34, 6);
    });

    it("centres the rails on the two squares it joins", () => {
        // given
        const from = 4;
        const to = 14;

        // when
        const ladder = buildLadder(from, to);

        // then
        const [railA, railB] = ladder.rails;
        const foot = cellCenter(from);
        const top = cellCenter(to);
        const footMid = midpoint(railA[0], railB[0]);
        const topMid = midpoint(railA[1], railB[1]);
        expectPoint(footMid, foot.x, foot.y);
        expectPoint(topMid, top.x, top.y);
    });

    it("spaces the rungs evenly up a vertical ladder", () => {
        // given
        const from = 71;
        const to = 91;

        // when
        const ladder = buildLadder(from, to);

        // then
        expect(ladder.rungs).toHaveLength(3);
        expectPoint(ladder.rungs[0][0], 967, 200);
        expectPoint(ladder.rungs[0][1], 933, 200);
        expectPoint(ladder.rungs[1][0], 967, 150);
        expectPoint(ladder.rungs[2][0], 967, 100);
    });

    it("gives every rung the same width as the gap between the rails", () => {
        // given
        const from = 28;
        const to = 84;

        // when
        const ladder = buildLadder(from, to);

        // then
        expect(ladder.rungs.length).toBeGreaterThan(1);
        for (const rung of ladder.rungs) {
            expect(distance(rung[0], rung[1])).toBeCloseTo(34, 6);
        }
    });

    it("adds a rung roughly every forty six units of length", () => {
        // given
        const from = 1;
        const to = 38;
        const length = distance(cellCenter(from), cellCenter(to));

        // when
        const ladder = buildLadder(from, to);

        // then
        expect(Math.floor(length / 46)).toBe(7);
        expect(ladder.rungs).toHaveLength(6);
    });

    it("still draws a rung on a ladder too short to space them out", () => {
        // given
        const from = 1;
        const to = 2;

        // when
        const ladder = buildLadder(from, to);

        // then
        expect(ladder.rungs).toHaveLength(1);
        expectPoint(ladder.rungs[0][0], 100, 967);
    });

    it("survives a ladder that starts and ends on the same square", () => {
        // given
        const square = 1;

        // when
        const ladder = buildLadder(square, square);

        // then
        expect(ladder.rungs).toHaveLength(1);
        for (const pair of [...ladder.rails, ...ladder.rungs]) {
            for (const point of pair) {
                expect(Number.isFinite(point.x)).toBe(true);
                expect(Number.isFinite(point.y)).toBe(true);
            }
        }
    });

    it("keeps every real ladder inside the drawing area", () => {
        // given
        const entries = Object.entries(LADDERS);

        // when
        const ladders = entries.map(([from, to]) => buildLadder(Number(from), to));

        // then
        for (const ladder of ladders) {
            for (const pair of [...ladder.rails, ...ladder.rungs]) {
                for (const point of pair) {
                    expect(point.x).toBeGreaterThanOrEqual(0);
                    expect(point.x).toBeLessThanOrEqual(VIEW);
                    expect(point.y).toBeGreaterThanOrEqual(0);
                    expect(point.y).toBeLessThanOrEqual(VIEW);
                }
            }
        }
    });
});

describe("buildSnake", () => {
    it("puts the head on the square the snake swallows you from", () => {
        // given
        const from = 16;
        const to = 6;

        // when
        const snake = buildSnake(from, to);

        // then
        expectPoint(snake.head, 450, 850);
        expect(snake.headRadius).toBe(30);
        expectPoint(cellCenter(to), 550, 950);
    });

    it("closes the body outline with a point either side of every step", () => {
        // given
        const steps = 28;

        // when
        const snake = buildSnake(47, 26);

        // then
        expect(snake.body.startsWith("M ")).toBe(true);
        expect(snake.body.endsWith(" Z")).toBe(true);
        expect(snake.body.split(" L ").length - 1).toBe(steps * 2 + 1);
    });

    it("draws the belly along the centre line from the head", () => {
        // given
        const snake = buildSnake(93, 73);

        // when
        const head = snake.head;

        // then
        expect(snake.belly.startsWith(`M ${head.x.toFixed(1)} ${head.y.toFixed(1)}`)).toBe(true);
        expect(snake.belly.split(" L ").length - 1).toBe(28);
        expect(snake.belly).not.toContain("Z");
    });

    it("rounds every drawn coordinate to a single decimal place", () => {
        // given
        const snake = buildSnake(62, 19);

        // when
        const drawn = `${snake.body} ${snake.belly} ${snake.tongue}`.match(/-?\d+\.\d+/g) ?? [];

        // then
        expect(drawn.length).toBeGreaterThan(0);
        for (const number of drawn) {
            expect(number).toMatch(/^-?\d+\.\d$/);
        }
    });

    it("faces the head away from the tail for every snake on the board", () => {
        // given
        const entries = Object.entries(SNAKES);

        // when
        const snakes = entries.map(([from, to]) => ({ from: Number(from), to, snake: buildSnake(Number(from), to) }));

        // then
        for (const { from, to, snake } of snakes) {
            const head = cellCenter(from);
            const tail = cellCenter(to);
            const away = { x: head.x - tail.x, y: head.y - tail.y };
            const awayLength = Math.hypot(away.x, away.y);
            const facing = { x: Math.cos(snake.angle), y: Math.sin(snake.angle) };
            const dot = (facing.x * away.x + facing.y * away.y) / awayLength;
            expect(dot).toBeGreaterThan(0);
        }
    });

    it("sets the eyes symmetrically about the head", () => {
        // given
        const snake = buildSnake(87, 24);

        // when
        const [left, right] = snake.eyes;

        // then
        expect(distance(left, right)).toBeCloseTo(23.4, 6);
        expect(distance(snake.head, left)).toBeCloseTo(distance(snake.head, right), 6);
    });

    it("points the pupils in the same direction as the head", () => {
        // given
        const snake = buildSnake(95, 75);

        // when
        const offset = snake.pupilOffset;

        // then
        expect(Math.hypot(offset.x, offset.y)).toBeCloseTo(3, 6);
        expect(Math.atan2(offset.y, offset.x)).toBeCloseTo(snake.angle, 6);
    });

    it("forks the tongue into two tips ahead of the head", () => {
        // given
        const snake = buildSnake(98, 78);

        // when
        const points = numbersIn(snake.tongue);

        // then
        expect(points).toHaveLength(12);
        const root = { x: points[0], y: points[1] };
        const tip = { x: points[2], y: points[3] };
        const forkA = { x: points[6], y: points[7] };
        const forkB = { x: points[10], y: points[11] };
        expect(distance(root, tip)).toBeCloseTo(26, 0);
        expect(distance(tip, forkA)).toBeCloseTo(distance(tip, forkB), 0);
        expect(distance(snake.head, root)).toBeCloseTo(23.4, 0);
    });

    it("survives a snake that starts and ends on the same square", () => {
        // given
        const square = 56;

        // when
        const snake = buildSnake(square, square);

        // then
        expect(snake.body).not.toContain("NaN");
        expect(snake.belly).not.toContain("NaN");
        expect(snake.tongue).not.toContain("NaN");
        expect(Number.isFinite(snake.angle)).toBe(true);
    });
});
