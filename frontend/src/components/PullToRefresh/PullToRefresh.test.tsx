import { act, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { PullToRefresh } from "./PullToRefresh";

const mocks = vi.hoisted(() => ({
    isNativeApp: vi.fn(),
    refetchQueries: vi.fn(),
}));

vi.mock("../../utils/authToken", () => ({ isNativeApp: mocks.isNativeApp }));
vi.mock("../../api/queryClient", () => ({ queryClient: { refetchQueries: mocks.refetchQueries } }));

function touches(...ys: number[]) {
    return ys.map(clientY => ({ clientX: 0, clientY }));
}

function setScrollY(value: number): void {
    Object.defineProperty(window, "scrollY", { configurable: true, value });
}

function renderPullToRefresh() {
    const result = renderWithProviders(
        <PullToRefresh>
            <p>Rokkenjima</p>
        </PullToRefresh>,
    );
    const indicator = result.container.firstElementChild as HTMLElement;
    return { ...result, indicator };
}

async function settleRefresh(): Promise<void> {
    await act(async () => {
        await vi.advanceTimersByTimeAsync(600);
    });
}

describe("PullToRefresh", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        mocks.isNativeApp.mockReturnValue(true);
        mocks.refetchQueries.mockResolvedValue(undefined);
        setScrollY(0);
    });

    afterEach(() => {
        delete document.body.dataset.chatPage;
        setScrollY(0);
    });

    it("renders whatever it wraps", () => {
        // given
        const child = "Rokkenjima";

        // when
        renderPullToRefresh();

        // then
        expect(screen.getByText(child)).toBeInTheDocument();
    });

    it("keeps the indicator hidden until the finger actually moves", () => {
        // given
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });

        // then
        expect(indicator).toHaveAttribute("aria-hidden", "true");
    });

    it("follows the finger at half speed", () => {
        // given
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });

        // then
        expect(indicator).toHaveAttribute("aria-hidden", "false");
        expect(indicator.style.transform).toContain("translateY(75px)");
    });

    it("refuses to stretch beyond the maximum pull", () => {
        // given
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(900) });

        // then
        expect(indicator.style.transform).toContain("translateY(110px)");
    });

    it("does not refresh when the pull stops short of the threshold", () => {
        // given
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(100) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
        expect(indicator.style.transform).toContain("translateY(0px)");
    });

    it("refreshes the active queries once the pull passes the threshold", async () => {
        // given
        renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).toHaveBeenCalledWith({ type: "active" });
        await settleRefresh();
    });

    it("holds the spinner at the threshold until the minimum spin has elapsed", async () => {
        // given
        const { indicator } = renderPullToRefresh();
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });

        // when
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(indicator.style.transform).toContain("translateY(70px)");
        await settleRefresh();
        expect(indicator.style.transform).toContain("translateY(0px)");
        expect(indicator).toHaveAttribute("aria-hidden", "true");
    });

    it("will not start a second refresh while one is still running", async () => {
        // given
        renderPullToRefresh();
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).toHaveBeenCalledOnce();
        await settleRefresh();
    });

    it("ignores touches entirely outside the native app", () => {
        // given
        mocks.isNativeApp.mockReturnValue(false);
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
        expect(indicator.style.transform).toContain("translateY(0px)");
    });

    it("ignores a gesture made with more than one finger", () => {
        // given
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0, 20) });
        fireEvent.touchMove(document.body, { touches: touches(150, 170) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
        expect(indicator.style.transform).toContain("translateY(0px)");
    });

    it("ignores a pull that starts partway down the page", () => {
        // given
        setScrollY(240);
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
        expect(indicator.style.transform).toContain("translateY(0px)");
    });

    it("leaves chat pages alone", () => {
        // given
        document.body.dataset.chatPage = "true";
        const { indicator } = renderPullToRefresh();

        // when
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
        expect(indicator.style.transform).toContain("translateY(0px)");
    });

    it("abandons the pull when the finger travels back upwards", () => {
        // given
        const { indicator } = renderPullToRefresh();
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });

        // when
        fireEvent.touchMove(document.body, { touches: touches(-10) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
        expect(indicator.style.transform).toContain("translateY(0px)");
    });

    it("stops listening once it is unmounted", () => {
        // given
        const { unmount } = renderPullToRefresh();

        // when
        unmount();
        fireEvent.touchStart(document.body, { touches: touches(0) });
        fireEvent.touchMove(document.body, { touches: touches(150) });
        fireEvent.touchEnd(document.body, { touches: touches() });

        // then
        expect(mocks.refetchQueries).not.toHaveBeenCalled();
    });
});
