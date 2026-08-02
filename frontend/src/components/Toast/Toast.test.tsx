import { act, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { Toast } from "./Toast";

describe("Toast", () => {
    it("announces its message politely to assistive technology", () => {
        // given
        const message = "Your theory was saved";

        // when
        renderWithProviders(<Toast>{message}</Toast>);

        // then
        const toast = screen.getByRole("status");
        expect(toast).toHaveTextContent(message);
        expect(toast).toHaveAttribute("aria-live", "polite");
    });

    it("dismisses itself after the default four seconds", () => {
        // given
        vi.useFakeTimers();
        const onDismiss = vi.fn();
        renderWithProviders(<Toast onDismiss={onDismiss}>Your theory was saved</Toast>);

        // when
        act(() => {
            vi.advanceTimersByTime(3999);
        });

        // then
        expect(onDismiss).not.toHaveBeenCalled();
        act(() => {
            vi.advanceTimersByTime(1);
        });
        expect(onDismiss).toHaveBeenCalledOnce();
    });

    it("honours a shorter lifetime when one is given", () => {
        // given
        vi.useFakeTimers();
        const onDismiss = vi.fn();
        renderWithProviders(
            <Toast onDismiss={onDismiss} duration={1000}>
                Your theory was saved
            </Toast>,
        );

        // when
        act(() => {
            vi.advanceTimersByTime(1000);
        });

        // then
        expect(onDismiss).toHaveBeenCalledOnce();
    });

    it("stays on screen forever when the duration is zero", () => {
        // given
        vi.useFakeTimers();
        const onDismiss = vi.fn();
        renderWithProviders(
            <Toast onDismiss={onDismiss} duration={0}>
                Your theory was saved
            </Toast>,
        );

        // when
        act(() => {
            vi.advanceTimersByTime(60_000);
        });

        // then
        expect(vi.getTimerCount()).toBe(0);
        expect(onDismiss).not.toHaveBeenCalled();
    });

    it("schedules nothing when nobody is listening for the dismissal", () => {
        // given
        vi.useFakeTimers();

        // when
        renderWithProviders(<Toast>Your theory was saved</Toast>);

        // then
        expect(vi.getTimerCount()).toBe(0);
    });

    it("cancels its timer when it is removed before the time is up", () => {
        // given
        vi.useFakeTimers();
        const onDismiss = vi.fn();
        const { unmount } = renderWithProviders(<Toast onDismiss={onDismiss}>Your theory was saved</Toast>);

        // when
        unmount();
        act(() => {
            vi.advanceTimersByTime(10_000);
        });

        // then
        expect(onDismiss).not.toHaveBeenCalled();
        expect(vi.getTimerCount()).toBe(0);
    });

    it("restarts the countdown when the duration changes", () => {
        // given
        vi.useFakeTimers();
        const onDismiss = vi.fn();
        const { rerender } = renderWithProviders(
            <Toast onDismiss={onDismiss} duration={1000}>
                Your theory was saved
            </Toast>,
        );
        act(() => {
            vi.advanceTimersByTime(900);
        });

        // when
        rerender(
            <Toast onDismiss={onDismiss} duration={2000}>
                Your theory was saved
            </Toast>,
        );
        act(() => {
            vi.advanceTimersByTime(1900);
        });

        // then
        expect(onDismiss).not.toHaveBeenCalled();
        act(() => {
            vi.advanceTimersByTime(100);
        });
        expect(onDismiss).toHaveBeenCalledOnce();
    });

    it("gives each variant its own styling", () => {
        // given
        const variants = ["default", "success", "error", "arcane"] as const;

        // when
        renderWithProviders(
            <>
                {variants.map(variant => (
                    <Toast key={variant} variant={variant}>
                        {variant}
                    </Toast>
                ))}
            </>,
        );

        // then
        const seen = new Set<string>();
        const toasts = screen.getAllByRole("status");
        for (const toast of toasts) {
            seen.add(toast.className);
        }
        expect(toasts).toHaveLength(variants.length);
        expect(seen.size).toBe(variants.length);
    });

    it("falls back to the default variant when none is given", () => {
        // given
        const markup = (
            <>
                <Toast>plain</Toast>
                <Toast variant="default">explicit</Toast>
                <Toast variant="error">error</Toast>
            </>
        );

        // when
        renderWithProviders(markup);

        // then
        const [plain, explicit, error] = screen.getAllByRole("status");
        expect(plain.className).toBe(explicit.className);
        expect(plain.className).not.toBe(error.className);
    });
});
