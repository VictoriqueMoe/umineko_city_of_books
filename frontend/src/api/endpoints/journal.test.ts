import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./journal";
import {
    deleteMock,
    fetchMock,
    postFormDataMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

vi.mock("../../platform/capabilities", () => ({
    isNativeApp: () => false,
    clientPlatform: () => "web",
}));

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

describe("the journal API", () => {
    const journalPayload = { title: "Notes on the epitaph", work: "umineko" as const };
    const entryPayload = { title: "The first twilight", body: "six chosen by the key", is_draft: false };

    const cases: RequestCase[] = [
        {
            name: "getJournal reads a single journal",
            call: () => api.getJournal("j-1"),
            transport: fetchMock,
            request: ["/journals/j-1"],
        },
        {
            name: "createJournal posts the whole payload to the journal collection",
            call: () => api.createJournal(journalPayload),
            transport: postMock,
            request: ["/journals", journalPayload],
        },
        {
            name: "updateJournal puts the whole payload back under its own id",
            call: () => api.updateJournal("j-1", journalPayload),
            transport: putMock,
            request: ["/journals/j-1", journalPayload],
        },
        {
            name: "deleteJournal deletes the journal",
            call: () => api.deleteJournal("j-1"),
            transport: deleteMock,
            request: ["/journals/j-1"],
        },
        {
            name: "followJournal posts an empty body to the follow",
            call: () => api.followJournal("j-1"),
            transport: postMock,
            request: ["/journals/j-1/follow", {}],
        },
        {
            name: "unfollowJournal deletes the follow",
            call: () => api.unfollowJournal("j-1"),
            transport: deleteMock,
            request: ["/journals/j-1/follow"],
        },
        {
            name: "getJournalEntry interpolates the numbered entry under its journal",
            call: () => api.getJournalEntry("j-1", 3),
            transport: fetchMock,
            request: ["/journals/j-1/entries/3"],
        },
        {
            name: "createJournalEntry posts the entry to the journal's entry collection",
            call: () => api.createJournalEntry("j-1", entryPayload),
            transport: postMock,
            request: ["/journals/j-1/entries", entryPayload],
        },
        {
            name: "updateJournalEntry puts the entry back on the flat entry path",
            call: () => api.updateJournalEntry("e-1", entryPayload),
            transport: putMock,
            request: ["/journal-entries/e-1", entryPayload],
        },
        {
            name: "deleteJournalEntry deletes the entry",
            call: () => api.deleteJournalEntry("e-1"),
            transport: deleteMock,
            request: ["/journal-entries/e-1"],
        },
    ];

    runRequestCases(cases);
});

describe("journal entry media uploads", () => {
    it("uploadJournalEntryMedia posts the file under the media field", async () => {
        // given
        const file = new File(["x"], "entry.png", { type: "image/png" });

        // when
        await api.uploadJournalEntryMedia("e-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/journal-entries/e-1/media");
        expect(formData.get("media")).toBe(file);
    });
});
