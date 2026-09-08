package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	MysteryDAO interface {
		Create(ctx context.Context, s spec.NewMystery, tx ...*sql.Tx) (*model.MysteryRow, error)
		AddClue(ctx context.Context, s spec.NewMysteryClue, tx ...*sql.Tx) (*dto.MysteryClue, error)
		Update(ctx context.Context, s spec.MysteryOwnerUpdate, tx ...*sql.Tx) error
		UpdateAsAdmin(ctx context.Context, s spec.MysteryUpdate, tx ...*sql.Tx) error
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.MysteryRow, error)
		List(ctx context.Context, s spec.MysteryListFilter, tx ...*sql.Tx) ([]model.MysteryRow, int, error)
		ListByUser(ctx context.Context, s spec.MysteryUserListFilter, tx ...*sql.Tx) ([]model.MysteryRow, int, error)
		GetClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryClue, error)
		DeleteClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error
		DeleteClue(ctx context.Context, clueID int, tx ...*sql.Tx) error
		UpdateClue(ctx context.Context, s spec.MysteryClueUpdate, tx ...*sql.Tx) error
		GetAuthorID(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		CreateAttempt(ctx context.Context, s spec.NewMysteryAttempt, tx ...*sql.Tx) (*model.MysteryAttemptRow, error)
		DeleteAttempt(ctx context.Context, s spec.MysteryAttemptDeletion, tx ...*sql.Tx) error
		DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetAttempts(ctx context.Context, s spec.MysteryAttemptQuery, tx ...*sql.Tx) ([]model.MysteryAttemptRow, error)
		GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)

		VoteAttempt(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error

		GetAttemptOwner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error)
		SetMysteryWinner(ctx context.Context, s spec.MysteryWinner, tx ...*sql.Tx) error
		SetAttemptWinner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) error
		MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error
		UserHasWinningAttempt(ctx context.Context, s spec.MysterySolverQuery, tx ...*sql.Tx) (bool, error)
		GetSolverIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)
		IsSolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error)
		IsPaused(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error)
		SetPaused(ctx context.Context, s spec.MysteryPauseUpdate, tx ...*sql.Tx) error
		SetGmAway(ctx context.Context, s spec.MysteryGmAwayUpdate, tx ...*sql.Tx) error

		GetLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.LeaderboardEntry, error)
		GetTopDetectiveIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error)
		GetGMLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.GMLeaderboardEntry, error)
		GetTopGMIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error)

		CountAttempts(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error)
		CountClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error)
		GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error)

		UpdateComment(ctx context.Context, s spec.CommentUpdate, tx ...*sql.Tx) error
		DeleteComment(ctx context.Context, s spec.CommentDeletion, tx ...*sql.Tx) error
		GetComments(ctx context.Context, q spec.CommentQuery[uuid.UUID], tx ...*sql.Tx) ([]model.CommentRow, int, error)
		GetCommentEntityID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetCommentAuthorID(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		LikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		UnlikeComment(ctx context.Context, s spec.CommentLike, tx ...*sql.Tx) error
		AddCommentMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateCommentMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateCommentMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetCommentMedia(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		GetCommentMediaBatch(ctx context.Context, commentIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]model.PostMediaRow, error)

		AddAttachment(ctx context.Context, s spec.NewMysteryAttachment, tx ...*sql.Tx) (int64, error)
		DeleteAttachment(ctx context.Context, s spec.MysteryAttachmentDeletion, tx ...*sql.Tx) error
		GetAttachments(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryAttachment, error)

		AddMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		GetMedia(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]model.PostMediaRow, error)
		DeleteMedia(ctx context.Context, s spec.MediaDeletion, tx ...*sql.Tx) (string, error)

		GetAttachmentPaths(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectCommentMediaPaths(ctx context.Context, entityID uuid.UUID, tx ...*sql.Tx) ([]string, error)
		CollectSingleCommentMediaPaths(ctx context.Context, commentID uuid.UUID, tx ...*sql.Tx) ([]string, error)
	}

	mysteryDAO struct {
		db *sql.DB
		*ownedDAO
		attemptVotes *voteDAO
		*commentDAO[uuid.UUID]
		*mediaDAO
	}

	mysteryJoinRow        = sqlcgen.GetMysteryByIDRow
	mysteryAttemptJoinRow = sqlcgen.GetMysteryAttemptsRow
	mysteryClueRow        = sqlcgen.GetMysteryCluesRow
)

