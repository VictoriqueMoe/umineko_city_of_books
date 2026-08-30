import { describe, expect, it } from "vitest";
import { buildMentionMatcher } from "./mentions";

function matcherFor(username: string): (body: string) => boolean {
    const matcher = buildMentionMatcher(username);
    if (matcher === null) {
        throw new Error("expected a matcher for a non-empty username");
    }
    return matcher;
}

describe("buildMentionMatcher", () => {
    it("returns nothing when there is no username to match against", () => {
        // given / when / then
        expect(buildMentionMatcher(undefined)).toBeNull();
        expect(buildMentionMatcher("")).toBeNull();
    });

    it("matches a mention anywhere in the body", () => {
        // given
        const matches = matcherFor("beatrice");

        // when / then
        expect(matches("@beatrice come here")).toBe(true);
        expect(matches("hey @beatrice how are you")).toBe(true);
        expect(matches("the golden witch @beatrice")).toBe(true);
    });

    it("ignores the case of the mention", () => {
        // given
        const matches = matcherFor("beatrice");

        // when / then
        expect(matches("@Beatrice")).toBe(true);
        expect(matches("@BEATRICE")).toBe(true);
    });

    it("ignores the case of the configured username", () => {
        // given
        const matches = matcherFor("BeaTrice");

        // when
        const result = matches("hello @beatrice");

        // then
        expect(result).toBe(true);
    });

    it("does not match the bare username without an at sign", () => {
        // given
        const matches = matcherFor("beatrice");

        // when / then
        expect(matches("beatrice was here")).toBe(false);
        expect(matches("")).toBe(false);
    });

    it("does not match a longer username that merely starts with this one", () => {
        // given
        const matches = matcherFor("beato");

        // when / then
        expect(matches("@beatorice")).toBe(false);
        expect(matches("@beato2")).toBe(false);
        expect(matches("@beato_lite")).toBe(false);
    });

    it("still matches when the mention is followed by punctuation or the end of the body", () => {
        // given
        const matches = matcherFor("beato");

        // when / then
        expect(matches("@beato")).toBe(true);
        expect(matches("@beato!")).toBe(true);
        expect(matches("@beato, are you there?")).toBe(true);
        expect(matches("@beato-chan")).toBe(true);
        expect(matches("(@beato)")).toBe(true);
    });

    it("does not match an email address that happens to end in the username", () => {
        // given
        const matches = matcherFor("beatrice");

        // when / then
        expect(matches("mail me at kujo@beatrice")).toBe(false);
        expect(matches("kujo@beatrice.example")).toBe(false);
        expect(matches("battler_1@beatrice")).toBe(false);
    });

    it("still matches when the at sign follows punctuation or an opening bracket", () => {
        // given
        const matches = matcherFor("beatrice");

        // when / then
        expect(matches("(@beatrice)")).toBe(true);
        expect(matches("hey,@beatrice")).toBe(true);
        expect(matches("line one\n@beatrice")).toBe(true);
    });

    it("matches only one of several mentions being enough", () => {
        // given
        const matches = matcherFor("battler");

        // when / then
        expect(matches("@beatrice and @battler are arguing")).toBe(true);
        expect(matches("@beatrice and @ronove are arguing")).toBe(false);
    });

    it("treats regex metacharacters in the username as literal text", () => {
        // given
        const dotted = matcherFor("b.a");
        const plussed = matcherFor("a+b");

        // when / then
        expect(dotted("@b.a")).toBe(true);
        expect(dotted("@bxa")).toBe(false);
        expect(plussed("@a+b")).toBe(true);
        expect(plussed("@ab")).toBe(false);
    });

    it("gives each username its own independent matcher", () => {
        // given
        const beato = matcherFor("beatrice");
        const battler = matcherFor("battler");

        // when / then
        expect(beato("@battler")).toBe(false);
        expect(battler("@battler")).toBe(true);
    });
});
