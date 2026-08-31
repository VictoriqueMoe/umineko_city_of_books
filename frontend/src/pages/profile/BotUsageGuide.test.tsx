import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { SiteInfo } from "../../types/api";
import { BotUsageGuide } from "./BotUsageGuide";

describe("BotUsageGuide", () => {
    it("explains mentioning, replying and clearing a direct message", () => {
        // given / when
        renderWithProviders(<BotUsageGuide />);

        // then
        expect(screen.getByText("How to talk to me")).toBeInTheDocument();
        expect(screen.getByText(/Send me a direct message/)).toBeInTheDocument();
        expect(screen.getByText(/Mention me with @/)).toBeInTheDocument();
        expect(screen.getByText(/begins fresh, with no memory of the conversation before it/)).toBeInTheDocument();
        expect(screen.getByText(/carry on the same conversation/)).toBeInTheDocument();
        expect(screen.getByText("I read text only, and cannot see images.")).toBeInTheDocument();
        expect(screen.getByText(/To start again in a direct message/)).toBeInTheDocument();
        expect(screen.getByText("Delete Chat")).toBeInTheDocument();
    });

    it("scopes each memory limit to the surfaces the server actually applies it to", () => {
        // given / when
        renderWithProviders(<BotUsageGuide />, {
            siteInfo: { chatbot_context_messages: 20, chatbot_max_reply_chain: 25 },
        });

        // then
        const mention = screen.getByText(
            /in a chat room, on the game board or in comments to start a new conversation/,
        );
        expect(mention).toBeInTheDocument();
        expect(mention.textContent).not.toMatch(/\d+ messages/);

        expect(screen.getByText(/carry on the same conversation\. Replying is how I remember/)).toHaveTextContent(
            "back through up to 25 messages",
        );
        expect(screen.getByText(/In a direct message there is no need to mention me/)).toHaveTextContent(
            "read back over the last 20 messages",
        );
        expect(screen.queryByText(/direct messages and chat rooms/i)).not.toBeInTheDocument();
    });

    it("reads both limits from the live settings rather than a constant", () => {
        const cases: { name: string; siteInfo: Partial<SiteInfo>; context: RegExp; replyChain: RegExp }[] = [
            {
                name: "defaults",
                siteInfo: { chatbot_context_messages: 20, chatbot_max_reply_chain: 25 },
                context: /read back over the last 20 messages/,
                replyChain: /back through up to 25 messages/,
            },
            {
                name: "admin raised both",
                siteInfo: { chatbot_context_messages: 50, chatbot_max_reply_chain: 80 },
                context: /read back over the last 50 messages/,
                replyChain: /back through up to 80 messages/,
            },
            {
                name: "distinct values are not swapped",
                siteInfo: { chatbot_context_messages: 7, chatbot_max_reply_chain: 9 },
                context: /read back over the last 7 messages/,
                replyChain: /back through up to 9 messages/,
            },
        ];

        for (const testCase of cases) {
            // given / when
            const { unmount } = renderWithProviders(<BotUsageGuide />, { siteInfo: testCase.siteInfo });

            // then
            expect(screen.getByText(testCase.context), testCase.name).toBeInTheDocument();
            expect(screen.getByText(testCase.replyChain), testCase.name).toBeInTheDocument();

            unmount();
        }
    });

    it("degrades to plain wording when a limit is missing or not positive", () => {
        const cases: { name: string; siteInfo: Partial<SiteInfo> }[] = [
            { name: "zero", siteInfo: { chatbot_context_messages: 0, chatbot_max_reply_chain: 0 } },
            { name: "negative", siteInfo: { chatbot_context_messages: -5, chatbot_max_reply_chain: -1 } },
            {
                name: "absent",
                siteInfo: {
                    chatbot_context_messages: undefined as unknown as number,
                    chatbot_max_reply_chain: undefined as unknown as number,
                },
            },
            {
                name: "not a number",
                siteInfo: {
                    chatbot_context_messages: Number.NaN,
                    chatbot_max_reply_chain: "many" as unknown as number,
                },
            },
        ];

        for (const testCase of cases) {
            // given / when
            const { unmount } = renderWithProviders(<BotUsageGuide />, { siteInfo: testCase.siteInfo });

            // then
            expect(
                screen.getByText(/read back over a limited number of recent messages/),
                testCase.name,
            ).toBeInTheDocument();
            expect(screen.getByText(/back through a limited number of messages/), testCase.name).toBeInTheDocument();
            expect(screen.queryByText(/read back over the last/), testCase.name).not.toBeInTheDocument();
            expect(screen.queryByText(/back through up to/), testCase.name).not.toBeInTheDocument();
            expect(screen.queryByText(/\d+ messages/), testCase.name).not.toBeInTheDocument();

            unmount();
        }
    });
});
