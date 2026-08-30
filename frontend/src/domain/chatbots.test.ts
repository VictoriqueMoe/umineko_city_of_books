import { describe, expect, it } from "vitest";
import {
    buildPayload,
    EMPTY_CHATBOT_FIELDS,
    fieldsFromBot,
    usernameHint,
    type ChatbotFields,
    type UsernameCheck,
} from "./chatbots";
import type { Chatbot } from "../types/api";

function makeBot(overrides: Partial<Chatbot> = {}): Chatbot {
    return {
        id: "bot-1",
        user_id: "user-1",
        username: "beato",
        display_name: "Beatrice",
        avatar_url: "https://example.com/beato.png",
        system_prompt: "You are the Golden Witch.",
        base_prompt_id: "base-1",
        model: "gpt-5",
        reasoning_effort: "high",
        verbosity: "medium",
        max_output_tokens: 2048,
        enabled: true,
        ...overrides,
    };
}

function makeFields(overrides: Partial<ChatbotFields> = {}): ChatbotFields {
    return { ...EMPTY_CHATBOT_FIELDS, ...overrides };
}

describe("EMPTY_CHATBOT_FIELDS", () => {
    it("starts every field of a new bot blank", () => {
        // given
        const keys = Object.keys(EMPTY_CHATBOT_FIELDS) as (keyof ChatbotFields)[];

        // when
        const values = keys.map(key => EMPTY_CHATBOT_FIELDS[key]);

        // then
        expect(keys).toHaveLength(9);
        expect(values.every(value => value === "")).toBe(true);
    });
});

describe("fieldsFromBot", () => {
    it("seeds the form from a saved bot", () => {
        // given
        const bot = makeBot();

        // when
        const fields = fieldsFromBot(bot);

        // then
        expect(fields).toEqual({
            username: "beato",
            displayName: "Beatrice",
            avatarURL: "https://example.com/beato.png",
            prompt: "You are the Golden Witch.",
            basePromptID: "base-1",
            model: "gpt-5",
            effort: "high",
            verbosity: "medium",
            maxTokens: "2048",
        });
    });

    it("turns a bot with no base prompt into the None selection", () => {
        // given
        const bot = makeBot({ base_prompt_id: null });

        // when
        const fields = fieldsFromBot(bot);

        // then
        expect(fields.basePromptID).toBe("");
    });

    it("leaves the token cap blank rather than showing a zero the admin did not type", () => {
        // given
        const bots = [makeBot({ max_output_tokens: 0 }), makeBot({ max_output_tokens: -1 })];

        // when
        const caps = bots.map(bot => fieldsFromBot(bot).maxTokens);

        // then
        expect(caps).toEqual(["", ""]);
    });
});

describe("buildPayload", () => {
    it("trims the identity fields the server matches on", () => {
        // given
        const fields = makeFields({
            username: "  beato  ",
            displayName: "  Beatrice  ",
            avatarURL: "  https://example.com/beato.png  ",
            model: "  gpt-5  ",
        });

        // when
        const payload = buildPayload(fields, true);

        // then
        expect(payload.username).toBe("beato");
        expect(payload.display_name).toBe("Beatrice");
        expect(payload.avatar_url).toBe("https://example.com/beato.png");
        expect(payload.model).toBe("gpt-5");
    });

    it("sends the system prompt exactly as typed, whitespace included", () => {
        // given
        const fields = makeFields({ prompt: "\n  You are the Golden Witch.  \n" });

        // when
        const payload = buildPayload(fields, true);

        // then
        expect(payload.system_prompt).toBe("\n  You are the Golden Witch.  \n");
    });

    it("sends no base prompt when the None option is selected", () => {
        // given
        const fields = makeFields({ basePromptID: "" });

        // when
        const payload = buildPayload(fields, true);

        // then
        expect(payload.base_prompt_id).toBeNull();
    });

    it("sends the chosen base prompt id when one is selected", () => {
        // given
        const fields = makeFields({ basePromptID: "base-9" });

        // when
        const payload = buildPayload(fields, true);

        // then
        expect(payload.base_prompt_id).toBe("base-9");
    });

    it("parses the token cap as a whole number and truncates anything after the digits", () => {
        // given
        const typed = ["2048", "12.9", "2048abc"];

        // when
        const caps = typed.map(maxTokens => buildPayload(makeFields({ maxTokens }), true).max_output_tokens);

        // then
        expect(caps).toEqual([2048, 12, 2048]);
    });

    it("falls back to no cap when the field is blank or not a number", () => {
        // given
        const typed = ["", "   ", "lots"];

        // when
        const caps = typed.map(maxTokens => buildPayload(makeFields({ maxTokens }), true).max_output_tokens);

        // then
        expect(caps).toEqual([0, 0, 0]);
    });

    it("carries the enabled flag from the caller rather than the form", () => {
        // given
        const fields = makeFields({ username: "beato" });

        // when
        const enabledFlags = [buildPayload(fields, true).enabled, buildPayload(fields, false).enabled];

        // then
        expect(enabledFlags).toEqual([true, false]);
    });

    it("round trips a saved bot back to the payload that produced it", () => {
        // given
        const bot = makeBot();

        // when
        const payload = buildPayload(fieldsFromBot(bot), bot.enabled);

        // then
        expect(payload).toEqual({
            username: "beato",
            display_name: "Beatrice",
            avatar_url: "https://example.com/beato.png",
            system_prompt: "You are the Golden Witch.",
            base_prompt_id: "base-1",
            model: "gpt-5",
            reasoning_effort: "high",
            verbosity: "medium",
            max_output_tokens: 2048,
            enabled: true,
        });
    });
});

describe("usernameHint", () => {
    it("explains that an existing handle is fixed, whatever the check last said", () => {
        // given
        const checks: UsernameCheck[] = [
            { state: "idle" },
            { state: "checking" },
            { state: "available", username: "beato" },
            { state: "taken", username: "beato" },
            { state: "failed" },
        ];

        // when
        const hints = checks.map(check => usernameHint(check, true));

        // then
        expect(new Set(hints)).toEqual(new Set(["A bot's handle cannot be changed after it is created."]));
    });

    it("describes the handle rule before anything has been checked", () => {
        // given
        const check: UsernameCheck = { state: "idle" };

        // when
        const hint = usernameHint(check, false);

        // then
        expect(hint).toBe("The handle members type to reach the bot. It has to be free, exactly like a human account.");
    });

    it("says a check is in flight", () => {
        // given
        const check: UsernameCheck = { state: "checking" };

        // when
        const hint = usernameHint(check, false);

        // then
        expect(hint).toBe("Checking whether that handle is free...");
    });

    it("names the free handle", () => {
        // given
        const check: UsernameCheck = { state: "available", username: "beato" };

        // when
        const hint = usernameHint(check, false);

        // then
        expect(hint).toBe("@beato is free.");
    });

    it("names the taken handle and asks for another", () => {
        // given
        const check: UsernameCheck = { state: "taken", username: "beato" };

        // when
        const hint = usernameHint(check, false);

        // then
        expect(hint).toBe("@beato is already taken. Pick another.");
    });

    it("says a failed check does not block saving", () => {
        // given
        const check: UsernameCheck = { state: "failed" };

        // when
        const hint = usernameHint(check, false);

        // then
        expect(hint).toBe("Could not check that handle just now. Saving will still tell you if it is taken.");
    });
});
