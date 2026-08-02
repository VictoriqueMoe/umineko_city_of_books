import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { SearchResult } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { MessageSearchPanel } from "./MessageSearchPanel";

const { useRoomMessageSearch } = vi.hoisted(() => ({ useRoomMessageSearch: vi.fn() }));

vi.mock("../../../api/queries/search", () => ({ useRoomMessageSearch }));

const roomId = "room-1";

function makeResult(overrides: Partial<SearchResult> = {}): SearchResult {
    return {
        type: "chat_message",
        id: "msg-1",
        parent_id: roomId,
        parent_title: "Golden Land",
        title: "",
        snippet: "the golden witch",
        url: "/chat/room-1",
        author: {
            id: "user-1",
            username: "beatrice",
            display_name: "Beatrice",
            avatar_url: "",
        },
        created_at: "2026-08-02T10:30:00Z",
        ...overrides,
    };
}

function stubSearch(overrides: { results?: SearchResult[]; total?: number; loading?: boolean } = {}) {
    useRoomMessageSearch.mockReturnValue({
        results: overrides.results ?? [],
        total: overrides.total ?? 0,
        loading: overrides.loading ?? false,
    });
}

function renderPanel(overrides: Partial<ComponentProps<typeof MessageSearchPanel>> = {}) {
    const onClose = vi.fn();
    const onJump = vi.fn();
    const result = renderWithProviders(
        <MessageSearchPanel roomId={roomId} isOpen onClose={onClose} onJump={onJump} {...overrides} />,
    );

    return { ...result, onClose, onJump };
}

