import { screen } from "@testing-library/react";
import { Route, Routes } from "react-router";
import { describe, expect, it } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { MissingEmailRoute } from "./MissingEmailRoute";

function renderGuarded(email: string) {
    return renderWithProviders(
        <Routes>
            <Route element={<MissingEmailRoute />}>
                <Route path="/set-email" element={<p>add your email</p>} />
            </Route>
            <Route path="/" element={<p>front page</p>} />
        </Routes>,
        { route: "/set-email", user: makeUser({ email }) },
    );
}

describe("MissingEmailRoute", () => {
    it("sends somebody who already has an email back to the front page", () => {
        // given
        const email = "battler@rokkenjima.jp";

        // when
        renderGuarded(email);

        // then
        expect(screen.getByText("front page")).toBeInTheDocument();
        expect(screen.queryByText("add your email")).not.toBeInTheDocument();
    });

    it("opens the form for somebody with no email yet", () => {
        // given
        const email = "";

        // when
        renderGuarded(email);

        // then
        expect(screen.getByText("add your email")).toBeInTheDocument();
    });
});