func toMysteryRow(row mysteryJoinRow) model.MysteryRow {
	out := model.MysteryRow{
		ID:                 row.ID,
		UserID:             row.UserID,
		Title:              row.Title,
		Body:               row.Body,
		Difficulty:         row.Difficulty,
		Solved:             row.Solved,
		Paused:             row.Paused,
		GmAway:             row.GmAway,
		FreeForAll:         row.FreeForAll,
		KeepOpenAfterSolve: row.KeepOpenAfterSolve,
		Knox: dto.KnoxContract{
			CulpritNamedEarly:    row.KnoxCulpritNamedEarly,
			NoSupernatural:       row.KnoxNoSupernatural,
			PassagesDeclared:     row.KnoxPassagesDeclared,
			NoUnknownPoison:      row.KnoxNoUnknownPoison,
			NoOutsider:           row.KnoxNoOutsider,
			NoLuckyAccident:      row.KnoxNoLuckyAccident,
			DetectiveNotCulprit:  row.KnoxDetectiveNotCulprit,
			CluesShown:           row.KnoxCluesShown,
			NarratorHidesNothing: row.KnoxNarratorHidesNothing,
			NoUnannouncedTwins:   row.KnoxNoUnannouncedTwins,
		},
		KnoxPublished:         row.KnoxContractPublished,
		WinnerID:              row.WinnerID,
		SolvedAt:              nullTimeToStringPtr(row.SolvedAt),
		PausedAt:              nullTimeToStringPtr(row.PausedAt),
		PausedDurationSeconds: int(row.PausedDurationSeconds),
		AuthorUsername:        row.AuthorUsername,
		AuthorDisplayName:     row.AuthorDisplayName,
		AuthorAvatarURL:       row.AuthorAvatarUrl,
		AuthorRole:            row.AuthorRole,
		AttemptCount:          int(row.AttemptCount),
		ClueCount:             int(row.ClueCount),
		SolverCount:           int(row.SolverCount),
		CreatedAt:             row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             row.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if row.WinnerUsername.Valid {
		out.WinnerUsername = new(row.WinnerUsername.String)
		out.WinnerDisplayName = new(row.WinnerDisplayName.String)
		out.WinnerAvatarURL = new(row.WinnerAvatarUrl.String)
		out.WinnerRole = new(row.WinnerRole)
	}

	return out
}

func toMysteryAttemptRow(row mysteryAttemptJoinRow) model.MysteryAttemptRow {
	return model.MysteryAttemptRow{
		ID:                row.ID,
		MysteryID:         row.MysteryID,
		UserID:            row.UserID,
		ParentID:          row.ParentID,
		Body:              row.Body,
		IsWinner:          row.IsWinner,
		AuthorUsername:    row.AuthorUsername,
		AuthorDisplayName: row.AuthorDisplayName,
		AuthorAvatarURL:   row.AuthorAvatarUrl,
		AuthorRole:        row.AuthorRole,
		VoteScore:         int(row.VoteScore),
		UserVote:          int(row.UserVote),
		CreatedAt:         row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toMysteryClue(row mysteryClueRow) dto.MysteryClue {
	return dto.MysteryClue{
		ID:        int(row.ID),
		Body:      row.Body,
		TruthType: row.TruthType,
		SortOrder: int(row.SortOrder),
		PlayerID:  row.PlayerID,
	}
}

func (r *mysteryDAO) Create(ctx context.Context, s spec.NewMystery, tx ...*sql.Tx) (*model.MysteryRow, error) {
	created, err := genQueries(r.db, tx).CreateMystery(ctx, sqlcgen.CreateMysteryParams{
		UserID:                   s.UserID,
		Title:                    s.Title,
		Body:                     s.Body,
		Difficulty:               s.Difficulty,
		FreeForAll:               s.FreeForAll,
		KeepOpenAfterSolve:       s.KeepOpenAfterSolve,
		KnoxCulpritNamedEarly:    s.Knox.CulpritNamedEarly,
		KnoxNoSupernatural:       s.Knox.NoSupernatural,
		KnoxPassagesDeclared:     s.Knox.PassagesDeclared,
		KnoxNoUnknownPoison:      s.Knox.NoUnknownPoison,
		KnoxNoOutsider:           s.Knox.NoOutsider,
		KnoxNoLuckyAccident:      s.Knox.NoLuckyAccident,
		KnoxDetectiveNotCulprit:  s.Knox.DetectiveNotCulprit,
		KnoxCluesShown:           s.Knox.CluesShown,
		KnoxNarratorHidesNothing: s.Knox.NarratorHidesNothing,
		KnoxNoUnannouncedTwins:   s.Knox.NoUnannouncedTwins,
	})
	if err != nil {
		return nil, fmt.Errorf("create mystery: %w", err)
	}

	return new(toMysteryRow(mysteryJoinRow(created))), nil
}

func (r *mysteryDAO) AddClue(ctx context.Context, s spec.NewMysteryClue, tx ...*sql.Tx) (*dto.MysteryClue, error) {
	created, err := genQueries(r.db, tx).AddMysteryClue(ctx, sqlcgen.AddMysteryClueParams{
		MysteryID: s.MysteryID,
		Body:      s.Body,
		TruthType: s.TruthType,
		SortOrder: int32(s.SortOrder),
		PlayerID:  s.PlayerID,
	})
	if err != nil {
		return nil, fmt.Errorf("add clue: %w", err)
	}

	return new(toMysteryClue(mysteryClueRow(created))), nil
}

func (r *mysteryDAO) Update(ctx context.Context, s spec.MysteryOwnerUpdate, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).UpdateMystery(ctx, sqlcgen.UpdateMysteryParams{
		Title:      s.Title,
		Body:       s.Body,
		Difficulty: s.Difficulty,
		ID:         s.ID,
		UserID:     s.UserID,
	})
	if err != nil {
		return fmt.Errorf("update mystery: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("mystery not found or not owned")
	}

	return nil
}

func (r *mysteryDAO) UpdateAsAdmin(ctx context.Context, s spec.MysteryUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateMysteryAsAdmin(ctx, sqlcgen.UpdateMysteryAsAdminParams{
		Title:                    s.Title,
		Body:                     s.Body,
		Difficulty:               s.Difficulty,
		FreeForAll:               s.FreeForAll,
		KeepOpenAfterSolve:       s.KeepOpenAfterSolve,
		KnoxCulpritNamedEarly:    s.Knox.CulpritNamedEarly,
		KnoxNoSupernatural:       s.Knox.NoSupernatural,
		KnoxPassagesDeclared:     s.Knox.PassagesDeclared,
		KnoxNoUnknownPoison:      s.Knox.NoUnknownPoison,
		KnoxNoOutsider:           s.Knox.NoOutsider,
		KnoxNoLuckyAccident:      s.Knox.NoLuckyAccident,
		KnoxDetectiveNotCulprit:  s.Knox.DetectiveNotCulprit,
		KnoxCluesShown:           s.Knox.CluesShown,
		KnoxNarratorHidesNothing: s.Knox.NarratorHidesNothing,
		KnoxNoUnannouncedTwins:   s.Knox.NoUnannouncedTwins,
		ID:                       s.ID,
	})
	if err != nil {
		return fmt.Errorf("update mystery as admin: %w", err)
	}

	return nil
}

func (r *mysteryDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*model.MysteryRow, error) {
	row, err := genQueries(r.db, tx).GetMysteryByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get mystery: %w", err)
	}

	return new(toMysteryRow(row)), nil
}

func (r *mysteryDAO) List(ctx context.Context, s spec.MysteryListFilter, tx ...*sql.Tx) ([]model.MysteryRow, int, error) {
	queries := genQueries(r.db, tx)

	solved := sql.NullBool{}
	if s.Solved != nil {
		solved = sql.NullBool{Bool: *s.Solved, Valid: true}
	}

	exclude := joinUUIDs(s.ExcludeUserIDs)

	total, err := queries.CountMysteriesFiltered(ctx, sqlcgen.CountMysteriesFilteredParams{
		Solved:         solved,
		ExcludeUserIds: exclude,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count mysteries: %w", err)
	}

	var rows []mysteryJoinRow

	switch s.Sort {
	case "old":
		found, err := queries.ListMysteriesByOld(ctx, sqlcgen.ListMysteriesByOldParams{
			Solved:         solved,
			ExcludeUserIds: exclude,
			RowLimit:       int32(s.Limit),
			RowOffset:      int32(s.Offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list mysteries: %w", err)
		}

		for _, row := range found {
			rows = append(rows, mysteryJoinRow(row))
		}
	default:
		found, err := queries.ListMysteriesByNew(ctx, sqlcgen.ListMysteriesByNewParams{
			Solved:         solved,
			ExcludeUserIds: exclude,
			RowLimit:       int32(s.Limit),
			RowOffset:      int32(s.Offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list mysteries: %w", err)
		}

		for _, row := range found {
			rows = append(rows, mysteryJoinRow(row))
		}
	}

	var result []model.MysteryRow
	for _, row := range rows {
		result = append(result, toMysteryRow(row))
	}

	return result, int(total), nil
}

func (r *mysteryDAO) GetClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryClue, error) {
	rows, err := genQueries(r.db, tx).GetMysteryClues(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("get clues: %w", err)
	}

	var clues []dto.MysteryClue
	for _, row := range rows {
		clues = append(clues, toMysteryClue(row))
	}

	return clues, nil
}

func (r *mysteryDAO) DeleteClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteMysteryClues(ctx, mysteryID); err != nil {
		return fmt.Errorf("delete clues: %w", err)
	}

	return nil
}

func (r *mysteryDAO) DeleteClue(ctx context.Context, clueID int, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteMysteryClue(ctx, int64(clueID)); err != nil {
		return fmt.Errorf("delete clue: %w", err)
	}

	return nil
}

func (r *mysteryDAO) UpdateClue(ctx context.Context, s spec.MysteryClueUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateMysteryClue(ctx, sqlcgen.UpdateMysteryClueParams{
		Body: s.Body,
		ID:   int64(s.ClueID),
	})
	if err != nil {
		return fmt.Errorf("update clue: %w", err)
	}

	return nil
}

func (r *mysteryDAO) CreateAttempt(ctx context.Context, s spec.NewMysteryAttempt, tx ...*sql.Tx) (*model.MysteryAttemptRow, error) {
	created, err := genQueries(r.db, tx).CreateMysteryAttempt(ctx, sqlcgen.CreateMysteryAttemptParams{
		MysteryID: s.MysteryID,
		UserID:    s.UserID,
		ParentID:  s.ParentID,
		Body:      s.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("create attempt: %w", err)
	}

	return new(toMysteryAttemptRow(mysteryAttemptJoinRow(created))), nil
}

func (r *mysteryDAO) DeleteAttempt(ctx context.Context, s spec.MysteryAttemptDeletion, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).DeleteMysteryAttempt(ctx, sqlcgen.DeleteMysteryAttemptParams{
		ID:     s.ID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete attempt: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("attempt not found or not owned")
	}

	return nil
}

func (r *mysteryDAO) DeleteAttemptAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).DeleteMysteryAttemptAsAdmin(ctx, id); err != nil {
		return fmt.Errorf("admin delete attempt: %w", err)
	}

	return nil
}

func (r *mysteryDAO) GetAttempts(ctx context.Context, s spec.MysteryAttemptQuery, tx ...*sql.Tx) ([]model.MysteryAttemptRow, error) {
	rows, err := genQueries(r.db, tx).GetMysteryAttempts(ctx, sqlcgen.GetMysteryAttemptsParams{
		UserID:    s.ViewerID,
		MysteryID: s.MysteryID,
	})
	if err != nil {
		return nil, fmt.Errorf("get attempts: %w", err)
	}

	var result []model.MysteryAttemptRow
	for _, row := range rows {
		result = append(result, toMysteryAttemptRow(row))
	}

	return result, nil
}

func (r *mysteryDAO) GetAttemptAuthorID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	authorID, err := genQueries(r.db, tx).GetMysteryAttemptAuthorID(ctx, attemptID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get attempt author: %w", err)
	}

	return authorID, nil
}

func (r *mysteryDAO) GetAttemptMysteryID(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	mysteryID, err := genQueries(r.db, tx).GetMysteryAttemptMysteryID(ctx, attemptID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get attempt mystery: %w", err)
	}

	return mysteryID, nil
}

func (r *mysteryDAO) VoteAttempt(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error {
	return r.attemptVotes.Vote(ctx, s, tx...)
}

func (r *mysteryDAO) GetAttemptOwner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error) {
	row, err := genQueries(r.db, tx).GetMysteryAttemptOwner(ctx, attemptID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get attempt for winner: %w", err)
	}

	return row.UserID, row.MysteryID, nil
}

func (r *mysteryDAO) SetMysteryWinner(ctx context.Context, s spec.MysteryWinner, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetMysteryWinner(ctx, sqlcgen.SetMysteryWinnerParams{
		WinnerID: &s.WinnerID,
		ID:       s.MysteryID,
	})
	if err != nil {
		return fmt.Errorf("mark solved: %w", err)
	}

	return nil
}

func (r *mysteryDAO) SetAttemptWinner(ctx context.Context, attemptID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).SetMysteryAttemptWinner(ctx, attemptID); err != nil {
		return fmt.Errorf("set winning attempt: %w", err)
	}

	return nil
}

func (r *mysteryDAO) MarkPermanentlySolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).MarkMysteryPermanentlySolved(ctx, mysteryID)
	if err != nil {
		return fmt.Errorf("mark permanently solved: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("mystery not found or already solved")
	}

	return nil
}

func (r *mysteryDAO) UserHasWinningAttempt(ctx context.Context, s spec.MysterySolverQuery, tx ...*sql.Tx) (bool, error) {
	exists, err := genQueries(r.db, tx).UserHasWinningMysteryAttempt(ctx, sqlcgen.UserHasWinningMysteryAttemptParams{
		MysteryID: s.MysteryID,
		UserID:    s.UserID,
	})
	if err != nil {
		return false, fmt.Errorf("check user winning attempt: %w", err)
	}

	return exists, nil
}

func (r *mysteryDAO) GetSolverIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).GetMysterySolverIDs(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("get solver ids: %w", err)
	}

	return ids, nil
}

