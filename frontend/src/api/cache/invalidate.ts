import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";

type UserList =
    | "posts"
    | "art"
    | "galleries"
    | "ships"
    | "mysteries"
    | "fanfics"
    | "fanfic-favourites"
    | "journals"
    | "followed-journals";

interface DomainViews {
    keys: readonly QueryKey[];
    userLists: readonly UserList[];
    profileStats: boolean;
}

const domainViews = {
    post: { keys: [queryKeys.post.all], userLists: ["posts"], profileStats: false },
    art: {
        keys: [queryKeys.art.all, queryKeys.gallery.allGalleries(), queryKeys.gallery.root()],
        userLists: ["art", "galleries"],
        profileStats: false,
    },
    theory: { keys: [queryKeys.theory.all], userLists: [], profileStats: true },
    mystery: { keys: [queryKeys.mystery.all], userLists: ["mysteries"], profileStats: true },
    ship: { keys: [queryKeys.ship.all], userLists: ["ships"], profileStats: true },
    fanfic: { keys: [queryKeys.fanfic.all], userLists: ["fanfics", "fanfic-favourites"], profileStats: true },
    journal: { keys: [queryKeys.journal.all], userLists: ["journals", "followed-journals"], profileStats: false },
    oc: { keys: [queryKeys.oc.all], userLists: [], profileStats: false },
    secret: { keys: [queryKeys.secrets.all], userLists: [], profileStats: false },
    announcement: {
        keys: [queryKeys.announcements.all, queryKeys.admin.announcements()],
        userLists: [],
        profileStats: false,
    },
    rooms: { keys: [queryKeys.chat.rooms(), queryKeys.chat.roomsList()], userLists: [], profileStats: false },
    gameRooms: {
        keys: [
            queryKeys.gameRoom.list(),
            queryKeys.gameRoom.live(),
            queryKeys.gameRoom.finishedAll(),
            queryKeys.gameRoom.scoreboardAll(),
        ],
        userLists: [],
        profileStats: false,
    },
} as const satisfies Record<string, DomainViews>;

export type ViewDomain = keyof typeof domainViews;

export function invalidateViews(qc: QueryClient, domain: ViewDomain): void {
    const views: DomainViews = domainViews[domain];

    for (const key of views.keys) {
        qc.invalidateQueries({ queryKey: key });
    }

    if (views.userLists.length > 0) {
        qc.invalidateQueries({
            predicate: query =>
                query.queryKey[0] === "user" && views.userLists.some(list => list === query.queryKey[2]),
        });
    }

    if (views.profileStats) {
        qc.invalidateQueries({ queryKey: queryKeys.profile.all });
    }
}

export function invalidateReferencedViews(qc: QueryClient, referenceType: string): void {
    const family = referenceType.split(":")[0].split("_")[0];
    if (!Object.keys(domainViews).includes(family)) {
        return;
    }

    invalidateViews(qc, family as ViewDomain);
}
