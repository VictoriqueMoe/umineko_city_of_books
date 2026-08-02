import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Quote } from "../../../types/api";
import { ResponseEditor } from "./ResponseEditor";

const { createResponse, pickerQuote } = vi.hoisted(() => ({
    createResponse: vi.fn(() => Promise.resolve()),
    pickerQuote: {
        text: "The seven stakes of purgatory are the culprits.",
        textHtml: "",
        characterId: "battler",
        character: "Battler",
        audioId: "ep2_0042",
        episode: 2,
        contentType: "dialogue",
        hasRedTruth: true,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 42,
    },
}));

vi.mock("../../../api/mutations/theory", () => ({
    useCreateResponse: () => ({ mutateAsync: createResponse }),
}));

vi.mock("../../truth/TruthPicker/TruthPicker", () => ({
    TruthPicker: ({ isOpen, onSelect }: { isOpen: boolean; onSelect: (quote: Quote, lang: string) => void }) =>
        isOpen ? (
            <button type="button" onClick={() => onSelect(pickerQuote, "en")}>
                pick the stakes
            </button>
        ) : null,
}));

describe("ResponseEditor", () => {
    beforeEach(() => {
        createResponse.mockResolvedValue(undefined);
    });

    it("offers both sides of the debate for a fresh response", () => {
        // given
        const onCreated = vi.fn();

        // when
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={onCreated} />);

        // then
        expect(screen.getByRole("heading", { name: "Add your response" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /With love, it can be seen/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Without love, it cannot be seen/ })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("State your argument...")).toBeInTheDocument();
    });

    it("uses the wording of the series it belongs to", () => {
        // given
        const series = "higurashi";

        // when
        renderWithProviders(<ResponseEditor theoryId="t1" series={series} onCreated={vi.fn()} />);

        // then
        expect(screen.getByRole("button", { name: /Nipah~!/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Auau~!/ })).toBeInTheDocument();
    });

    it("hides the side choice when it is answering another response", () => {
        // given
        const parentId = "r1";

        // when
        renderWithProviders(
            <ResponseEditor theoryId="t1" parentId={parentId} inheritedSide="with_love" onCreated={vi.fn()} />,
        );

        // then
        expect(screen.getByRole("heading", { name: "Reply" })).toBeInTheDocument();
        expect(screen.getByPlaceholderText("Write your reply...")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /With love, it can be seen/ })).not.toBeInTheDocument();
    });

    it("keeps the submit button disabled until a side and a body are both chosen", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={vi.fn()} />);

        // when
        await user.type(screen.getByPlaceholderText("State your argument..."), "Kanon was never there.");

        // then
        expect(screen.getByRole("button", { name: "Submit Response" })).toBeDisabled();
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        expect(screen.getByRole("button", { name: "Submit Response" })).toBeEnabled();
    });

    it("submits the chosen side with the trimmed body and clears the editor", async () => {
        // given
        const onCreated = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={onCreated} />);

        // when
        await user.click(screen.getByRole("button", { name: /Without love, it cannot be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "  Kanon was never there.  ");
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // then
        expect(createResponse).toHaveBeenCalledExactlyOnceWith({
            parent_id: undefined,
            side: "without_love",
            body: "Kanon was never there.",
            evidence: [],
        });
        expect(onCreated).toHaveBeenCalledOnce();
        expect(screen.getByPlaceholderText("State your argument...")).toHaveValue("");
    });

    it("asks for the side again once a top level response has been posted", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "Agreed.");
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // when
        await user.type(screen.getByPlaceholderText("State your argument..."), "Another thought.");

        // then
        expect(screen.getByRole("button", { name: "Submit Response" })).toBeDisabled();
    });

    it("keeps the inherited side once a reply has been posted", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(
            <ResponseEditor theoryId="t1" parentId="r1" inheritedSide="without_love" onCreated={vi.fn()} />,
        );
        await user.type(screen.getByPlaceholderText("Write your reply..."), "Denied.");
        await user.click(screen.getByRole("button", { name: "Reply" }));

        // when
        await user.type(screen.getByPlaceholderText("Write your reply..."), "Still denied.");

        // then
        expect(createResponse).toHaveBeenCalledWith({
            parent_id: "r1",
            side: "without_love",
            body: "Denied.",
            evidence: [],
        });
        expect(screen.getByRole("button", { name: "Reply" })).toBeEnabled();
    });

    it("shows the reason the server gave for rejecting the response", async () => {
        // given
        const onCreated = vi.fn();
        createResponse.mockRejectedValue(new Error("Your response tripped the banned words filter"));
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={onCreated} />);

        // when
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "Agreed.");
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // then
        expect(screen.getByText("Your response tripped the banned words filter")).toBeInTheDocument();
        expect(onCreated).not.toHaveBeenCalled();
        expect(screen.getByPlaceholderText("State your argument...")).toHaveValue("Agreed.");
    });

    it("falls back to a generic message when the failure carries no message", async () => {
        // given
        createResponse.mockRejectedValue("the golden truth denies it");
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={vi.fn()} />);

        // when
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "Agreed.");
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // then
        expect(screen.getByText("Failed to submit response.")).toBeInTheDocument();
    });

    it("shows a submitting label while the response is in flight", async () => {
        // given
        let release: () => void = () => {};
        createResponse.mockImplementation(
            () =>
                new Promise<void>(resolve => {
                    release = resolve;
                }),
        );
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "Agreed.");

        // when
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // then
        expect(screen.getByRole("button", { name: "Submitting..." })).toBeDisabled();
        release();
        await waitFor(() => expect(screen.getByRole("button", { name: "Submit Response" })).toBeInTheDocument());
    });

    it("attaches a quote chosen from the picker to the response", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "Agreed.");

        // when
        await user.click(screen.getByRole("button", { name: "+ Attach Evidence" }));
        await user.click(screen.getByRole("button", { name: "pick the stakes" }));
        await user.type(screen.getByPlaceholderText("Why is this relevant?"), "Said in red");
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // then
        expect(createResponse).toHaveBeenCalledWith(
            expect.objectContaining({
                evidence: [{ audio_id: "ep2_0042", quote_index: undefined, note: "Said in red", lang: "en" }],
            }),
        );
    });

    it("clears the attached evidence once the response is posted", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<ResponseEditor theoryId="t1" onCreated={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /With love, it can be seen/ }));
        await user.type(screen.getByPlaceholderText("State your argument..."), "Agreed.");
        await user.click(screen.getByRole("button", { name: "+ Attach Evidence" }));
        await user.click(screen.getByRole("button", { name: "pick the stakes" }));

        // when
        await user.click(screen.getByRole("button", { name: "Submit Response" }));

        // then
        expect(screen.queryByText(pickerQuote.text)).not.toBeInTheDocument();
    });
});