func (r *mysteryDAO) IsSolved(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	solved, err := genQueries(r.db, tx).IsMysterySolved(ctx, mysteryID)
	if err != nil {
		return false, fmt.Errorf("check mystery solved: %w", err)
	}

	return solved, nil
}

func (r *mysteryDAO) IsPaused(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (bool, error) {
	paused, err := genQueries(r.db, tx).IsMysteryPaused(ctx, mysteryID)
	if err != nil {
		return false, fmt.Errorf("check mystery paused: %w", err)
	}

	return paused, nil
}

func (r *mysteryDAO) SetPaused(ctx context.Context, s spec.MysteryPauseUpdate, tx ...*sql.Tx) error {
	if s.Paused {
		if err := genQueries(r.db, tx).SetMysteryPaused(ctx, s.MysteryID); err != nil {
			return fmt.Errorf("set mystery paused: %w", err)
		}

		return nil
	}

	if err := genQueries(r.db, tx).SetMysteryUnpaused(ctx, s.MysteryID); err != nil {
		return fmt.Errorf("set mystery unpaused: %w", err)
	}

	return nil
}

func (r *mysteryDAO) SetGmAway(ctx context.Context, s spec.MysteryGmAwayUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).SetMysteryGmAway(ctx, sqlcgen.SetMysteryGmAwayParams{
		GmAway: s.Away,
		ID:     s.MysteryID,
	})
	if err != nil {
		return fmt.Errorf("set mystery gm_away: %w", err)
	}

	return nil
}

