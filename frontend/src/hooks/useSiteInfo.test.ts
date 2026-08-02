import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { providerWrapper } from "../test-utils/render";
import { useSiteInfo } from "./useSiteInfo";

describe("useSiteInfo", () => {
    it("returns the site info the provider holds", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { site_name: "City of Books", version: "6.10.0" } });

        // when
        const { result } = renderHook(() => useSiteInfo(), { wrapper });

        // then
        expect(result.current.site_name).toBe("City of Books");
        expect(result.current.version).toBe("6.10.0");
    });

    it("keeps the defaults for fields the caller did not override", () => {
        // given
        const wrapper = providerWrapper({ siteInfo: { maintenance_mode: true } });

        // when
        const { result } = renderHook(() => useSiteInfo(), { wrapper });

        // then
        expect(result.current.maintenance_mode).toBe(true);
        expect(result.current.registration_type).toBe("open");
        expect(result.current.default_theme).toBe("featherine");
    });

    it("exposes feature switches that gate whole areas of the app", () => {
        // given
        const wrapper = providerWrapper({
            siteInfo: { voice_enabled: false, email_enabled: false, turnstile_enabled: true },
        });

        // when
        const { result } = renderHook(() => useSiteInfo(), { wrapper });

        // then
        expect(result.current.voice_enabled).toBe(false);
        expect(result.current.email_enabled).toBe(false);
        expect(result.current.turnstile_enabled).toBe(true);
    });

    it("exposes empty leaderboard lists rather than undefined", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useSiteInfo(), { wrapper });

        // then
        expect(result.current.top_detective_ids).toEqual([]);
        expect(result.current.vanity_roles).toEqual([]);
        expect(result.current.vanity_role_assignments).toEqual({});
    });

    it("throws when it is used outside SiteInfoProvider", () => {
        // given
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useSiteInfo());

        // then
        expect(attempt).toThrow("useSiteInfo must be used within SiteInfoProvider");
        consoleError.mockRestore();
    });
});