describe("MessageSearchPanel", () => {
    it("renders nothing while the panel is closed", () => {
        // given
        stubSearch();

        // when
        const { container } = renderPanel({ isOpen: false });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("leaves the search query disabled while the panel is closed", () => {
        // given
        stubSearch();

        // when
        renderPanel({ isOpen: false });

        // then
        expect(useRoomMessageSearch).toHaveBeenLastCalledWith(roomId, "", 30, 0, false);
    });

    it("asks for at least two characters before it will search", () => {
        // given
        stubSearch();

        // when
        renderPanel();

        // then
        expect(screen.getByText("Type at least 2 characters to search.")).toBeInTheDocument();
    });

    it("still asks for more characters when a single letter is typed", async () => {
        // given
        stubSearch();
        const user = userEvent.setup();
        renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "b");

        // then
        await waitFor(() => {
            expect(useRoomMessageSearch).toHaveBeenLastCalledWith(roomId, "b", 30, 0, true);
        });
        expect(screen.getByText("Type at least 2 characters to search.")).toBeInTheDocument();
    });

    it("hands the debounced term to the search query", async () => {
        // given
        stubSearch();
        const user = userEvent.setup();
        renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "beat");

        // then
        await waitFor(() => {
            expect(useRoomMessageSearch).toHaveBeenLastCalledWith(roomId, "beat", 30, 0, true);
        });
    });

    it("shows a searching notice while the results are on their way", async () => {
        // given
        stubSearch({ loading: true });
        const user = userEvent.setup();
        renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "beat");

        // then
        expect(await screen.findByText("Searching...")).toBeInTheDocument();
    });

    it("reports that nothing matched once the search has settled", async () => {
        // given
        stubSearch({ results: [], total: 0 });
        const user = userEvent.setup();
        renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "beat");

        // then
        expect(await screen.findByText("No matching messages.")).toBeInTheDocument();
    });

    it("falls back to the username when a result author has no display name", async () => {
        // given
        stubSearch({
            results: [
                makeResult({
                    id: "msg-1",
                    author: { id: "u1", username: "beatrice", display_name: "", avatar_url: "" },
                }),
                makeResult({
                    id: "msg-2",
                    author: { id: "u2", username: "battler", display_name: "Battler", avatar_url: "" },
                }),
            ],
            total: 2,
        });
        const user = userEvent.setup();
        renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "witch");

        // then
        expect(await screen.findByText("beatrice")).toBeInTheDocument();
        expect(screen.getByText("Battler")).toBeInTheDocument();
    });

    it("keeps the highlight markup in a snippet but strips anything unsafe", async () => {
        // given
        stubSearch({
            results: [
                makeResult({
                    snippet: 'the <mark>golden</mark> witch<script>alert("kill")</script><img src="x" onerror="x">',
                }),
            ],
            total: 1,
        });
        const user = userEvent.setup();
        const { container } = renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "golden");

        // then
        await screen.findByText("Beatrice");
        expect(container.querySelector("mark")?.textContent).toBe("golden");
        expect(container.querySelector("script")).toBeNull();
        expect(container.querySelector("img")).toBeNull();
    });

    it("jumps to the chosen message and then closes the panel", async () => {
        // given
        stubSearch({ results: [makeResult()], total: 1 });
        const user = userEvent.setup();
        const { onClose, onJump } = renderPanel();
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "golden");

        // when
        await user.click(await screen.findByRole("button", { name: /Beatrice/ }));

        // then
        expect(onJump).toHaveBeenCalledWith("msg-1", "2026-08-02T10:30:00Z");
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("closes when the close control is pressed", async () => {
        // given
        stubSearch();
        const user = userEvent.setup();
        const { onClose } = renderPanel();

        // when
        await user.click(screen.getByRole("button", { name: "Close" }));

        // then
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("closes on a click outside the drawer but not on a click inside it", async () => {
        // given
        stubSearch();
        const user = userEvent.setup();
        const { onClose } = renderPanel();
        const drawer = screen.getByRole("dialog", { name: "Search messages" });

        // when
        await user.click(drawer);
        const clicksFromInside = onClose.mock.calls.length;
        await user.click(drawer.parentElement as HTMLElement);

        // then
        expect(clicksFromInside).toBe(0);
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("hides the pager until there is at least one match", async () => {
        // given
        stubSearch({ results: [], total: 0 });
        const user = userEvent.setup();
        renderPanel();

        // when
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "golden");

        // then
        await screen.findByText("No matching messages.");
        expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
    });

    it("pages the search offset forward when the next page is requested", async () => {
        // given
        stubSearch({ results: [makeResult()], total: 65 });
        const user = userEvent.setup();
        renderPanel();
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "golden");
        expect(await screen.findByText("1-30 of 65")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        await waitFor(() => {
            expect(useRoomMessageSearch).toHaveBeenLastCalledWith(roomId, "golden", 30, 30, true);
        });
        expect(screen.getByText("31-60 of 65")).toBeInTheDocument();
    });

    it("returns to the first page when the term is edited", async () => {
        // given
        stubSearch({ results: [makeResult()], total: 65 });
        const user = userEvent.setup();
        renderPanel();
        const input = screen.getByPlaceholderText("Search this conversation...");
        await user.type(input, "golden");
        await screen.findByText("1-30 of 65");
        await user.click(screen.getByRole("button", { name: "Next" }));
        await screen.findByText("31-60 of 65");

        // when
        await user.type(input, "!");

        // then
        await waitFor(() => {
            expect(useRoomMessageSearch).toHaveBeenLastCalledWith(roomId, "golden!", 30, 0, true);
        });
        expect(screen.getByText("1-30 of 65")).toBeInTheDocument();
    });

    it("jumps straight to the last page when the last control is used", async () => {
        // given
        stubSearch({ results: [makeResult()], total: 65 });
        const user = userEvent.setup();
        renderPanel();
        await user.type(screen.getByPlaceholderText("Search this conversation..."), "golden");
        await screen.findByText("1-30 of 65");

        // when
        await user.click(screen.getByRole("button", { name: "Last »" }));

        // then
        await waitFor(() => {
            expect(useRoomMessageSearch).toHaveBeenLastCalledWith(roomId, "golden", 30, 60, true);
        });
        expect(screen.getByText("61-65 of 65")).toBeInTheDocument();
    });
});