func (r *mysteryDAO) CountAttempts(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountMysteryAttempts(ctx, mysteryID)

	return int(count), err
}

func (r *mysteryDAO) CountClues(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountMysteryClues(ctx, mysteryID)

	return int(count), err
}

func (r *mysteryDAO) GetPlayerIDs(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]uuid.UUID, error) {
	ids, err := genQueries(r.db, tx).GetMysteryPlayerIDs(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("get player ids: %w", err)
	}

	return ids, nil
}

func (r *mysteryDAO) ListByUser(ctx context.Context, s spec.MysteryUserListFilter, tx ...*sql.Tx) ([]model.MysteryRow, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountUserMysteries(ctx, s.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count user mysteries: %w", err)
	}

	rows, err := queries.ListMysteriesByUser(ctx, sqlcgen.ListMysteriesByUserParams{
		UserID: s.UserID,
		Limit:  int32(s.Limit),
		Offset: int32(s.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user mysteries: %w", err)
	}

	var result []model.MysteryRow
	for _, row := range rows {
		result = append(result, toMysteryRow(mysteryJoinRow(row)))
	}

	return result, int(total), nil
}

func (r *mysteryDAO) GetLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := genQueries(r.db, tx).GetMysteryLeaderboard(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}

	var result []model.LeaderboardEntry
	for _, row := range rows {
		result = append(result, model.LeaderboardEntry{
			UserID:          row.UserID,
			Username:        row.Username,
			DisplayName:     row.DisplayName,
			AvatarURL:       row.AvatarUrl,
			Role:            row.Role,
			Score:           int(row.Score),
			EasySolved:      int(row.EasySolved),
			MediumSolved:    int(row.MediumSolved),
			HardSolved:      int(row.HardSolved),
			NightmareSolved: int(row.NightmareSolved),
			ScoreAdjustment: int(row.ScoreAdjustment),
		})
	}

	return result, nil
}

func (r *mysteryDAO) GetTopDetectiveIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).GetTopMysteryDetectiveIDs(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, id := range rows {
		ids = append(ids, id.String())
	}

	return ids, nil
}

