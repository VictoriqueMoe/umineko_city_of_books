import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { VoiceButton } from "./VoiceButton";

function renderButton(overrides: Partial<ComponentProps<typeof VoiceButton>> = {}) {
    const onJoin = vi.fn();
    const onLeave = vi.fn();
    const result = renderWithProviders(
        <VoiceButton enabled status="idle" presenceCount={0} onJoin={onJoin} onLeave={onLeave} {...overrides} />,
    );

    return { ...result, onJoin, onLeave };
}

describe("VoiceButton", () => {
    it("renders nothing when voice is switched off for the site", () => {
        // given
        const enabled = false;

        // when
        const { container } = renderButton({ enabled });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("invites the viewer into an empty call without a count", () => {
        // given
        const presenceCount = 0;

        // when
        renderButton({ presenceCount });

        // then
        const button = screen.getByTitle("Join voice");
        expect(button).toHaveTextContent("Voice");
        expect(button).not.toHaveTextContent("·");
    });

    it("shows how many people are already talking", () => {
        // given
        const presenceCount = 3;

        // when
        renderButton({ presenceCount });

        // then
        expect(screen.getByTitle("Join voice")).toHaveTextContent("Voice · 3");
    });

    it("asks first rather than joining the moment it is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onJoin } = renderButton();

        // when
        await user.click(screen.getByTitle("Join voice"));

        // then
        expect(screen.getByText("Join the voice call?")).toBeInTheDocument();
        expect(onJoin).not.toHaveBeenCalled();
    });

    it("joins the call once the viewer confirms", async () => {
        // given
        const user = userEvent.setup();
        const { onJoin, onLeave } = renderButton();

        // when
        await user.click(screen.getByTitle("Join voice"));
        await user.click(screen.getByRole("button", { name: "Join voice" }));

        // then
        expect(onJoin).toHaveBeenCalledTimes(1);
        expect(onLeave).not.toHaveBeenCalled();
    });

    it("joins nothing when the viewer backs out, which is the misclick it exists for", async () => {
        // given
        const user = userEvent.setup();
        const { onJoin } = renderButton();

        // when
        await user.click(screen.getByTitle("Join voice"));
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(onJoin).not.toHaveBeenCalled();
        expect(screen.queryByText("Join the voice call?")).not.toBeInTheDocument();
    });

    it("blocks a second press while the call is still connecting", () => {
        // given
        const status = "connecting" as const;

        // when
        renderButton({ status, presenceCount: 2 });

        // then
        const button = screen.getByTitle("Join voice");
        expect(button).toBeDisabled();
        expect(button).toHaveTextContent("Joining");
    });

    it("offers to leave once the viewer is connected", () => {
        // given
        const status = "connected" as const;

        // when
        renderButton({ status });

        // then
        expect(screen.getByTitle("Leave voice")).toHaveTextContent("Leave voice");
        expect(screen.queryByTitle("Join voice")).not.toBeInTheDocument();
    });

    it("leaves the call when the connected button is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { onJoin, onLeave } = renderButton({ status: "connected" });

        // when
        await user.click(screen.getByTitle("Leave voice"));

        // then
        expect(onLeave).toHaveBeenCalledTimes(1);
        expect(onJoin).not.toHaveBeenCalled();
    });

    it("hides the leave control too when voice is switched off mid call", () => {
        // given
        const enabled = false;

        // when
        const { container } = renderButton({ enabled, status: "connected" });

        // then
        expect(container).toBeEmptyDOMElement();
    });
});
