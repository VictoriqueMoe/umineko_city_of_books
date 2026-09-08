-- name: CreateUser :one
WITH u AS (
    INSERT INTO users (username, email, password_hash, display_name, avatar_url, home_page, is_bot, dms_enabled, email_verified)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id, username, password_hash, display_name, display_name_locked, created_at, bio, avatar_url, banner_url,
              favourite_character, gender, pronoun_subject, pronoun_possessive, banned_at, banned_by, ban_reason,
              locked_at, locked_by, lock_reason, approved_at, approved_by, social_twitter, social_discord,
              social_waifulist, social_tumblr, social_github, social_bluesky, website, banner_position, dms_enabled,
              episode_progress, higurashi_arc_progress, ciconia_chapter_progress, email, email_public, email_verified,
              verify_grace_until, dob, dob_public, email_notifications, play_message_sound, play_notification_sound,
              home_page, game_board_sort, default_profile_tab, theme, font, wide_layout, ip,
              mystery_score_adjustment, gm_score_adjustment, is_bot, follow_activity_notifications, echoes_enabled
)
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM u
LEFT JOIN user_roles r ON r.user_id = u.id;

-- name: GetUserByID :one
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE u.id = $1;

-- name: GetUsersByIDs :many
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE u.id = ANY(string_to_array($1::text, ',')::uuid[]);

-- name: GetUserByUsername :one
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE LOWER(u.username) = LOWER($1);

-- name: GetUsersByUsernames :many
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE LOWER(u.username) = ANY(string_to_array($1::text, ','));

-- name: ListUsersByIP :many
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE u.ip = $1::text AND u.id <> $2
ORDER BY u.created_at DESC;

-- name: ListUsers :many
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE ($1::text = '' OR u.username ILIKE $2::text OR u.display_name ILIKE $2::text)
ORDER BY u.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountUsersFiltered :one
SELECT COUNT(*) FROM users u
WHERE ($1::text = '' OR u.username ILIKE $2::text OR u.display_name ILIKE $2::text);

-- name: ListPublicUsers :many
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE u.banned_at IS NULL AND NOT u.is_bot
ORDER BY LOWER(u.display_name);

-- name: SearchUsersByName :many
SELECT u.id, u.username, u.password_hash, u.display_name, u.display_name_locked, u.created_at, u.bio, u.avatar_url,
       u.banner_url, u.favourite_character, u.gender, u.pronoun_subject, u.pronoun_possessive,
       u.banned_at, u.banned_by, u.ban_reason, u.locked_at, u.locked_by, u.lock_reason, u.approved_at, u.approved_by,
       u.social_twitter, u.social_discord, u.social_waifulist, u.social_tumblr, u.social_github, u.social_bluesky,
       u.website, u.banner_position, u.dms_enabled, u.episode_progress, u.higurashi_arc_progress,
       u.ciconia_chapter_progress, u.email, u.email_public, u.email_verified, u.verify_grace_until, u.dob,
       u.dob_public, u.email_notifications, u.play_message_sound, u.play_notification_sound, u.home_page,
       u.game_board_sort, u.default_profile_tab, u.theme, u.font, u.wide_layout, u.ip,
       u.mystery_score_adjustment, u.gm_score_adjustment,
       COALESCE(r.role, '')::text AS role,
       u.is_bot, u.follow_activity_notifications, u.echoes_enabled
FROM users u
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE u.banned_at IS NULL AND (u.username ILIKE $1::text OR u.display_name ILIKE $1::text)
ORDER BY CASE WHEN u.username ILIKE $2::text THEN 0 ELSE 1 END, LOWER(u.display_name)
LIMIT $3;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountUsersByUsername :one
SELECT COUNT(*) FROM users WHERE LOWER(username) = LOWER($1);

-- name: GetUserPasswordHash :one
SELECT password_hash FROM users WHERE id = $1;

-- name: SetUserPasswordHash :exec
UPDATE users SET password_hash = $1 WHERE id = $2;

-- name: SetUserEmail :exec
UPDATE users SET email = $1, email_verified = FALSE WHERE id = $2;

-- name: SetUserDisplayName :exec
UPDATE users SET display_name = $1 WHERE id = $2;

-- name: SetUserDisplayNameLocked :exec
UPDATE users SET display_name_locked = $1 WHERE id = $2;

-- name: MarkUserEmailVerified :exec
UPDATE users SET email_verified = TRUE WHERE id = $1;

-- name: MarkUserEmailUnverified :exec
UPDATE users SET email_verified = FALSE WHERE id = $1;

-- name: UserEmailInUse :one
SELECT EXISTS(SELECT 1 FROM users u WHERE LOWER(u.email) = LOWER($1) AND u.email <> '' AND u.id <> $2) AS in_use;