func (r *mysteryDAO) GetGMLeaderboard(ctx context.Context, limit int, tx ...*sql.Tx) ([]model.GMLeaderboardEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := genQueries(r.db, tx).GetMysteryGMLeaderboard(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("get gm leaderboard: %w", err)
	}

	var result []model.GMLeaderboardEntry
	for _, row := range rows {
		result = append(result, model.GMLeaderboardEntry{
			UserID:       row.UserID,
			Username:     row.Username,
			DisplayName:  row.DisplayName,
			AvatarURL:    row.AvatarUrl,
			Role:         row.Role,
			Score:        int(row.Score),
			MysteryCount: int(row.MysteryCount),
			PlayerCount:  int(row.PlayerCount),
		})
	}

	return result, nil
}

func (r *mysteryDAO) GetTopGMIDs(ctx context.Context, tx ...*sql.Tx) ([]string, error) {
	rows, err := genQueries(r.db, tx).GetTopMysteryGMIDs(ctx)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, id := range rows {
		ids = append(ids, id.String())
	}

	return ids, nil
}

func (r *mysteryDAO) AddAttachment(ctx context.Context, s spec.NewMysteryAttachment, tx ...*sql.Tx) (int64, error) {
	id, err := genQueries(r.db, tx).AddMysteryAttachment(ctx, sqlcgen.AddMysteryAttachmentParams{
		MysteryID: s.MysteryID,
		FileUrl:   s.FileURL,
		FileName:  s.FileName,
		FileSize:  int32(s.FileSize),
	})
	if err != nil {
		return 0, fmt.Errorf("add attachment: %w", err)
	}

	return id, nil
}

