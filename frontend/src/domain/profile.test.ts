import { describe, expect, it } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import {
    deriveGender,
    derivePronouns,
    formatDate,
    formatDOBWithAge,
    socialEntries,
    socialHref,
    socialUrl,
} from "./profile";

const now = new Date("2026-08-02T12:00:00Z");

describe("deriveGender", () => {
    it("keeps a gender it recognises as it is", () => {
        // given
        const gender = "Male";

        // when
        const derived = deriveGender(gender);

        // then
        expect(derived.gender).toBe("Male");
        expect(derived.customGender).toBe("");
    });

    it("moves a gender it does not recognise into the custom box", () => {
        // given
        const gender = "Agender";

        // when
        const derived = deriveGender(gender);

        // then
        expect(derived.gender).toBe("Custom");
        expect(derived.customGender).toBe("Agender");
    });

    it("treats a blank gender as prefer not to say", () => {
        // given
        const gender = "";

        // when
        const derived = deriveGender(gender);

        // then
        expect(derived.gender).toBe("Prefer not to say");
        expect(derived.customGender).toBe("");
    });
});

describe("derivePronouns", () => {
    it("does not call pronouns custom when they match the gender default", () => {
        // given
        const profile = makeUser({ gender: "Female", pronoun_subject: "she", pronoun_possessive: "her" });

        // when
        const derived = derivePronouns(profile);

        // then
        expect(derived.isCustom).toBe(false);
    });

    it("spots pronouns that were customised away from the gender default", () => {
        // given
        const profile = makeUser({ gender: "Male", pronoun_subject: "xe", pronoun_possessive: "xyr" });

        // when
        const derived = derivePronouns(profile);

        // then
        expect(derived.isCustom).toBe(true);
        expect(derived.subject).toBe("xe");
        expect(derived.possessive).toBe("xyr");
    });

    it("measures the pronouns of an unrecognised gender against the neutral default", () => {
        // given
        const profile = makeUser({ gender: "Agender", pronoun_subject: "they", pronoun_possessive: "their" });

        // when
        const derived = derivePronouns(profile);

        // then
        expect(derived.isCustom).toBe(false);
    });
});

describe("formatDate", () => {
    it("spells the month out in full", () => {
        // given
        const iso = "2026-01-15T10:00:00Z";

        // when
        const formatted = formatDate(iso);

        // then
        expect(formatted).toContain("January");
        expect(formatted).toContain("15");
        expect(formatted).toContain("2026");
    });

    it("says nothing at all about a date it cannot read", () => {
        // given
        const iso = "sometime";

        // when
        const formatted = formatDate(iso);

        // then
        expect(formatted).toBe("");
    });
});

describe("formatDOBWithAge", () => {
    it("works the age out from the date of birth", () => {
        // given
        const dob = "1995-07-15";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toMatch(/\(31 years old\)$/);
        expect(formatted).toContain("July");
        expect(formatted).toContain("1995");
    });

    it("uses the singular year for a one year old", () => {
        // given
        const dob = "2025-01-01";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toMatch(/\(1 year old\)$/);
    });

    it("counts the birthday itself as the new age", () => {
        // given
        const dob = "2000-08-02";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toMatch(/\(26 years old\)$/);
    });

    it("still counts last year's age the day before the birthday", () => {
        // given
        const dob = "2000-08-03";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toMatch(/\(25 years old\)$/);
    });

    it("still counts last year's age when the birthday month is yet to come", () => {
        // given
        const dob = "2000-09-01";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toMatch(/\(25 years old\)$/);
    });

    it("leaves the age off a date of birth that has not happened yet", () => {
        // given
        const dob = "2030-01-01";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).not.toMatch(/years old/);
        expect(formatted).toContain("January");
        expect(formatted).toContain("01");
        expect(formatted).toContain("2030");
    });

    it("hands back a date of birth it cannot split into three parts", () => {
        // given
        const dob = "sometime";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toBe("sometime");
    });

    it("leaves a date of birth whose parts are not numbers exactly as it stands", () => {
        // given
        const dob = "1995-July-15";

        // when
        const formatted = formatDOBWithAge(dob, now);

        // then
        expect(formatted).toBe("1995-July-15");
    });
});

