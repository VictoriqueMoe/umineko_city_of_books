import { describe, expect, it } from "vitest";
import type { SiteSettings } from "../types/api";
import {
    boolValue,
    bytesToMB,
    chatbotOptInRoles,
    isEnabled,
    mbToBytes,
    mpToPixels,
    pixelsToMP,
    validateSiteSettings,
} from "./siteSettings";

const VALID: SiteSettings = {
    max_body_size: String(50 * 1024 * 1024),
    max_image_size: String(10 * 1024 * 1024),
    max_image_pixels: "24000000",
    max_video_size: String(20 * 1024 * 1024),
    max_general_size: String(20 * 1024 * 1024),
    min_password_length: "8",
    session_duration_days: "30",
    max_theories_per_day: "5",
    max_responses_per_day: "20",
};

const CHATBOT: SiteSettings = {
    ...VALID,
    chatbot_enabled: "true",
    chatbot_max_output_tokens: "2048",
};

interface StubVanityRole {
    id: string;
    label: string;
    permissions: string[];
}

describe("isEnabled", () => {
    it("reads only the exact string the settings table stores", () => {
        // given / when / then
        expect(isEnabled("true")).toBe(true);
        expect(isEnabled("false")).toBe(false);
        expect(isEnabled("TRUE")).toBe(false);
        expect(isEnabled("1")).toBe(false);
        expect(isEnabled("")).toBe(false);
    });

    it("treats a setting that was never written as off", () => {
        // given
        const missing = undefined;

        // when / then
        expect(isEnabled(missing)).toBe(false);
    });
});

describe("boolValue", () => {
    it("writes a toggle back as the string the settings table expects", () => {
        // given / when / then
        expect(boolValue(true)).toBe("true");
        expect(boolValue(false)).toBe("false");
    });

    it("round trips through isEnabled", () => {
        // given / when / then
        expect(isEnabled(boolValue(true))).toBe(true);
        expect(isEnabled(boolValue(false))).toBe(false);
    });
});

describe("bytesToMB", () => {
    it("shows the file size limits in megabytes", () => {
        // given
        const settings = { ...VALID };

        // when / then
        expect(bytesToMB(settings.max_image_size)).toBe("10");
        expect(bytesToMB(settings.max_body_size)).toBe("50");
    });

    it("treats a size that is not a number as zero", () => {
        // given
        const raw = "not a number";

        // when
        const mb = bytesToMB(raw);

        // then
        expect(mb).toBe("0");
    });

    it("treats a size that was never written as zero", () => {
        // given / when / then
        expect(bytesToMB(undefined)).toBe("0");
        expect(bytesToMB("")).toBe("0");
    });

    it("counts a megabyte as 1024 * 1024 bytes rather than a million", () => {
        // given
        const fiftyMebibytes = "52428800";

        // when
        const mb = bytesToMB(fiftyMebibytes);

        // then
        expect(mb).toBe("50");
        expect(mb).not.toBe("52");
    });

    it("rounds a part megabyte to the nearest whole one", () => {
        // given / when / then
        expect(bytesToMB("1500000")).toBe("1");
        expect(bytesToMB(String(1024 * 1024 + 1))).toBe("1");
        expect(bytesToMB(String(1024 * 1024 * 1.5))).toBe("2");
    });
});

describe("mbToBytes", () => {
    it("stores a size typed in megabytes as bytes", () => {
        // given
        const typed = "8";

        // when
        const bytes = mbToBytes(typed);

        // then
        expect(bytes).toBe(String(8 * 1024 * 1024));
    });

    it("treats a size that is not a number as zero", () => {
        // given / when / then
        expect(mbToBytes("not a number")).toBe("0");
        expect(mbToBytes("")).toBe("0");
    });

    it("keeps a fractional megabyte by rounding it to whole bytes", () => {
        // given
        const typed = "1.5";

        // when
        const bytes = mbToBytes(typed);

        // then
        expect(bytes).toBe("1572864");
    });
});

describe("pixelsToMP", () => {
    it("shows the image pixel ceiling in megapixels", () => {
        // given
        const settings = { ...VALID };

        // when
        const mp = pixelsToMP(settings.max_image_pixels);

        // then
        expect(mp).toBe("24");
    });

    it("counts a megapixel as a flat million pixels", () => {
        // given
        const pixels = "1048576";

        // when
        const mp = pixelsToMP(pixels);

        // then
        expect(mp).toBe("1.048576");
        expect(mp).not.toBe("1");
    });

    it("treats a pixel ceiling that is not a number as zero", () => {
        // given / when / then
        expect(pixelsToMP("not a number")).toBe("0");
        expect(pixelsToMP(undefined)).toBe("0");
    });

    it("keeps the remainder instead of rounding it away", () => {
        // given
        const pixels = "12345678";

        // when
        const mp = pixelsToMP(pixels);

        // then
        expect(mp).toBe("12.345678");
    });
});

