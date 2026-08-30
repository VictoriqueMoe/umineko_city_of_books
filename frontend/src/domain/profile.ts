import type { UserProfile } from "../types/api";
import { parseServerDate } from "../utils/time";

export const GENDER_OPTIONS = ["Prefer not to say", "Male", "Female", "Custom"];

export const SOCIAL_LABELS: Record<string, string> = {
    social_twitter: "Twitter / X",
    social_discord: "Discord",
    social_waifulist: "WaifuList",
    social_tumblr: "Tumblr",
    social_github: "GitHub",
    social_bluesky: "Bluesky",
};

export interface SocialEntry {
    key: string;
    label: string;
    value: string;
}

export const PRONOUN_DEFAULTS: Record<string, { subject: string; possessive: string }> = {
    Male: { subject: "he", possessive: "his" },
    Female: { subject: "she", possessive: "her" },
    "Prefer not to say": { subject: "they", possessive: "their" },
    Custom: { subject: "they", possessive: "their" },
};

export interface DerivedGender {
    gender: string;
    customGender: string;
}

export interface DerivedPronouns {
    subject: string;
    possessive: string;
    isCustom: boolean;
}

export function deriveGender(genderValue: string): DerivedGender {
    const g = genderValue || "";
    if (GENDER_OPTIONS.includes(g)) {
        return { gender: g, customGender: "" };
    }
    if (g) {
        return { gender: "Custom", customGender: g };
    }
    return { gender: "Prefer not to say", customGender: "" };
}

export function derivePronouns(profile: UserProfile): DerivedPronouns {
    const subject = profile.pronoun_subject || "they";
    const possessive = profile.pronoun_possessive || "their";
    const defaults = PRONOUN_DEFAULTS[GENDER_OPTIONS.includes(profile.gender) ? profile.gender : "Custom"];
    const isCustom =
        (!!profile.pronoun_subject && subject !== defaults.subject) ||
        (!!profile.pronoun_possessive && possessive !== defaults.possessive);
    return { subject, possessive, isCustom };
}

export function formatDate(iso: string): string {
    const d = parseServerDate(iso);
    if (!d) {
        return "";
    }
    return d.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" });
}

export function formatDOBWithAge(value: string, now: Date): string {
    const parts = value.split("-");
    if (parts.length !== 3) {
        return value;
    }

    const year = Number(parts[0]);
    const month = Number(parts[1]);
    const day = Number(parts[2]);

    if (!Number.isInteger(year) || !Number.isInteger(month) || !Number.isInteger(day)) {
        return value;
    }

    const parsed = new Date(Date.UTC(year, month - 1, day));
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }

    let age = now.getUTCFullYear() - year;
    if (now.getUTCMonth() + 1 < month || (now.getUTCMonth() + 1 === month && now.getUTCDate() < day)) {
        age -= 1;
    }

    const ageLabel = age === 1 ? "year old" : "years old";
    const formatted = parsed.toLocaleDateString(undefined, {
        year: "numeric",
        month: "long",
        day: "2-digit",
        timeZone: "UTC",
    });

    if (age < 0) {
        return formatted;
    }

    return `${formatted} (${age} ${ageLabel})`;
}

export function socialUrl(key: string, value: string): string {
    if (value.startsWith("http://") || value.startsWith("https://")) {
        return value;
    }
    switch (key) {
        case "social_twitter":
            return `https://x.com/${value}`;
        case "social_github":
            return `https://github.com/${value}`;
        case "social_bluesky":
            return `https://bsky.app/profile/${value.replace(/^@/, "")}`;
        case "social_tumblr":
            return `https://${value}.tumblr.com`;
        case "social_waifulist":
            return value.includes("/") ? `https://${value}` : `https://waifulist.moe/${value}`;
        default:
            return value;
    }
}

export function socialHref(entry: SocialEntry): string {
    if (entry.key === "website") {
        return entry.value.startsWith("http") ? entry.value : `https://${entry.value}`;
    }

    if (entry.key === "email") {
        return `mailto:${entry.value}`;
    }

    return socialUrl(entry.key, entry.value);
}

export function socialEntries(profile: UserProfile): SocialEntry[] {
    const entries = Object.entries(SOCIAL_LABELS)
        .map(([key, label]) => ({
            key,
            label,
            value: profile[key as keyof UserProfile] as string,
        }))
        .filter(entry => entry.value);

    if (profile.website) {
        entries.push({ key: "website", label: "Website", value: profile.website });
    }

    if (profile.email) {
        entries.push({ key: "email", label: "Email", value: profile.email });
    }

    return entries;
}
