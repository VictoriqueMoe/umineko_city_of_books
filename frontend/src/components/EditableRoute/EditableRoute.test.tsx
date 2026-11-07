import { screen } from "@testing-library/react";
import { Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { EditableRoute } from "./EditableRoute";

const mocks = vi.hoisted(() => ({ useEditable: vi.fn() }));

vi.mock("../../hooks/useEditable", () => ({ useEditable: mocks.useEditable }));

function renderGuarded() {
    return renderWithProviders(
        <Routes>
            <Route element={<EditableRoute resource="mystery" />}>
                <Route path="/mystery/:id/edit" element={<p>the game board</p>} />
            </Route>
            <Route path="/" element={<p>front page</p>} />
        </Routes>,
        { route: "/mystery/mystery-1/edit", user: makeUser() },
    );
}

describe("EditableRoute", () => {
    it("holds the route while the resource is loading", () => {
        // given
        mocks.useEditable.mockReturnValue({ loading: true, allowed: false });

        // when
        renderGuarded();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("front page")).not.toBeInTheDocument();
        expect(screen.queryByText("the game board")).not.toBeInTheDocument();
    });

    it("sends somebody who may not edit back to the front page", () => {
        // given
        mocks.useEditable.mockReturnValue({ loading: false, allowed: false });

        // when
        renderGuarded();

        // then
        expect(screen.getByText("front page")).toBeInTheDocument();
        expect(screen.queryByText("the game board")).not.toBeInTheDocument();
    });

    it("opens the editor for somebody who may edit", () => {
        // given
        mocks.useEditable.mockReturnValue({ loading: false, allowed: true });

        // when
        renderGuarded();

        // then
        expect(screen.getByText("the game board")).toBeInTheDocument();
    });

    it("asks about the resource it guards", () => {
        // given
        mocks.useEditable.mockReturnValue({ loading: false, allowed: true });

        // when
        renderGuarded();

        // then
        expect(mocks.useEditable).toHaveBeenCalledWith("mystery");
    });
});