-- name: UserRequiresEmailVerification :one
SELECT (NOT u.email_verified AND NOW() >= u.verify_grace_until)::boolean AS blocked
FROM users u
WHERE u.id = $1;

-- name: UpdateUserProfile :exec
UPDATE users SET display_name = $1, bio = $2, banner_position = $3, favourite_character = $4, gender = $5,
    pronoun_subject = $6, pronoun_possessive = $7,
    social_twitter = $8, social_discord = $9, social_waifulist = $10, social_tumblr = $11, social_github = $12,
    social_bluesky = $13, website = $14, dms_enabled = $15, episode_progress = $16, higurashi_arc_progress = $17,
    ciconia_chapter_progress = $18, email = $19, email_public = $20, dob = $21, dob_public = $22,
    email_notifications = $23, play_message_sound = $24, play_notification_sound = $25, home_page = $26,
    game_board_sort = $27, default_profile_tab = $28, follow_activity_notifications = $29, echoes_enabled = $30
WHERE id = $31;

-- name: UpdateUserAvatarURL :exec
UPDATE users SET avatar_url = $1 WHERE id = $2;

-- name: UpdateUserBannerURL :exec
UPDATE users SET banner_url = $1 WHERE id = $2;

-- name: UpdateUserIP :exec
UPDATE users SET ip = $1::text WHERE id = $2;

-- name: UpdateUserGameBoardSort :exec
UPDATE users SET game_board_sort = $1 WHERE id = $2;

-- name: UpdateUserAppearance :exec
UPDATE users SET theme = $1, font = $2, wide_layout = $3 WHERE id = $4;

-- name: UpdateUserMysteryScoreAdjustment :exec
UPDATE users SET mystery_score_adjustment = $1 WHERE id = $2;

-- name: UpdateUserGMScoreAdjustment :exec
UPDATE users SET gm_score_adjustment = $1 WHERE id = $2;

-- name: GetUserDetectiveRawScore :one
SELECT COALESCE(SUM(
    CASE m.difficulty
        WHEN 'easy' THEN 2
        WHEN 'medium' THEN 4
        WHEN 'hard' THEN 6
        WHEN 'nightmare' THEN 8
        ELSE 4
    END
), 0)::bigint AS score
FROM mysteries m
WHERE m.winner_id = $1 AND m.solved = TRUE;

-- name: GetUserGMRawScore :one
SELECT COALESCE(SUM(
    CASE m.difficulty
        WHEN 'easy' THEN 2
        WHEN 'medium' THEN 4
        WHEN 'hard' THEN 6
        WHEN 'nightmare' THEN 8
        ELSE 4
    END
    + LEAST((SELECT COUNT(DISTINCT a.user_id) FROM mystery_attempts a WHERE a.mystery_id = m.id), 5)
), 0)::bigint AS score
FROM mysteries m
WHERE m.user_id = $1 AND m.solved = TRUE;

-- name: CountProfileTheories :one
SELECT COUNT(*) FROM theories WHERE user_id = $1;

-- name: CountProfileResponses :one
SELECT COUNT(*) FROM responses WHERE user_id = $1;

-- name: SumProfileTheoryVotes :one
SELECT COALESCE(SUM(tv.value), 0)::bigint AS total
FROM theory_votes tv
JOIN theories t ON tv.theory_id = t.id
WHERE t.user_id = $1;

-- name: SumProfileResponseVotes :one
SELECT COALESCE(SUM(rv.value), 0)::bigint AS total
FROM response_votes rv
JOIN responses r ON rv.response_id = r.id
WHERE r.user_id = $1;

-- name: CountProfileShips :one
SELECT COUNT(*) FROM ships WHERE user_id = $1;

-- name: CountProfileMysteries :one
SELECT COUNT(*) FROM mysteries WHERE user_id = $1;

-- name: CountProfileFanfics :one
SELECT COUNT(*) FROM fanfics WHERE user_id = $1;

-- name: BanUser :exec
UPDATE users SET banned_at = NOW(), banned_by = $1, ban_reason = $2 WHERE id = $3;

-- name: UnbanUser :exec
UPDATE users SET banned_at = NULL, banned_by = NULL, ban_reason = '' WHERE id = $1;

-- name: GetUserBannedAt :one
SELECT banned_at FROM users WHERE id = $1;

-- name: LockUser :exec
UPDATE users SET locked_at = NOW(), locked_by = $1, lock_reason = $2 WHERE id = $3;

-- name: UnlockUser :exec
UPDATE users SET locked_at = NULL, locked_by = NULL, lock_reason = '' WHERE id = $1;

-- name: GetUserLockedAt :one
SELECT locked_at FROM users WHERE id = $1;

-- name: ApproveUser :exec
UPDATE users SET approved_at = NOW(), approved_by = $1 WHERE id = $2;

-- name: UnapproveUser :exec
UPDATE users SET approved_at = NULL, approved_by = NULL WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
