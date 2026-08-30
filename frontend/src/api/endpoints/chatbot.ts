import { apiDelete, apiFetch, apiPost, apiPut, buildQueryString } from "../client";
import type {
    Chatbot,
    ChatbotBasePrompt,
    ChatbotBasePromptListResponse,
    ChatbotBasePromptPayload,
    ChatbotListResponse,
    ChatbotModels,
    ChatbotPayload,
    ChatbotTestResult,
    ChatbotUsage,
} from "../../types/api";

export async function listChatbots(): Promise<ChatbotListResponse> {
    return apiFetch<ChatbotListResponse>("/chatbots");
}

export async function getChatbots(): Promise<Chatbot[]> {
    return apiFetch<Chatbot[]>("/admin/chatbots");
}

export async function createChatbot(data: ChatbotPayload): Promise<Chatbot> {
    return apiPost<Chatbot, ChatbotPayload>("/admin/chatbots", data);
}

export async function updateChatbot(id: string, data: ChatbotPayload): Promise<void> {
    await apiPut<unknown, ChatbotPayload>(`/admin/chatbots/${id}`, data);
}

export async function deleteChatbot(id: string): Promise<void> {
    await apiDelete(`/admin/chatbots/${id}`);
}

export async function getChatbotBasePrompts(): Promise<ChatbotBasePromptListResponse> {
    return apiFetch<ChatbotBasePromptListResponse>("/admin/chatbots/base-prompts");
}

export async function createChatbotBasePrompt(data: ChatbotBasePromptPayload): Promise<ChatbotBasePrompt> {
    return apiPost<ChatbotBasePrompt, ChatbotBasePromptPayload>("/admin/chatbots/base-prompts", data);
}

export async function updateChatbotBasePrompt(id: string, data: ChatbotBasePromptPayload): Promise<ChatbotBasePrompt> {
    return apiPut<ChatbotBasePrompt, ChatbotBasePromptPayload>(`/admin/chatbots/base-prompts/${id}`, data);
}

export async function deleteChatbotBasePrompt(id: string): Promise<void> {
    await apiDelete(`/admin/chatbots/base-prompts/${id}`);
}

export async function getChatbotUsage(days: number): Promise<ChatbotUsage> {
    return apiFetch<ChatbotUsage>(`/admin/chatbots/usage${buildQueryString({ days })}`);
}

export async function getChatbotModels(): Promise<ChatbotModels> {
    return apiFetch<ChatbotModels>("/admin/chatbots/models");
}

export async function testChatbotModel(model: string): Promise<ChatbotTestResult> {
    return apiPost<ChatbotTestResult, { model: string }>("/admin/chatbots/test", { model });
}
