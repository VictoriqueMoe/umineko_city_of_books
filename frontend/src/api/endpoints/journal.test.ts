import { beforeEach, describe, vi } from "vitest";
import * as api from "./journal";
import {
    deleteMock,
    fetchMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

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
            request: ["/journals/j-1/entries", { ...entryPayload, media: [] }],
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

describe("journal media", () => {
    const entryPayload = { title: "first twilight", body: "six chosen by the key", is_draft: false };
    const media = [{ upload_id: "upload-1", is_spoiler: true }];

    const cases: RequestCase[] = [
        {
            name: "attachJournalEntryMedia posts the finished upload to the entry",
            call: () => api.attachJournalEntryMedia("e-1", media[0]),
            transport: postMock,
            request: ["/journal-entries/e-1/media", media[0]],
        },
        {
            name: "createJournalEntry sends the entry and its finished uploads in one JSON request",
            call: () => api.createJournalEntry("j-1", entryPayload, media),
            transport: postMock,
            request: ["/journals/j-1/entries", { ...entryPayload, media }],
        },
        {
            name: "createJournalEntry sends an empty media list when nothing is attached",
            call: () => api.createJournalEntry("j-1", entryPayload),
            transport: postMock,
            request: ["/journals/j-1/entries", { ...entryPayload, media: [] }],
        },
        {
            name: "createJournalComment sends an entry comment and its finished uploads in one JSON request",
            call: () => api.createJournalComment("j-1", "", undefined, "e-1", media),
            transport: postMock,
            request: ["/journals/j-1/comments", { body: "", parent_id: undefined, entry_id: "e-1", media }],
        },
    ];

    runRequestCases(cases);
});
