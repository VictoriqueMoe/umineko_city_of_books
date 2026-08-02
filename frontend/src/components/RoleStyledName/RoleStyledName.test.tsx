import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RoleStyledName } from "./RoleStyledName";

function classFor(role?: string): string {
    const { container, unmount } = render(<RoleStyledName name="Beatrice" role={role} />);
    const cls = (container.firstElementChild as HTMLElement).getAttribute("class") ?? "";
    unmount();
    return cls;
}

describe("RoleStyledName", () => {
    it("shows the display name it was given", () => {
        // given
        const name = "Beatrice";

        // when
        render(<RoleStyledName name={name} />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("renders the name as plain text rather than a link", () => {
        // given
        const name = "Beatrice";

        // when
        const { container } = render(<RoleStyledName name={name} role="admin" />);

        // then
        expect(container.querySelector("a")).toBeNull();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("styles a moderator differently from a plain member", () => {
        // given
        const plain = classFor(undefined);

        // when
        const moderator = classFor("moderator");

        // then
        expect(moderator).not.toBe(plain);
    });

    it("gives each staff rank its own styling", () => {
        // given
        const moderator = classFor("moderator");

        // when
        const admin = classFor("admin");
        const superAdmin = classFor("super_admin");

        // then
        expect(new Set([moderator, admin, superAdmin]).size).toBe(3);
    });

    it("styles an unrecognised role like a plain member", () => {
        // given
        const plain = classFor(undefined);

        // when
        const unknown = classFor("archivist");

        // then
        expect(unknown).toBe(plain);
    });

    it("styles an empty role like a plain member", () => {
        // given
        const plain = classFor(undefined);

        // when
        const empty = classFor("");

        // then
        expect(empty).toBe(plain);
    });
});
