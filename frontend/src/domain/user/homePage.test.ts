import { describe, expect, it } from "vitest";
import { HOME_PAGE_FALLBACK, HOME_PAGE_OPTIONS, HOME_PAGE_ROUTES, homePageRoute } from "./homePage";

describe("home page options", () => {
    it("offers exactly the twenty-one destinations the settings form lists", () => {
        // given
        const options = HOME_PAGE_OPTIONS;

        // when
        const count = options.length;

        // then
        expect(count).toBe(21);
    });

    it("gives every option a route, so choosing one never lands on the fallback by accident", () => {
        // given
        const offered = HOME_PAGE_OPTIONS.map(option => option.value).sort();

        // when
        const routed = Object.keys(HOME_PAGE_ROUTES).sort();

        // then
        expect(routed).toEqual(offered);
    });

    it("names every option, so the settings form never renders a blank entry", () => {
        // given
        const options = HOME_PAGE_OPTIONS;

        // when
        const unnamed = options.filter(option => option.label.trim() === "");

        // then
        expect(unnamed).toEqual([]);
    });

    it("offers each destination once", () => {
        // given
        const values = HOME_PAGE_OPTIONS.map(option => option.value);

        // when
        const distinct = new Set(values);

        // then
        expect(distinct.size).toBe(values.length);
    });
});

describe("homePageRoute", () => {
    it("sends a member to the page they chose", () => {
        // given
        const chosen = "journals";

        // when
        const route = homePageRoute(chosen);

        // then
        expect(route).toBe("/journals");
    });

    it("sends a member who has chosen nothing to the landing page", () => {
        // given
        const chosen = undefined;

        // when
        const route = homePageRoute(chosen);

        // then
        expect(route).toBe(HOME_PAGE_FALLBACK);
    });

    it("falls back to the landing page for a destination nobody offers", () => {
        // given
        const chosen = "the golden land";

        // when
        const route = homePageRoute(chosen);

        // then
        expect(route).toBe(HOME_PAGE_FALLBACK);
    });
});
