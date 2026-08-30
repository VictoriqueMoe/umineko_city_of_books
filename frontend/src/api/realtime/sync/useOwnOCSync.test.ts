import { act, renderHook } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import type { OCSummary } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { useOwnOCSync } from "./useOwnOCSync";

const viewerId = "11111111-1111-1111-1111-111111111111";

function makeSummary(id: string, name: string): OCSummary {
    return { id, name, series: "umineko" };
}

function changed(action: string, oc: OCSummary): RealtimeEvent {
    return { type: "user_ocs_changed", data: { action, oc } } as unknown as RealtimeEvent;
}

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

let queryClient: QueryClient;

function seed(userId: string, summaries: OCSummary[]): void {
    queryClient.setQueryData<OCSummary[]>(queryKeys.oc.userSummaries(userId), summaries);
}

function read(userId: string): OCSummary[] | undefined {
    return queryClient.getQueryData<OCSummary[]>(queryKeys.oc.userSummaries(userId));
}

function mount(userId: string, currentUserId?: string) {
    return renderHook(() => useOwnOCSync(userId, currentUserId), { wrapper: providerWrapper({ queryClient }) });
}

beforeEach(() => {
    queryClient = createTestQueryClient();
});

describe("useOwnOCSync", () => {
    it("appends a newly created oc in alphabetical order", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice"), makeSummary("v", "Virgilia")]);
        mount(viewerId, viewerId);

        // when
        emit(changed("created", makeSummary("l", "Lambdadelta")));

        // then
        expect(read(viewerId)?.map(item => item.name)).toEqual(["Beatrice", "Lambdadelta", "Virgilia"]);
    });

    it("ignores a created oc that is already in the list", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        mount(viewerId, viewerId);

        // when
        emit(changed("created", makeSummary("b", "Beatrice renamed")));

        // then
        expect(read(viewerId)).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("replaces the matching oc when one is updated", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        mount(viewerId, viewerId);

        // when
        emit(changed("updated", makeSummary("b", "Beato")));

        // then
        expect(read(viewerId)).toEqual([makeSummary("b", "Beato")]);
    });

    it("drops the matching oc when one is deleted", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        mount(viewerId, viewerId);

        // when
        emit(changed("deleted", makeSummary("b", "Beatrice")));

        // then
        expect(read(viewerId)).toEqual([]);
    });

    it("leaves the list alone for an action it does not recognise", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        mount(viewerId, viewerId);

        // when
        emit(changed("archived", makeSummary("b", "Beatrice")));

        // then
        expect(read(viewerId)).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("ignores socket messages about anything other than the viewer's ocs", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        mount(viewerId, viewerId);

        // when
        emit({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(read(viewerId)).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("leaves the list alone when the owner is not the viewer", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        mount(viewerId, "someone-else");

        // when
        emit(changed("deleted", makeSummary("b", "Beatrice")));

        // then
        expect(read(viewerId)).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("leaves the cache alone when there is no owner id", () => {
        // given
        mount("", "");

        // when
        emit(changed("created", makeSummary("b", "Beatrice")));

        // then
        expect(read("")).toBeUndefined();
    });

    it("stops listening when the hook unmounts", () => {
        // given
        seed(viewerId, [makeSummary("b", "Beatrice")]);
        const view = mount(viewerId, viewerId);

        // when
        view.unmount();
        emit(changed("deleted", makeSummary("b", "Beatrice")));

        // then
        expect(read(viewerId)).toEqual([makeSummary("b", "Beatrice")]);
    });
});
