import { useState } from "react";
import { can } from "../domain/permissions";
import { useAuth } from "./useAuth";
import { useBlock } from "./useBlock";
import { useFollow } from "./useFollow";
import { usePageOffset, type PageOffset } from "./usePageOffset";
import { usePageTitle } from "./usePageTitle";
import { useUserOCs } from "./queries/oc";
import { useProfile } from "./queries/profile";
import { useTheoryFeed } from "./queries/theory";
import {
    useFollowers,
    useFollowing,
    useUserActivity,
    useUserArt,
    useUserFanficFavourites,
    useUserFanfics,
    useUserFollowedJournals,
    useUserGalleries,
    useUserJournals,
    useUserMysteries,
    useUserPosts,
    useUserShips,
} from "./queries/user";

export type ProfileTab =
    | "posts"
    | "theories"
    | "art"
    | "galleries"
    | "ships"
    | "ocs"
    | "mysteries"
    | "fanfics"
    | "fanfic-favourites"
    | "journals"
    | "journal-follows"
    | "activity"
    | "followers"
    | "following";

export interface ProfileList<T> {
    items: T[];
    loading: boolean;
    total: number;
    page: PageOffset | null;
}

function list<T>(items: T[], loading: boolean, total: number, page: PageOffset | null): ProfileList<T> {
    return { items, loading, total, page };
}

export function useProfilePage(username: string) {
    const { user: viewer } = useAuth();
    const { profile, loading } = useProfile(username);
    usePageTitle(profile?.display_name ?? "Profile");

    const [explicitTab, setExplicitTab] = useState<ProfileTab | null>(null);
    const activeTab: ProfileTab =
        explicitTab ?? (profile?.private?.default_profile_tab as ProfileTab | undefined) ?? "posts";
    const setActiveTab = setExplicitTab as (tab: ProfileTab) => void;

    const follow = useFollow(profile?.id ?? "");
    const block = useBlock(profile?.id ?? "");
    const canManageUser = can(viewer, "view_users") && !!profile && viewer?.id !== profile.id;

    const profileID = profile?.id ?? "";

    const theoriesPage = usePageOffset({ limit: 20 });
    const theoriesQuery = useTheoryFeed({
        sort: "new",
        episode: 0,
        authorId: profileID,
        offset: theoriesPage.offset,
        limit: theoriesPage.limit,
        enabled: activeTab === "theories" && !!profileID,
    });

    const postsPage = usePageOffset({ limit: 20 });
    const postsQuery = useUserPosts(activeTab === "posts" ? profileID : "", postsPage.limit, postsPage.offset);

    const artPage = usePageOffset({ limit: 24 });
    const artQuery = useUserArt(activeTab === "art" ? profileID : "", artPage.limit, artPage.offset);

    const galleriesQuery = useUserGalleries(activeTab === "galleries" ? profileID : "");

    const shipsPage = usePageOffset({ limit: 20 });
    const shipsQuery = useUserShips(activeTab === "ships" ? profileID : "", shipsPage.limit, shipsPage.offset);

    const ocsQuery = useUserOCs(activeTab === "ocs" ? profileID : "");

    const mysteriesPage = usePageOffset({ limit: 20 });
    const mysteriesQuery = useUserMysteries(
        activeTab === "mysteries" ? profileID : "",
        mysteriesPage.limit,
        mysteriesPage.offset,
    );

    const fanficsPage = usePageOffset({ limit: 20 });
    const fanficsQuery = useUserFanfics(
        activeTab === "fanfics" ? profileID : "",
        fanficsPage.limit,
        fanficsPage.offset,
    );

    const favouritesPage = usePageOffset({ limit: 20 });
    const favouritesQuery = useUserFanficFavourites(
        activeTab === "fanfic-favourites" ? profileID : "",
        favouritesPage.limit,
        favouritesPage.offset,
    );

    const journalsPage = usePageOffset({ limit: 20 });
    const journalsQuery = useUserJournals(
        activeTab === "journals" ? profileID : "",
        journalsPage.limit,
        journalsPage.offset,
    );

    const followedJournalsPage = usePageOffset({ limit: 20 });
    const followedJournalsQuery = useUserFollowedJournals(
        activeTab === "journal-follows" ? profileID : "",
        followedJournalsPage.limit,
        followedJournalsPage.offset,
    );

    const activityPage = usePageOffset({ limit: 20 });
    const activityQuery = useUserActivity(
        activeTab === "activity" ? username : "",
        activityPage.limit,
        activityPage.offset,
    );

    const followersQuery = useFollowers(activeTab === "followers" ? profileID : "");
    const followingQuery = useFollowing(activeTab === "following" ? profileID : "");
    const followSource = activeTab === "followers" ? followersQuery : followingQuery;
    const followListLoading = activeTab === "followers" || activeTab === "following" ? followSource.loading : false;

    return {
        profile,
        loading,
        viewer,
        follow,
        block,
        canManageUser,
        activeTab,
        setActiveTab,
        posts: list(postsQuery.posts, postsQuery.loading, postsQuery.total, postsPage),
        refreshPosts: postsQuery.refresh,
        theories: list(theoriesQuery.theories, theoriesQuery.loading, theoriesQuery.total, theoriesPage),
        art: list(artQuery.art, artQuery.loading, artQuery.total, artPage),
        galleries: list(galleriesQuery.galleries, galleriesQuery.loading, galleriesQuery.galleries.length, null),
        refreshGalleries: galleriesQuery.refresh,
        ships: list(shipsQuery.ships, shipsQuery.loading, shipsQuery.total, shipsPage),
        ocs: list(ocsQuery.ocs, ocsQuery.loading, ocsQuery.total, null),
        mysteries: list(mysteriesQuery.mysteries, mysteriesQuery.loading, mysteriesQuery.total, mysteriesPage),
        fanfics: list(fanficsQuery.fanfics, fanficsQuery.loading, fanficsQuery.total, fanficsPage),
        favourites: list(favouritesQuery.fanfics, favouritesQuery.loading, favouritesQuery.total, favouritesPage),
        journals: list(journalsQuery.journals, journalsQuery.loading, journalsQuery.total, journalsPage),
        followedJournals: list(
            followedJournalsQuery.journals,
            followedJournalsQuery.loading,
            followedJournalsQuery.total,
            followedJournalsPage,
        ),
        activity: list(activityQuery.activity, activityQuery.loading, activityQuery.total, activityPage),
        followList: list(followSource.users, followListLoading, followSource.total, null),
    };
}
