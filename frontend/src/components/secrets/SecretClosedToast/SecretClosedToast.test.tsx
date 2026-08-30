import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SecretClosedToast } from "./SecretClosedToast";
import { renderWithProviders } from "../../../test-utils/render";
import { emitRealtimeEvent } from "../../../test-utils/ws";

describe("SecretClosedToast", () => {
    it("announces a secret that somebody else solved first", () => {
        // given
        renderWithProviders(<SecretClosedToast />);

        // when
        emitRealtimeEvent({
            type: "secret_closed",
            data: {
                secret_id: "secret-1",
                secret_title: "The Golden Truth",
                solver: { display_name: "Beatrice", username: "beato" },
            },
        });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Beatrice solved The Golden Truth");
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/secrets/secret-1");
    });

    it("ignores a secret closed event that names no solver", () => {
        // given
        renderWithProviders(<SecretClosedToast />);

        // when
        emitRealtimeEvent({ type: "secret_closed", data: { secret_id: "secret-1" } });

        // then
        expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });

    it("falls back to the username when the solver has no display name", () => {
        // given
        renderWithProviders(<SecretClosedToast />);

        // when
        emitRealtimeEvent({
            type: "secret_closed",
            data: {
                secret_id: "secret-2",
                secret_title: "The Witch's Epitaph",
                solver: { display_name: "", username: "beato" },
            },
        });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("beato solved The Witch's Epitaph");
    });

    it("says nothing until a secret is solved", () => {
        // given
        const { container } = renderWithProviders(<SecretClosedToast />);

        // when
        const toast = container.querySelector("[role=status]");

        // then
        expect(toast).toBeNull();
    });
});
