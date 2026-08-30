import { describe, expect, it } from "vitest";
import { getLastLocation, isAuthPath, recordLocation } from "./lastLocation";

describe("isAuthPath", () => {
    const authSpellings = [
        "/login",
        "/login/",
        "/LOGIN",
        "/Login/",
        "/login//",
        "/login///",
        "/%6Cogin",
        "/log%69n",
        "/forgot-password",
        "/reset-password//",
        "/set-email",
        "/verify-email",
    ];

    for (const pathname of authSpellings) {
        it(`treats ${pathname} as an auth path, since react-router routes it to an auth page`, () => {
            expect(isAuthPath(pathname)).toBe(true);
        });
    }

    const browsablePaths = ["/", "/users", "/gallery/umineko", "/theory/abc", "/logins", "/login-help"];

    for (const pathname of browsablePaths) {
        it(`treats ${pathname} as browsable`, () => {
            expect(isAuthPath(pathname)).toBe(false);
        });
    }

    it("fails closed on a malformed percent escape rather than recording it", () => {
        expect(isAuthPath("/%")).toBe(true);
    });
});

describe("recordLocation", () => {
    it("keeps the full location including search and hash", () => {
        recordLocation("/gallery/umineko", "?sort=new", "#art-42");

        expect(getLastLocation()).toBe("/gallery/umineko?sort=new#art-42");
    });

    it("leaves the previous location untouched when navigating to an auth page", () => {
        recordLocation("/theory/abc", "?tab=comments", "");
        recordLocation("/login", "", "");

        expect(getLastLocation()).toBe("/theory/abc?tab=comments");
    });

    it("leaves the previous location untouched for an auth page spelled unusually", () => {
        recordLocation("/theory/abc", "", "");
        recordLocation("/login//", "", "");

        expect(getLastLocation()).toBe("/theory/abc");
    });
});