describe("mpToPixels", () => {
    it("stores a pixel ceiling typed in megapixels as pixels", () => {
        // given
        const typed = "12";

        // when
        const pixels = mpToPixels(typed);

        // then
        expect(pixels).toBe("12000000");
    });

    it("treats a pixel ceiling that is not a number as zero", () => {
        // given / when / then
        expect(mpToPixels("not a number")).toBe("0");
        expect(mpToPixels("")).toBe("0");
    });
});

describe("size round trips", () => {
    it("gives back the same byte count when it is a whole number of megabytes", () => {
        // given
        const bytes = String(50 * 1024 * 1024);

        // when
        const roundTripped = mbToBytes(bytesToMB(bytes));

        // then
        expect(roundTripped).toBe(bytes);
    });

    it("loses the remainder of a byte count that is not a whole megabyte", () => {
        // given
        const bytes = "1500000";

        // when
        const roundTripped = mbToBytes(bytesToMB(bytes));

        // then
        expect(roundTripped).toBe("1048576");
        expect(roundTripped).not.toBe(bytes);
    });

    it("loses nothing on the pixel round trip because megapixels keep their remainder", () => {
        // given
        const pixels = "12345678";

        // when
        const roundTripped = mpToPixels(pixelsToMP(pixels));

        // then
        expect(roundTripped).toBe(pixels);
    });
});

describe("chatbotOptInRoles", () => {
    it("offers only the vanity roles that already carry the permission", () => {
        // given
        const roles: StubVanityRole[] = [
            { id: "role-witch", label: "Witch's Familiar", permissions: ["use_chatbot"] },
            { id: "role-goat", label: "Goat", permissions: ["manage_roles"] },
            { id: "role-furniture", label: "Furniture", permissions: ["use_chatbot", "manage_roles"] },
        ];

        // when
        const offered = chatbotOptInRoles(roles);

        // then
        expect(offered.map(role => role.id)).toEqual(["role-witch", "role-furniture"]);
    });

    it("offers nothing when no role holds the permission", () => {
        // given
        const roles: StubVanityRole[] = [{ id: "role-goat", label: "Goat", permissions: [] }];

        // when
        const offered = chatbotOptInRoles(roles);

        // then
        expect(offered).toEqual([]);
    });
});

