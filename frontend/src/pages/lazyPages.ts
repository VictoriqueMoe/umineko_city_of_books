import { type ComponentType, lazy } from "react";

// React.lazy only accepts modules with a `default` export, but our pages use
// named exports. `named` does that adapter so each page below fits on one line.
// The `any` mirrors React's own `lazy<T extends ComponentType<any>>` signature —
// you can't satisfy that constraint with `never`, `unknown`, or `object` because
// of how props variance interacts with class components inside ComponentType.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function named<T extends ComponentType<any>, K extends string>(loader: () => Promise<Record<K, T>>, name: K) {
    return lazy(() => loader().then(m => ({ default: m[name] })));
}

//  Theories
export const FeedPage = named(() => import("./theories/FeedPage"), "FeedPage");
export const TheoryPage = named(() => import("./theories/TheoryPage"), "TheoryPage");
export const CreateTheoryPage = named(() => import("./theories/CreateTheoryPage"), "CreateTheoryPage");
export const EditTheoryPage = named(() => import("./theories/EditTheoryPage"), "EditTheoryPage");

//  Auth & Quotes
export const LoginPage = named(() => import("./auth/LoginPage"), "LoginPage");
export const QuoteBrowserPage = named(() => import("./quotes/QuoteBrowserPage"), "QuoteBrowserPage");

//  Profile
export const ProfilePage = named(() => import("./profile/ProfilePage"), "ProfilePage");
export const SettingsPage = named(() => import("./profile/SettingsPage"), "SettingsPage");

//  Admin
export const AdminLayout = named(() => import("./admin/AdminLayout"), "AdminLayout");
export const AdminDashboard = named(() => import("./admin/AdminDashboard"), "AdminDashboard");
export const AdminUsers = named(() => import("./admin/AdminUsers"), "AdminUsers");
export const AdminUserDetail = named(() => import("./admin/AdminUserDetail"), "AdminUserDetail");
export const AdminSettings = named(() => import("./admin/AdminSettings"), "AdminSettings");
export const AdminAuditLog = named(() => import("./admin/AdminAuditLog"), "AdminAuditLog");
export const AdminInvites = named(() => import("./admin/AdminInvites"), "AdminInvites");
export const AdminReports = named(() => import("./admin/AdminReports"), "AdminReports");
export const AdminContentRules = named(() => import("./admin/AdminContentRules"), "AdminContentRules");
export const AdminVanityRoles = named(() => import("./admin/AdminVanityRoles"), "AdminVanityRoles");
export const AdminAnnouncementsPage = named(() => import("./admin/AdminAnnouncements"), "AdminAnnouncements");
export const AdminBannedGifs = named(() => import("./admin/AdminBannedGifs"), "AdminBannedGifs");
export const AdminBannedWords = named(() => import("./admin/AdminBannedWords"), "AdminBannedWords");

//  Announcements
export const AnnouncementsListPage = named(() => import("./announcements/AnnouncementsPage"), "AnnouncementsPage");
export const AnnouncementDetailPage = named(
    () => import("./announcements/AnnouncementDetailPage"),
    "AnnouncementDetailPage",
);

//  Mysteries
export const MysteryListPage = named(() => import("./mysteries/MysteryListPage"), "MysteryListPage");
export const MysteryDetailPage = named(() => import("./mysteries/MysteryDetailPage"), "MysteryDetailPage");
export const CreateMysteryPage = named(() => import("./mysteries/CreateMysteryPage"), "CreateMysteryPage");

//  Ships
export const ShipsListPage = named(() => import("./ships/ShipsListPage"), "ShipsListPage");
export const ShipDetailPage = named(() => import("./ships/ShipDetailPage"), "ShipDetailPage");
export const CreateShipPage = named(() => import("./ships/CreateShipPage"), "CreateShipPage");

//  Fanfiction
export const FanfictionListPage = named(() => import("./fanfiction/FanfictionListPage"), "FanfictionListPage");
export const FanficDetailPage = named(() => import("./fanfiction/FanficDetailPage"), "FanficDetailPage");
export const FanficChapterPage = named(() => import("./fanfiction/FanficChapterPage"), "FanficChapterPage");
export const FanficEditorPage = named(() => import("./fanfiction/FanficEditorPage"), "FanficEditorPage");
export const ChapterEditorPage = named(() => import("./fanfiction/ChapterEditorPage"), "ChapterEditorPage");

//  Suggestions
export const SuggestionsPage = named(() => import("./suggestions/SuggestionsPage"), "SuggestionsPage");

//  Game Board feed
export const SocialFeedPage = named(() => import("./feed/SocialFeedPage"), "SocialFeedPage");
export const PostDetailPage = named(() => import("./feed/PostDetailPage"), "PostDetailPage");

//  Users & Chat
export const UsersPage = named(() => import("./users/UsersPage"), "UsersPage");
export const ChatPage = named(() => import("./chat/ChatPage"), "ChatPage");

//  Gallery
export const ArtGalleryPage = named(() => import("./gallery/ArtGalleryPage"), "ArtGalleryPage");
export const ArtDetailPage = named(() => import("./gallery/ArtDetailPage"), "ArtDetailPage");
export const GalleryDetailPage = named(() => import("./gallery/GalleryDetailPage"), "GalleryDetailPage");

//  Notifications
export const NotificationsPage = named(() => import("./notifications/NotificationsPage"), "NotificationsPage");

//  Reading Journals
export const JournalsFeedPage = named(() => import("./journals/FeedPage"), "JournalsFeedPage");
export const JournalPage = named(() => import("./journals/JournalPage"), "JournalPage");
export const CreateJournalPage = named(() => import("./journals/CreateJournalPage"), "CreateJournalPage");
export const EditJournalPage = named(() => import("./journals/EditJournalPage"), "EditJournalPage");

//  Chat Rooms
export const RoomsListPage = named(() => import("./rooms/RoomsListPage"), "RoomsListPage");
export const RoomPage = named(() => import("./rooms/RoomPage"), "RoomPage");

//  Not Found
export const NotFoundPage = named(() => import("./notfound/NotFoundPage"), "NotFoundPage");

//  Secrets
export const SecretsListPage = named(() => import("./secrets/SecretsListPage"), "SecretsListPage");
export const SecretDetailPage = named(() => import("./secrets/SecretDetailPage"), "SecretDetailPage");

//  Games
export const GamesListPage = named(() => import("./games/GamesListPage"), "GamesListPage");
export const LiveGamesPage = named(() => import("./games/LiveGamesPage"), "LiveGamesPage");
export const PastGamesPage = named(() => import("./games/PastGamesPage"), "PastGamesPage");
export const GameHubPage = named(() => import("./games/GameHubPage"), "GameHubPage");
export const NewChessGamePage = named(() => import("./games/NewChessGamePage"), "NewChessGamePage");
export const ChessGamePage = named(() => import("./games/ChessGamePage"), "ChessGamePage");
export const NewCheckersGamePage = named(() => import("./games/NewCheckersGamePage"), "NewCheckersGamePage");
export const CheckersGamePage = named(() => import("./games/CheckersGamePage"), "CheckersGamePage");
