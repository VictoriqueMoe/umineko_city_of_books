import { describe, expect, it } from "vitest";
import { placeMenu, type MenuPoint, type MenuSize } from "./placeMenu";

const MENU: MenuSize = { width: 172, height: 200 };
const VIEWPORT: MenuSize = { width: 1000, height: 800 };

interface PlacementCase {
    name: string;
    origin: MenuPoint;
    size?: MenuSize;
    viewport?: MenuSize;
    rightToLeft?: boolean;
    want: MenuPoint;
}

const placementCases: PlacementCase[] = [
    {
        name: "opens down and to the right of the pointer when there is room",
        origin: { x: 100, y: 120 },
        want: { x: 100, y: 120 },
    },
    {
        name: "flips back by its own width at the right edge",
        origin: { x: 900, y: 120 },
        want: { x: 728, y: 120 },
    },
    {
        name: "flips up by its own height at the bottom edge",
        origin: { x: 100, y: 700 },
        want: { x: 100, y: 500 },
    },
    {
        name: "flips on both axes in the bottom corner",
        origin: { x: 980, y: 780 },
        want: { x: 808, y: 580 },
    },
    {
        name: "keeps a margin rather than touching the top left corner",
        origin: { x: 0, y: 0 },
        want: { x: 8, y: 8 },
    },
    {
        name: "opens to the left of the pointer in a right to left layout",
        origin: { x: 500, y: 120 },
        rightToLeft: true,
        want: { x: 328, y: 120 },
    },
    {
        name: "flips back to the right when a right to left menu would leave the viewport",
        origin: { x: 60, y: 120 },
        rightToLeft: true,
        want: { x: 60, y: 120 },
    },
    {
        name: "clamps into a stream chat sidebar too narrow for the flipped menu",
        origin: { x: 295, y: 120 },
        viewport: { width: 300, height: 800 },
        want: { x: 120, y: 120 },
    },
    {
        name: "clamps to the margin when the menu is taller than the viewport",
        origin: { x: 100, y: 60 },
        viewport: { width: 1000, height: 150 },
        want: { x: 100, y: 8 },
    },
    {
        name: "rounds a fractional pointer position to whole pixels",
        origin: { x: 100.6, y: 120.4 },
        want: { x: 101, y: 120 },
    },
];

describe("placeMenu", () => {
    for (const placementCase of placementCases) {
        it(placementCase.name, () => {
            // given
            const size = placementCase.size ?? MENU;
            const viewport = placementCase.viewport ?? VIEWPORT;

            // when
            const placed = placeMenu(placementCase.origin, size, viewport, placementCase.rightToLeft ?? false);

            // then
            expect(placed).toEqual(placementCase.want);
        });
    }

    it("never places the menu past either far edge of the viewport", () => {
        // given
        const viewport: MenuSize = { width: 420, height: 320 };

        // when
        const placed = placeMenu({ x: 419, y: 319 }, MENU, viewport, false);

        // then
        expect(placed.x + MENU.width).toBeLessThanOrEqual(viewport.width - 8);
        expect(placed.y + MENU.height).toBeLessThanOrEqual(viewport.height - 8);
    });
});