describe("validateSiteSettings", () => {
    it("passes a fully filled in settings table", () => {
        // given
        const settings = { ...VALID };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("refuses a max body size that is still zero", () => {
        // given
        const settings: SiteSettings = {};

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max body size must be greater than 0");
    });

    it("refuses a max image size that is still zero", () => {
        // given
        const settings = { ...VALID, max_image_size: "0" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max image size must be greater than 0");
    });

    it("refuses a max image pixel ceiling that is still zero", () => {
        // given
        const settings = { ...VALID, max_image_pixels: "0" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max image pixels must be greater than 0");
    });

    it("refuses an image limit that is larger than the whole request limit", () => {
        // given
        const settings = { ...VALID, max_image_size: String(60 * 1024 * 1024) };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max image size (60 MB) cannot exceed max body size (50 MB)");
    });

    it("refuses a video limit that is larger than the whole request limit", () => {
        // given
        const settings = { ...VALID, max_video_size: String(60 * 1024 * 1024) };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max video size (60 MB) cannot exceed max body size (50 MB)");
    });

    it("refuses an audio limit that is larger than the whole request limit", () => {
        // given
        const settings = { ...VALID, max_audio_size: String(60 * 1024 * 1024) };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max audio size (60 MB) cannot exceed max body size (50 MB)");
    });

    it("refuses a general file limit that is larger than the whole request limit", () => {
        // given
        const settings = { ...VALID, max_general_size: String(60 * 1024 * 1024) };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max general size (60 MB) cannot exceed max body size (50 MB)");
    });

    it("refuses a password with no minimum length at all", () => {
        // given
        const settings = { ...VALID, min_password_length: "0" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Minimum password length must be at least 1");
    });

    it("refuses a session shorter than a day", () => {
        // given
        const settings = { ...VALID, session_duration_days: "0" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Session duration must be at least 1 day");
    });

    it("refuses a negative daily theory allowance", () => {
        // given
        const settings = { ...VALID, max_theories_per_day: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max theories per day cannot be negative");
    });

    it("refuses a negative daily response allowance", () => {
        // given
        const settings = { ...VALID, max_responses_per_day: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max responses per day cannot be negative");
    });

    it("refuses to switch voice chat on without the LiveKit credentials", () => {
        // given
        const settings = { ...VALID, voice_enabled: "true" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Voice chat requires LiveKit URL, API key and API secret");
    });

    it("accepts voice chat once all three LiveKit fields are filled in", () => {
        // given
        const settings = {
            ...VALID,
            voice_enabled: "true",
            livekit_url: "wss://livekit.example",
            livekit_api_key: "key",
            livekit_api_secret: "secret",
        };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("leaves the LiveKit fields alone while voice chat is switched off", () => {
        // given
        const settings = { ...VALID, voice_enabled: "false", livekit_url: "" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("refuses a chatbot that may not answer with a single token", () => {
        // given
        const settings = { ...CHATBOT, chatbot_max_output_tokens: "0" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Chatbot max output tokens must be at least 1");
    });

    it("refuses a negative chatbot context window", () => {
        // given
        const settings = { ...CHATBOT, chatbot_context_messages: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Chatbot context messages cannot be negative");
    });

    it("refuses a negative chatbot reply chain", () => {
        // given
        const settings = { ...CHATBOT, chatbot_max_reply_chain: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Chatbot max reply chain cannot be negative");
    });

    it("refuses a negative chatbot reply cooldown", () => {
        // given
        const settings = { ...CHATBOT, chatbot_reply_cooldown_seconds: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Chatbot reply cooldown cannot be negative");
    });

    it("refuses a negative per member daily chatbot allowance", () => {
        // given
        const settings = { ...CHATBOT, chatbot_max_replies_per_user_per_day: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Chatbot max replies per user per day cannot be negative");
    });

    it("refuses a negative site wide daily chatbot allowance", () => {
        // given
        const settings = { ...CHATBOT, chatbot_max_replies_per_day: "-1" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Chatbot max replies per day cannot be negative");
    });

    it("leaves the chatbot limits alone while the chatbot is switched off", () => {
        // given
        const settings = { ...VALID, chatbot_enabled: "false", chatbot_max_output_tokens: "0" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("refuses the restriction with no opt-in role chosen", () => {
        // given
        const settings = { ...CHATBOT, chatbot_require_permission: "true", chatbot_opt_in_role: "" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Restricting characters to a permission requires an opt-in role so members can opt in");
    });

    it("points at the master switch when the restriction was left on with the chatbot off", () => {
        // given
        const settings = { ...VALID, chatbot_require_permission: "true", chatbot_opt_in_role: "" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe(
            "Restricting characters to a permission requires an opt-in role. Switch Enable Chatbot on to choose one.",
        );
    });

    it("treats an opt-in role of nothing but spaces as no role at all", () => {
        // given
        const settings = { ...CHATBOT, chatbot_require_permission: "true", chatbot_opt_in_role: "   " };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Restricting characters to a permission requires an opt-in role so members can opt in");
    });

    it("accepts the restriction once a role is chosen", () => {
        // given
        const settings = { ...CHATBOT, chatbot_require_permission: "true", chatbot_opt_in_role: "role-witch" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("refuses the Cloudflare email provider without its credentials", () => {
        // given
        const settings = { ...VALID, email_provider: "cloudflare" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Cloudflare email requires account ID, API token and from address");
    });

    it("accepts the Cloudflare email provider once all three fields are filled in", () => {
        // given
        const settings = {
            ...VALID,
            email_provider: "cloudflare",
            cloudflare_account_id: "account",
            cloudflare_api_token: "token",
            cloudflare_email_from: "witch@example.com",
        };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("asks for no Cloudflare credentials while SMTP is the provider", () => {
        // given
        const settings = { ...VALID, email_provider: "smtp" };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBeNull();
    });

    it("reports the sizes in megabytes the admin typed rather than raw bytes", () => {
        // given
        const settings = { ...VALID, max_body_size: String(5 * 1024 * 1024), max_image_size: String(10 * 1024 * 1024) };

        // when
        const error = validateSiteSettings(settings);

        // then
        expect(error).toBe("Max image size (10 MB) cannot exceed max body size (5 MB)");
    });
});
