import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { providerWrapper } from "../test-utils/render";
import { usePageTitle } from "./usePageTitle";

describe("usePageTitle", () => {
    it("appends the site name to the given title", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { site_name: "When They Cry" } });

        // when
        renderHook(() => usePageTitle("Theories"), { wrapper });

        // then
        expect(document.title).toBe("Theories | When They Cry");
    });

    it("shows the site name on its own when no title is given", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { site_name: "When They Cry" } });

        // when
        renderHook(() => usePageTitle(), { wrapper });

        // then
        expect(document.title).toBe("When They Cry");
    });

    it("treats an empty title as no title at all", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { site_name: "When They Cry" } });

        // when
        renderHook(() => usePageTitle(""), { wrapper });

        // then
        expect(document.title).toBe("When They Cry");
    });

    it("prefixes the number of unread notifications", () => {
        // given
        const wrapper = providerWrapper({
            siteInfo: { site_name: "When They Cry" },
            notification: { unreadCount: 3 },
        });

        // when
        renderHook(() => usePageTitle("Theories"), { wrapper });

        // then
        expect(document.title).toBe("(3) Theories | When They Cry");
    });

    it("prefixes the unread count even without a page title", () => {
        // given
        const wrapper = providerWrapper({
            siteInfo: { site_name: "When They Cry" },
            notification: { unreadCount: 12 },
        });

        // when
        renderHook(() => usePageTitle(), { wrapper });

        // then
        expect(document.title).toBe("(12) When They Cry");
    });

    it("leaves the title unprefixed when nothing is unread", () => {
        // given
        const wrapper = providerWrapper({
            siteInfo: { site_name: "When They Cry" },
            notification: { unreadCount: 0 },
        });

        // when
        renderHook(() => usePageTitle("Theories"), { wrapper });

        // then
        expect(document.title).toBe("Theories | When They Cry");
    });

    it("retitles the document when the page title changes", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { site_name: "When They Cry" } });
        const { rerender } = renderHook(({ title }: { title?: string }) => usePageTitle(title), {
            wrapper,
            initialProps: { title: "Theories" },
        });

        // when
        rerender({ title: "Favourites" });

        // then
        expect(document.title).toBe("Favourites | When They Cry");
    });

    it("uses the site name the provider supplies", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { site_name: "City of Books" } });

        // when
        renderHook(() => usePageTitle("Theories"), { wrapper });

        // then
        expect(document.title).toBe("Theories | City of Books");
    });
});
