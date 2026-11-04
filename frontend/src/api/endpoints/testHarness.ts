import { expect, it, vi, type Mock } from "vitest";
import {
    apiDelete,
    apiDeleteWithBody,
    apiFetch,
    apiFetchText,
    apiPatch,
    apiPost,
    apiPostFormData,
    apiPut,
} from "../client";

export interface RequestCase {
    name: string;
    call: () => Promise<unknown>;
    transport: Mock;
    request: unknown[];
    strict?: boolean;
}

export const fetchMock = vi.mocked(apiFetch);
export const fetchTextMock = vi.mocked(apiFetchText);
export const postMock = vi.mocked(apiPost);
export const putMock = vi.mocked(apiPut);
export const patchMock = vi.mocked(apiPatch);
export const deleteMock = vi.mocked(apiDelete);
export const deleteWithBodyMock = vi.mocked(apiDeleteWithBody);
export const postFormDataMock = vi.mocked(apiPostFormData);

export function resetTransports(): void {
    fetchMock.mockResolvedValue({});
    fetchTextMock.mockResolvedValue("");
    postMock.mockResolvedValue({});
    putMock.mockResolvedValue({});
    patchMock.mockResolvedValue({});
    deleteMock.mockResolvedValue({});
    deleteWithBodyMock.mockResolvedValue({});
    postFormDataMock.mockResolvedValue({});
}

export function lastFormData(): FormData {
    const calls = postFormDataMock.mock.calls;
    return calls[calls.length - 1][1];
}

export function runRequestCases(cases: RequestCase[]): void {
    const looseCases = cases.filter(entry => !entry.strict);
    const strictCases = cases.filter(entry => entry.strict);

    it.each(looseCases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it.each(strictCases)("$name", async ({ call, transport, request }) => {
        // given the call and the exact request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport.mock.lastCall).toStrictEqual(request);
    });
}