func (r *mysteryDAO) DeleteAttachment(ctx context.Context, s spec.MysteryAttachmentDeletion, tx ...*sql.Tx) error {
	n, err := genQueries(r.db, tx).DeleteMysteryAttachment(ctx, sqlcgen.DeleteMysteryAttachmentParams{
		ID:        s.ID,
		MysteryID: s.MysteryID,
	})
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("attachment not found")
	}

	return nil
}

func (r *mysteryDAO) GetAttachments(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]dto.MysteryAttachment, error) {
	rows, err := genQueries(r.db, tx).GetMysteryAttachments(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("get attachments: %w", err)
	}

	var attachments []dto.MysteryAttachment
	for _, row := range rows {
		attachments = append(attachments, dto.MysteryAttachment{
			ID:       int(row.ID),
			FileURL:  row.FileUrl,
			FileName: row.FileName,
			FileSize: int(row.FileSize),
		})
	}

	return attachments, nil
}

func (r *mysteryDAO) GetAttachmentPaths(ctx context.Context, mysteryID uuid.UUID, tx ...*sql.Tx) ([]string, error) {
	urls, err := genQueries(r.db, tx).GetMysteryAttachmentPaths(ctx, mysteryID)
	if err != nil {
		return nil, fmt.Errorf("get mystery attachment paths: %w", err)
	}

	var paths []string
	for _, fileURL := range urls {
		if fileURL != "" {
			paths = append(paths, fileURL)
		}
	}

	return paths, nil
}
