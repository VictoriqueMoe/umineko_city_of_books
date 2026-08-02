import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { NativeLinkInterceptor } from "./NativeLinkInterceptor";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));
const { capacitor } = vi.hoisted(() => ({ capacitor: { isNativePlatform: vi.fn() } }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => navigate };
});

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));

const observed: boolean[] = [];

function observe(event: Event): void {
    observed.push(event.defaultPrevented);
    event.preventDefault();
}

function renderInterceptor(children: ReactNode) {
    return renderWithProviders(
        <>
            <NativeLinkInterceptor />
            {children}
        </>,
    );
}

beforeEach(() => {
    observed.length = 0;
    capacitor.isNativePlatform.mockReturnValue(true);
    window.addEventListener("click", observe);
});

afterEach(() => {
    window.removeEventListener("click", observe);
});

describe("NativeLinkInterceptor in the app", () => {
    it("keeps an internal link inside the app", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        await user.click(screen.getByRole("link", { name: "a theory" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/12");
    });

    it("stops the browser from following an internal link itself", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        await user.click(screen.getByRole("link", { name: "a theory" }));

        // then
        expect(observed).toEqual([true]);
    });

    it("carries the query and fragment of an internal link across", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="/ships?sort=new#top">ships</a>);

        // when
        await user.click(screen.getByRole("link", { name: "ships" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/ships?sort=new#top");
    });

    it("treats a link to the canonical site as internal", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="https://whentheycry.social/ships/3">a ship</a>);

        // when
        await user.click(screen.getByRole("link", { name: "a ship" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/ships/3");
    });

    it("follows the nearest link when the click lands on something inside it", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(
            <a href="/theory/12">
                <span>the title</span>
            </a>,
        );

        // when
        await user.click(screen.getByText("the title"));

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/12");
    });

    it("leaves a link to another site to the browser", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="https://example.com/elsewhere">elsewhere</a>);

        // when
        await user.click(screen.getByRole("link", { name: "elsewhere" }));

        // then
        expect(navigate).not.toHaveBeenCalled();
        expect(observed).toEqual([false]);
    });

    it("leaves a link with a non web protocol to the browser", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="mailto:beatrice@example.com">write to us</a>);

        // when
        await user.click(screen.getByRole("link", { name: "write to us" }));

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("leaves a download link alone", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(
            <a href="/uploads/portrait.png" download>
                save the portrait
            </a>,
        );

        // when
        await user.click(screen.getByRole("link", { name: "save the portrait" }));

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("leaves a link that has no destination alone", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a>a link going nowhere</a>);

        // when
        await user.click(screen.getByText("a link going nowhere"));

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("leaves an unparseable destination alone", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="http://[">a broken link</a>);

        // when
        await user.click(screen.getByRole("link", { name: "a broken link" }));

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("ignores a click that did not land on a link at all", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<p>just some prose</p>);

        // when
        await user.click(screen.getByText("just some prose"));

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("ignores a click that something else has already handled", async () => {
        // given
        const prevent = (event: Event) => event.preventDefault();
        window.addEventListener("click", prevent, true);
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        fireEvent.click(screen.getByRole("link", { name: "a theory" }));
        window.removeEventListener("click", prevent, true);

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("ignores a click made with a modifier key held down", () => {
        // given
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        fireEvent.click(screen.getByRole("link", { name: "a theory" }), { ctrlKey: true });

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("ignores a click made with the shift key held down", () => {
        // given
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        fireEvent.click(screen.getByRole("link", { name: "a theory" }), { shiftKey: true });

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("ignores a click from a button other than the primary one", () => {
        // given
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        fireEvent.click(screen.getByRole("link", { name: "a theory" }), { button: 1 });

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("stops listening once it is taken off the page", async () => {
        // given
        const user = userEvent.setup();
        const { rerender } = renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        rerender(<a href="/theory/12">a theory</a>);
        await user.click(screen.getByRole("link", { name: "a theory" }));

        // then
        expect(navigate).not.toHaveBeenCalled();
    });
});

describe("NativeLinkInterceptor on the web", () => {
    beforeEach(() => {
        capacitor.isNativePlatform.mockReturnValue(false);
    });

    it("leaves every link to the browser", async () => {
        // given
        const user = userEvent.setup();
        renderInterceptor(<a href="/theory/12">a theory</a>);

        // when
        await user.click(screen.getByRole("link", { name: "a theory" }));

        // then
        expect(navigate).not.toHaveBeenCalled();
        expect(observed).toEqual([false]);
    });

    it("renders nothing of its own", () => {
        // given
        const platform = capacitor.isNativePlatform;

        // when
        const { container } = renderWithProviders(<NativeLinkInterceptor />);

        // then
        expect(container).toBeEmptyDOMElement();
        expect(platform).toHaveBeenCalled();
    });
});
