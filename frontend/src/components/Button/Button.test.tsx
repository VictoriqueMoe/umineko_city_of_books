import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { FormEvent } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Button } from "./Button";

describe("Button", () => {
    it("renders its children as the accessible name", () => {
        // given
        const label = "Reveal the truth";

        // when
        renderWithProviders(<Button>{label}</Button>);

        // then
        expect(screen.getByRole("button", { name: label })).toBeInTheDocument();
    });

    it("forwards native button attributes to the underlying element", () => {
        // given
        const title = "close the golden land";

        // when
        renderWithProviders(
            <Button type="submit" name="verdict" value="guilty" title={title} aria-label="Cast the verdict" />,
        );

        // then
        const button = screen.getByRole("button", { name: "Cast the verdict" });
        expect(button).toHaveAttribute("type", "submit");
        expect(button).toHaveAttribute("name", "verdict");
        expect(button).toHaveAttribute("value", "guilty");
        expect(button).toHaveAttribute("title", title);
    });

    it("calls the click handler it was given when pressed", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Button onClick={onClick}>Press me</Button>);

        // when
        await user.click(screen.getByRole("button", { name: "Press me" }));

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("ignores clicks while it is disabled", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <Button onClick={onClick} disabled>
                Press me
            </Button>,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Press me" }));

        // then
        expect(screen.getByRole("button", { name: "Press me" })).toBeDisabled();
        expect(onClick).not.toHaveBeenCalled();
    });

    it("submits the form it belongs to when it is a submit button", async () => {
        // given
        const onSubmit = vi.fn((event: FormEvent) => {
            event.preventDefault();
        });
        const user = userEvent.setup();
        renderWithProviders(
            <form onSubmit={onSubmit}>
                <Button type="submit">Send</Button>
            </form>,
        );

        // when
        await user.click(screen.getByRole("button", { name: "Send" }));

        // then
        expect(onSubmit).toHaveBeenCalledOnce();
    });

    it("keeps a caller supplied class alongside its own classes", () => {
        // given
        const className = "wide-button";

        // when
        renderWithProviders(<Button className={className}>Press me</Button>);

        // then
        const button = screen.getByRole("button", { name: "Press me" });
        expect(button).toHaveClass(className);
        expect(button.className.split(" ").length).toBeGreaterThan(1);
    });

    it("defaults to the secondary variant at the medium size", () => {
        // given
        const buttons = (
            <>
                <Button>default</Button>
                <Button variant="secondary" size="medium">
                    explicit
                </Button>
                <Button variant="primary" size="small">
                    primary
                </Button>
            </>
        );

        // when
        renderWithProviders(buttons);

        // then
        const fallback = screen.getByRole("button", { name: "default" });
        const explicit = screen.getByRole("button", { name: "explicit" });
        const primary = screen.getByRole("button", { name: "primary" });
        expect(fallback.className).toBe(explicit.className);
        expect(fallback.className).not.toBe(primary.className);
    });

    it("gives every variant its own styling", () => {
        // given
        const variants = ["primary", "secondary", "danger", "ghost"] as const;

        // when
        renderWithProviders(
            <>
                {variants.map(variant => (
                    <Button key={variant} variant={variant}>
                        {variant}
                    </Button>
                ))}
            </>,
        );

        // then
        const seen = new Set<string>();
        for (const variant of variants) {
            seen.add(screen.getByRole("button", { name: variant }).className);
        }
        expect(seen.size).toBe(variants.length);
    });
});
