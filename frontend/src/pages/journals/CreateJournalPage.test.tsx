import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { CreateJournalPage } from "./CreateJournalPage";

const { useCreateJournal, navigate } = vi.hoisted(() => ({ useCreateJournal: vi.fn(), navigate: vi.fn() }));

vi.mock("../../api/mutations/journal", () => ({ useCreateJournal }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const writer = makeUser({ id: "writer-1", username: "battler", display_name: "Battler" });

function stubCreate(create?: () => Promise<{ id: string }>) {
    const createAsync = vi.fn(create ?? (() => Promise.resolve({ id: "journal-new" })));
    useCreateJournal.mockReturnValue({ mutateAsync: createAsync });

    return { createAsync };
}

function renderPage() {
    return renderWithProviders(<CreateJournalPage />, { user: writer, route: "/journals/new" });
}

describe("CreateJournalPage", () => {
    it("explains what the reader is about to set up", () => {
        // given
        stubCreate();

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Start a Reading Journal" })).toBeInTheDocument();
        expect(screen.getByText(/After creating the journal, you'll add your first entry\./)).toBeInTheDocument();
    });

    it("refuses to submit until the journal has a title", () => {
        // given
        stubCreate();

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "Create Journal" })).toBeDisabled();
    });

    it("keeps the submit button disabled for a title of only spaces", async () => {
        // given
        stubCreate();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(screen.getByPlaceholderText("e.g. My first Umineko read-through"), "   ");

        // then
        expect(screen.getByRole("button", { name: "Create Journal" })).toBeDisabled();
    });

    it("starts the journal on the general work", async () => {
        // given
        const { createAsync } = stubCreate();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(screen.getByPlaceholderText("e.g. My first Umineko read-through"), "  My read-through  ");
        await user.click(screen.getByRole("button", { name: "Create Journal" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({ title: "My read-through", work: "general" });
    });

    it("sends the work the writer picked", async () => {
        // given
        const { createAsync } = stubCreate();
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(screen.getByPlaceholderText("e.g. My first Umineko read-through"), "Rokkenjima Notes");
        await user.selectOptions(screen.getByRole("combobox"), "higurashi");
        await user.click(screen.getByRole("button", { name: "Create Journal" }));

        // then
        expect(createAsync).toHaveBeenCalledWith({ title: "Rokkenjima Notes", work: "higurashi" });
    });

    it("sends the writer straight on to the first entry", async () => {
        // given
        stubCreate(() => Promise.resolve({ id: "journal-42" }));
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(screen.getByPlaceholderText("e.g. My first Umineko read-through"), "Rokkenjima Notes");
        await user.click(screen.getByRole("button", { name: "Create Journal" }));

        // then
        await waitFor(() => {
            expect(navigate).toHaveBeenCalledWith("/journals/journal-42/entry/new");
        });
    });

    it("reports why the journal could not be created", async () => {
        // given
        stubCreate(() => Promise.reject(new Error("The witch forbids it")));
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(screen.getByPlaceholderText("e.g. My first Umineko read-through"), "Rokkenjima Notes");
        await user.click(screen.getByRole("button", { name: "Create Journal" }));

        // then
        expect(await screen.findByText("The witch forbids it")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("offers every readable work to journal about", () => {
        // given
        stubCreate();

        // when
        renderPage();

        // then
        const options = screen.getAllByRole("option").map(o => o.textContent);
        expect(options).toEqual(["General", "Umineko", "Higurashi", "Ciconia", "Higanbana", "Rose Guns Days"]);
    });
});
