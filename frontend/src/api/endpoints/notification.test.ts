import { beforeEach, describe, vi } from "vitest";
import * as api from "./notification";
import { fetchMock, postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("the notification API", () => {
    const cases: RequestCase[] = [
        {
            name: "getNotifications defaults to the first page of twenty",
            call: () => api.getNotifications({}),
            transport: fetchMock,
            request: ["/notifications?limit=20"],
        },
        {
            name: "getNotifications pages through the list",
            call: () => api.getNotifications({ limit: 5, offset: 10 }),
            transport: fetchMock,
            request: ["/notifications?limit=5&offset=10"],
        },
        {
            name: "markNotificationRead posts to the numbered notification with no body",
            call: () => api.markNotificationRead(3),
            transport: postMock,
            request: ["/notifications/3/read", undefined],
        },
        {
            name: "markAllNotificationsRead posts to the collection with no body",
            call: () => api.markAllNotificationsRead(),
            transport: postMock,
            request: ["/notifications/read", undefined],
        },
        {
            name: "getUnreadCount reads the unread counter",
            call: () => api.getUnreadCount(),
            transport: fetchMock,
            request: ["/notifications/unread-count"],
        },
    ];

    runRequestCases(cases);
});
