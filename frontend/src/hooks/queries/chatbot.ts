import { useQuery } from "@tanstack/react-query";
import { listChatbots } from "../../api/endpoints/chatbot";
import { queryKeys } from "../../api/queryKeys";

export function useChatbotList() {
    const query = useQuery({
        queryKey: queryKeys.chatbots.list(),
        queryFn: () => listChatbots(),
        staleTime: 5 * 60_000,
    });

    return {
        chatbots: query.data?.chatbots ?? [],
        loading: query.isLoading,
    };
}