describe("socialUrl", () => {
    it("sends a bare twitter handle out to x", () => {
        expect(socialUrl("social_twitter", "beato")).toBe("https://x.com/beato");
    });

    it("sends a bare github handle out to github", () => {
        expect(socialUrl("social_github", "beato")).toBe("https://github.com/beato");
    });

    it("sends a bare bluesky handle out to bsky", () => {
        expect(socialUrl("social_bluesky", "beato.bsky.social")).toBe("https://bsky.app/profile/beato.bsky.social");
    });

    it("drops the leading at sign from a bluesky handle", () => {
        expect(socialUrl("social_bluesky", "@beato.bsky.social")).toBe("https://bsky.app/profile/beato.bsky.social");
    });

    it("puts a tumblr handle on its own subdomain", () => {
        expect(socialUrl("social_tumblr", "beato")).toBe("https://beato.tumblr.com");
    });

    it("sends a bare waifulist handle out to waifulist", () => {
        expect(socialUrl("social_waifulist", "beato")).toBe("https://waifulist.moe/beato");
    });

    it("keeps a waifulist path on the waifulist domain the player gave", () => {
        expect(socialUrl("social_waifulist", "waifulist.moe/list/beato")).toBe("https://waifulist.moe/list/beato");
    });

    it("leaves a full address exactly as the player gave it", () => {
        expect(socialUrl("social_twitter", "https://x.com/goldenwitch")).toBe("https://x.com/goldenwitch");
    });

    it("leaves a full bluesky address exactly as the player gave it", () => {
        expect(socialUrl("social_bluesky", "https://bsky.app/profile/beato.bsky.social")).toBe(
            "https://bsky.app/profile/beato.bsky.social",
        );
    });

    it("hands back a service it knows no address for untouched", () => {
        expect(socialUrl("social_discord", "beato#0001")).toBe("beato#0001");
    });
});

describe("socialHref", () => {
    it("turns a shared email address into a mail link", () => {
        // given
        const entry = { key: "email", label: "Email", value: "beato@example.com" };

        // then
        expect(socialHref(entry)).toBe("mailto:beato@example.com");
    });

    it("adds the missing scheme to a bare website", () => {
        // given
        const entry = { key: "website", label: "Website", value: "witchs.moe" };

        // then
        expect(socialHref(entry)).toBe("https://witchs.moe");
    });

    it("leaves a website the player already wrote in full alone", () => {
        // given
        const entry = { key: "website", label: "Website", value: "http://witchs.moe" };

        // then
        expect(socialHref(entry)).toBe("http://witchs.moe");
    });

    it("falls back to the service address for everything else", () => {
        // given
        const entry = { key: "social_github", label: "GitHub", value: "beato" };

        // then
        expect(socialHref(entry)).toBe("https://github.com/beato");
    });
});

describe("socialEntries", () => {
    it("lists only the services the player filled in, in label order", () => {
        // given
        const profile = makeUser({ social_twitter: "beato", social_github: "beato", social_bluesky: "beato" });

        // when
        const entries = socialEntries(profile);

        // then
        expect(entries.map(entry => entry.key)).toEqual(["social_twitter", "social_github", "social_bluesky"]);
        expect(entries.map(entry => entry.label)).toEqual(["Twitter / X", "GitHub", "Bluesky"]);
    });

    it("puts the website and the email last, in that order", () => {
        // given
        const profile = makeUser({ social_discord: "beato#0001", website: "witchs.moe", email: "beato@example.com" });

        // when
        const entries = socialEntries(profile);

        // then
        expect(entries.map(entry => entry.key)).toEqual(["social_discord", "website", "email"]);
    });

    it("leaves out an email the player never shared", () => {
        // given
        const profile = makeUser({ website: "witchs.moe" });

        // when
        const entries = socialEntries(profile);

        // then
        expect(entries.map(entry => entry.key)).toEqual(["website"]);
    });

    it("hands back nothing when the player shared nothing", () => {
        // given
        const profile = makeUser();

        // when
        const entries = socialEntries(profile);

        // then
        expect(entries).toEqual([]);
    });
});
