package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/role"
	"umineko_city_of_books/internal/text"
	"umineko_city_of_books/internal/utils"

	"github.com/google/uuid"
)

var (
	ErrRefutationRejected = errors.New("the response cannot refute this theory")
)

type (
	TheoryDAO interface {
		InsertTheory(ctx context.Context, s spec.NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error)
		InsertTheoryEvidence(ctx context.Context, s spec.NewTheoryEvidence, tx ...*sql.Tx) (*dto.EvidenceResponse, error)
		GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error)
		List(ctx context.Context, q spec.TheoryListFilter, tx ...*sql.Tx) ([]dto.TheoryResponse, int, error)
		UpdateTheory(ctx context.Context, s spec.TheoryUpdate, tx ...*sql.Tx) error
		ReplaceTheoryEvidence(ctx context.Context, s spec.TheoryEvidenceReplacement, tx ...*sql.Tx) error
		Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetEvidence(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error)
		InsertResponse(ctx context.Context, s spec.NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error)
		InsertResponseEvidence(ctx context.Context, s spec.NewResponseEvidence, tx ...*sql.Tx) (*dto.EvidenceResponse, error)
		DeleteResponse(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error
		DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error
		GetResponses(ctx context.Context, q spec.TheoryResponseQuery, tx ...*sql.Tx) ([]dto.ResponseResponse, error)
		GetResponseEvidence(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error)
		VoteTheory(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error
		VoteResponse(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error
		GetUserTheoryVote(ctx context.Context, q spec.TheoryVoteLookup, tx ...*sql.Tx) (int, error)
		GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error)
		GetResponseInfo(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (authorID uuid.UUID, theoryID uuid.UUID, err error)
		GetRecentActivityByUser(ctx context.Context, q spec.UserActivityQuery, tx ...*sql.Tx) ([]dto.ActivityItem, int, error)
		CountUserTheoriesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		CountUserResponsesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error)
		UpdateCredibilityScore(ctx context.Context, s spec.TheoryCredibilityUpdate, tx ...*sql.Tx) error
		GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (withLoveSum float64, withoutLoveSum float64, err error)
		RecomputeStatus(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) error
		MarkRefuted(ctx context.Context, s spec.TheoryRefutation, tx ...*sql.Tx) error
		GetResponseMeta(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (model.ResponseMeta, error)
		SetEvidenceTruthWeight(ctx context.Context, s spec.EvidenceTruthWeightUpdate, tx ...*sql.Tx) error
		GetTheoryTitle(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error)
		GetTheorySeries(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error)
	}

	theoryDAO struct {
		db            *sql.DB
		theoryVotes   *voteDAO
		responseVotes *voteDAO
	}

	theorySideCounts struct {
		withLove    int
		withoutLove int
	}

	theoryListRowVariant interface {
		sqlcgen.ListTheoriesByNewRow |
			sqlcgen.ListTheoriesByOldRow |
			sqlcgen.ListTheoriesByPopularRow |
			sqlcgen.ListTheoriesByPopularAscRow |
			sqlcgen.ListTheoriesByControversialRow |
			sqlcgen.ListTheoriesByControversialAscRow |
			sqlcgen.ListTheoriesByCredibilityRow |
			sqlcgen.ListTheoriesByCredibilityAscRow
	}

	theoryEvidenceRow = sqlcgen.ListTheoryEvidenceRow
	theoryResponseRow = sqlcgen.ListTheoryResponsesRow
	theoryListRow     = sqlcgen.ListTheoriesByNewRow
	theoryListParams  = sqlcgen.ListTheoriesByNewParams
)

func theoryQuoteIndexParam(v *int) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}

	return sql.NullInt32{Int32: int32(*v), Valid: true}
}

func theoryQuoteIndexValue(v sql.NullInt32) *int {
	if !v.Valid {
		return nil
	}

	return new(int(v.Int32))
}

func toTheoryEvidenceResponse(row theoryEvidenceRow) dto.EvidenceResponse {
	return dto.EvidenceResponse{
		ID:         int(row.ID),
		AudioID:    row.AudioID,
		QuoteIndex: theoryQuoteIndexValue(row.QuoteIndex),
		Note:       row.Note,
		Lang:       row.Lang,
		SortOrder:  int(row.SortOrder),
	}
}

func toTheoryResponseResponse(row theoryResponseRow) dto.ResponseResponse {
	return dto.ResponseResponse{
		ID:        row.ID,
		ParentID:  row.ParentID,
		Side:      row.Side,
		Body:      row.Body,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		Author: dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarUrl,
			Role:        role.Role(row.AuthorRole),
		},
	}
}

func toTheoryListResponse(row theoryListRow) dto.TheoryResponse {
	t := dto.TheoryResponse{
		ID:               row.ID,
		Title:            row.Title,
		Body:             row.Body,
		Episode:          int(row.Episode),
		Series:           row.Series,
		CredibilityScore: float64(row.CredibilityScore),
		Status:           dto.TheoryStatus(row.Status),
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
		Author: dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarUrl,
			Role:        role.Role(row.AuthorRole),
		},
	}

	if clipped := text.ClampRunes(t.Body, 200); len(clipped) != len(t.Body) {
		t.Body = clipped + "..."
	}

	return t
}

func theoryListRows[T theoryListRowVariant](in []T) []theoryListRow {
	out := make([]theoryListRow, 0, len(in))
	for _, row := range in {
		out = append(out, theoryListRow(row))
	}

	return out
}

func theoryRowsBySort(ctx context.Context, queries *sqlcgen.Queries, sort string, args theoryListParams) ([]theoryListRow, error) {
	switch sort {
	case "popular":
		rows, err := queries.ListTheoriesByPopular(ctx, sqlcgen.ListTheoriesByPopularParams(args))

		return theoryListRows(rows), err
	case "popular_asc":
		rows, err := queries.ListTheoriesByPopularAsc(ctx, sqlcgen.ListTheoriesByPopularAscParams(args))

		return theoryListRows(rows), err
	case "controversial":
		rows, err := queries.ListTheoriesByControversial(ctx, sqlcgen.ListTheoriesByControversialParams(args))

		return theoryListRows(rows), err
	case "controversial_asc":
		rows, err := queries.ListTheoriesByControversialAsc(ctx, sqlcgen.ListTheoriesByControversialAscParams(args))

		return theoryListRows(rows), err
	case "credibility":
		rows, err := queries.ListTheoriesByCredibility(ctx, sqlcgen.ListTheoriesByCredibilityParams(args))

		return theoryListRows(rows), err
	case "credibility_asc":
		rows, err := queries.ListTheoriesByCredibilityAsc(ctx, sqlcgen.ListTheoriesByCredibilityAscParams(args))

		return theoryListRows(rows), err
	case "old":
		rows, err := queries.ListTheoriesByOld(ctx, sqlcgen.ListTheoriesByOldParams(args))

		return theoryListRows(rows), err
	default:
		rows, err := queries.ListTheoriesByNew(ctx, args)

		return theoryListRows(rows), err
	}
}

func (r *theoryDAO) InsertTheory(ctx context.Context, s spec.NewTheory, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	series := s.Series
	if series == "" {
		series = "umineko"
	}

	row, err := genQueries(r.db, tx).CreateTheory(ctx, sqlcgen.CreateTheoryParams{
		UserID:  s.UserID,
		Title:   s.Title,
		Body:    s.Body,
		Episode: int32(s.Episode),
		Series:  series,
	})
	if err != nil {
		return nil, fmt.Errorf("insert theory: %w", err)
	}

	created := dto.TheoryDetailResponse{
		ID:               row.ID,
		Title:            row.Title,
		Body:             row.Body,
		Episode:          int(row.Episode),
		Series:           row.Series,
		CredibilityScore: float64(row.CredibilityScore),
		Status:           dto.TheoryStatus(row.Status),
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
		Author: dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarUrl,
			Role:        role.Role(row.AuthorRole),
		},
	}

	return &created, nil
}

func (r *theoryDAO) InsertTheoryEvidence(ctx context.Context, s spec.NewTheoryEvidence, tx ...*sql.Tx) (*dto.EvidenceResponse, error) {
	row, err := genQueries(r.db, tx).CreateTheoryEvidence(ctx, sqlcgen.CreateTheoryEvidenceParams{
		TheoryID:   s.TheoryID,
		AudioID:    s.Evidence.AudioID,
		QuoteIndex: theoryQuoteIndexParam(s.Evidence.QuoteIndex),
		Note:       s.Evidence.Note,
		SortOrder:  int32(s.SortOrder),
		Lang:       langOrDefault(s.Evidence.Lang),
	})
	if err != nil {
		return nil, fmt.Errorf("insert evidence: %w", err)
	}

	return new(toTheoryEvidenceResponse(theoryEvidenceRow(row))), nil
}

func (r *theoryDAO) GetByID(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) (*dto.TheoryDetailResponse, error) {
	row, err := genQueries(r.db, tx).GetTheoryByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get theory: %w", err)
	}

	t := dto.TheoryDetailResponse{
		ID:                  row.ID,
		Title:               row.Title,
		Body:                row.Body,
		Episode:             int(row.Episode),
		Series:              row.Series,
		CredibilityScore:    float64(row.CredibilityScore),
		Status:              dto.TheoryStatus(row.Status),
		RefutedByResponseID: row.RefutedByResponseID,
		CreatedAt:           row.CreatedAt.UTC().Format(time.RFC3339),
		Author: dto.UserResponse{
			ID:          row.AuthorID,
			Username:    row.AuthorUsername,
			DisplayName: row.AuthorDisplayName,
			AvatarURL:   row.AuthorAvatarUrl,
			Role:        role.Role(row.AuthorRole),
		},
	}

	if row.RefutedAt.Valid {
		t.RefutedAt = row.RefutedAt.Time.UTC().Format(time.RFC3339)
	}

	if row.RefuterID != nil {
		t.RefutedBy = &dto.UserResponse{
			ID:          *row.RefuterID,
			Username:    row.RefuterUsername,
			DisplayName: row.RefuterDisplayName,
			AvatarURL:   row.RefuterAvatarUrl,
			Role:        role.Role(row.RefuterRole),
		}
	}

	up, down, err := r.getTheoryVoteCounts(ctx, id, tx...)
	if err != nil {
		return nil, err
	}
	t.VoteScore = up - down

	withLove, withoutLove, err := r.getResponseSideCounts(ctx, id, tx...)
	if err != nil {
		return nil, err
	}
	t.WithLoveCount = withLove
	t.WithoutLoveCount = withoutLove

	return &t, nil
}

func (r *theoryDAO) List(ctx context.Context, q spec.TheoryListFilter, tx ...*sql.Tx) ([]dto.TheoryResponse, int, error) {
	p := q.Params

	queries := genQueries(r.db, tx)
	excluded := joinUUIDs(q.ExcludeUserIDs)

	total, err := queries.CountTheoriesFiltered(ctx, sqlcgen.CountTheoriesFilteredParams{
		Series:         p.Series,
		Episode:        int32(p.Episode),
		AuthorID:       p.AuthorID,
		Search:         p.Search,
		ExcludeUserIds: excluded,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count theories: %w", err)
	}

	rows, err := theoryRowsBySort(ctx, queries, p.Sort, theoryListParams{
		Series:         p.Series,
		Episode:        int32(p.Episode),
		AuthorID:       p.AuthorID,
		Search:         p.Search,
		ExcludeUserIds: excluded,
		RowOffset:      int32(p.Offset),
		RowLimit:       int32(p.Limit),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list theories: %w", err)
	}

	var theories []dto.TheoryResponse
	var ids []uuid.UUID
	for _, row := range rows {
		t := toTheoryListResponse(row)

		theories = append(theories, t)
		ids = append(ids, t.ID)
	}

	voteScores, err := r.theoryVoteScoresBatch(ctx, ids, tx...)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to get theory vote counts")
	}

	sideCounts, err := r.responseSideCountsBatch(ctx, ids, tx...)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to get response side counts")
	}

	var userVotes map[uuid.UUID]int
	if q.ViewerID != uuid.Nil {
		userVotes, err = r.userTheoryVotesBatch(ctx, q.ViewerID, ids, tx...)
		if err != nil {
			logger.Ctx(ctx).Error().Err(err).Msg("failed to get user theory vote")
		}
	}

	for i := range theories {
		theories[i].VoteScore = voteScores[theories[i].ID]

		sides := sideCounts[theories[i].ID]
		theories[i].WithLoveCount = sides.withLove
		theories[i].WithoutLoveCount = sides.withoutLove

		theories[i].UserVote = userVotes[theories[i].ID]
	}

	return theories, int(total), nil
}

func (r *theoryDAO) UpdateTheory(ctx context.Context, s spec.TheoryUpdate, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	var affected int64
	var err error
	if s.AsAdmin {
		affected, err = queries.UpdateTheoryAsAdmin(ctx, sqlcgen.UpdateTheoryAsAdminParams{
			Title:   s.Title,
			Body:    s.Body,
			Episode: int32(s.Episode),
			ID:      s.ID,
		})
	} else {
		affected, err = queries.UpdateTheoryOwned(ctx, sqlcgen.UpdateTheoryOwnedParams{
			Title:   s.Title,
			Body:    s.Body,
			Episode: int32(s.Episode),
			ID:      s.ID,
			UserID:  s.UserID,
		})
	}
	if err != nil {
		return fmt.Errorf("update theory: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("theory not found or not owned by user")
	}

	return nil
}

func (r *theoryDAO) ReplaceTheoryEvidence(ctx context.Context, s spec.TheoryEvidenceReplacement, tx ...*sql.Tx) error {
	queries := genQueries(r.db, tx)

	if err := queries.DeleteTheoryEvidenceByTheory(ctx, s.TheoryID); err != nil {
		return fmt.Errorf("delete old evidence: %w", err)
	}

	for i, ev := range s.Evidence {
		err := queries.AddTheoryEvidence(ctx, sqlcgen.AddTheoryEvidenceParams{
			TheoryID:   s.TheoryID,
			AudioID:    ev.AudioID,
			QuoteIndex: theoryQuoteIndexParam(ev.QuoteIndex),
			Note:       ev.Note,
			SortOrder:  int32(i),
		})
		if err != nil {
			return fmt.Errorf("insert evidence: %w", err)
		}
	}

	return nil
}

func (r *theoryDAO) Delete(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteTheoryOwned(ctx, sqlcgen.DeleteTheoryOwnedParams{
		ID:     s.ID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete theory: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("theory not found or not owned by user")
	}

	return nil
}

func (r *theoryDAO) DeleteAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteTheoryAsAdmin(ctx, id)
	if err != nil {
		return fmt.Errorf("admin delete theory: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("theory not found")
	}

	return nil
}

func (r *theoryDAO) GetEvidence(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error) {
	rows, err := genQueries(r.db, tx).ListTheoryEvidence(ctx, theoryID)
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}

	var evidence []dto.EvidenceResponse
	for _, row := range rows {
		evidence = append(evidence, toTheoryEvidenceResponse(row))
	}

	return evidence, nil
}

func (r *theoryDAO) InsertResponse(ctx context.Context, s spec.NewTheoryResponse, tx ...*sql.Tx) (*dto.ResponseResponse, error) {
	row, err := genQueries(r.db, tx).CreateTheoryResponse(ctx, sqlcgen.CreateTheoryResponseParams{
		TheoryID: s.TheoryID,
		UserID:   s.UserID,
		Side:     s.Side,
		Body:     s.Body,
		ParentID: s.ParentID,
	})
	if err != nil {
		return nil, fmt.Errorf("insert response: %w", err)
	}

	return new(toTheoryResponseResponse(theoryResponseRow(row))), nil
}

func (r *theoryDAO) InsertResponseEvidence(ctx context.Context, s spec.NewResponseEvidence, tx ...*sql.Tx) (*dto.EvidenceResponse, error) {
	row, err := genQueries(r.db, tx).CreateTheoryResponseEvidence(ctx, sqlcgen.CreateTheoryResponseEvidenceParams{
		ResponseID: s.ResponseID,
		AudioID:    s.Evidence.AudioID,
		QuoteIndex: theoryQuoteIndexParam(s.Evidence.QuoteIndex),
		Note:       s.Evidence.Note,
		SortOrder:  int32(s.SortOrder),
		Lang:       langOrDefault(s.Evidence.Lang),
	})
	if err != nil {
		return nil, fmt.Errorf("insert response evidence: %w", err)
	}

	return new(toTheoryEvidenceResponse(theoryEvidenceRow(row))), nil
}

func (r *theoryDAO) DeleteResponse(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteTheoryResponseOwned(ctx, sqlcgen.DeleteTheoryResponseOwnedParams{
		ID:     s.ID,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete response: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("response not found or not owned by user")
	}

	return nil
}

func (r *theoryDAO) DeleteResponseAsAdmin(ctx context.Context, id uuid.UUID, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).DeleteTheoryResponseAsAdmin(ctx, id)
	if err != nil {
		return fmt.Errorf("admin delete response: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("response not found")
	}

	return nil
}

func (r *theoryDAO) GetResponses(ctx context.Context, q spec.TheoryResponseQuery, tx ...*sql.Tx) ([]dto.ResponseResponse, error) {
	rows, err := genQueries(r.db, tx).ListTheoryResponses(ctx, q.TheoryID)
	if err != nil {
		return nil, fmt.Errorf("get responses: %w", err)
	}

	var all []dto.ResponseResponse
	var ids []uuid.UUID
	for _, row := range rows {
		resp := toTheoryResponseResponse(row)

		all = append(all, resp)
		ids = append(ids, resp.ID)
	}

	voteScores, err := r.responseVoteScoresBatch(ctx, ids, tx...)
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Msg("failed to get response vote counts")
	}

	var userVotes map[uuid.UUID]int
	if q.ViewerID != uuid.Nil {
		userVotes, err = r.userResponseVotesBatch(ctx, q.ViewerID, ids, tx...)
		if err != nil {
			logger.Ctx(ctx).Error().Err(err).Msg("failed to get user response vote")
		}
	}

	evidence, err := r.responseEvidenceBatch(ctx, ids, tx...)
	if err != nil {
		return nil, err
	}

	for i := range all {
		all[i].VoteScore = voteScores[all[i].ID]
		all[i].UserVote = userVotes[all[i].ID]
		all[i].Evidence = evidence[all[i].ID]
	}

	return utils.BuildTree(all,
		func(r dto.ResponseResponse) uuid.UUID { return r.ID },
		func(r dto.ResponseResponse) *uuid.UUID { return r.ParentID },
		func(r *dto.ResponseResponse, replies []dto.ResponseResponse) { r.Replies = replies },
	), nil
}

func (r *theoryDAO) GetResponseEvidence(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) ([]dto.EvidenceResponse, error) {
	rows, err := genQueries(r.db, tx).ListTheoryResponseEvidence(ctx, responseID)
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}

	var evidence []dto.EvidenceResponse
	for _, row := range rows {
		evidence = append(evidence, toTheoryEvidenceResponse(theoryEvidenceRow(row)))
	}

	return evidence, nil
}

func (r *theoryDAO) responseEvidenceBatch(ctx context.Context, responseIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID][]dto.EvidenceResponse, error) {
	if len(responseIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListTheoryResponseEvidenceBatch(ctx, joinUUIDs(responseIDs))
	if err != nil {
		return nil, fmt.Errorf("query evidence: %w", err)
	}

	byResponse := make(map[uuid.UUID][]dto.EvidenceResponse, len(responseIDs))
	for _, row := range rows {
		ev := toTheoryEvidenceResponse(theoryEvidenceRow{
			ID:         row.ID,
			AudioID:    row.AudioID,
			QuoteIndex: row.QuoteIndex,
			Note:       row.Note,
			SortOrder:  row.SortOrder,
			Lang:       row.Lang,
		})

		byResponse[row.ResponseID] = append(byResponse[row.ResponseID], ev)
	}

	return byResponse, nil
}

func (r *theoryDAO) VoteTheory(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error {
	return r.theoryVotes.Vote(ctx, s, tx...)
}

func (r *theoryDAO) VoteResponse(ctx context.Context, s spec.Vote, tx ...*sql.Tx) error {
	return r.responseVotes.Vote(ctx, s, tx...)
}

func (r *theoryDAO) GetUserTheoryVote(ctx context.Context, q spec.TheoryVoteLookup, tx ...*sql.Tx) (int, error) {
	value, err := genQueries(r.db, tx).GetUserTheoryVote(ctx, sqlcgen.GetUserTheoryVoteParams{
		UserID:   q.UserID,
		TheoryID: q.TheoryID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	return int(value), err
}

func (r *theoryDAO) getTheoryVoteCounts(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (int, int, error) {
	counts, err := genQueries(r.db, tx).GetTheoryVoteCounts(ctx, theoryID)
	if err != nil {
		return 0, 0, err
	}

	return int(counts.UpVotes), int(counts.DownVotes), nil
}

func (r *theoryDAO) theoryVoteScoresBatch(ctx context.Context, theoryIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]int, error) {
	if len(theoryIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListTheoryVoteScores(ctx, joinUUIDs(theoryIDs))
	if err != nil {
		return nil, fmt.Errorf("batch theory vote counts: %w", err)
	}

	scores := make(map[uuid.UUID]int, len(theoryIDs))
	for _, row := range rows {
		scores[row.TargetID] = int(row.UpVotes - row.DownVotes)
	}

	return scores, nil
}

func (r *theoryDAO) responseVoteScoresBatch(ctx context.Context, responseIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]int, error) {
	if len(responseIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListTheoryResponseVoteScores(ctx, joinUUIDs(responseIDs))
	if err != nil {
		return nil, fmt.Errorf("batch response vote counts: %w", err)
	}

	scores := make(map[uuid.UUID]int, len(responseIDs))
	for _, row := range rows {
		scores[row.TargetID] = int(row.UpVotes - row.DownVotes)
	}

	return scores, nil
}

func (r *theoryDAO) userTheoryVotesBatch(ctx context.Context, userID uuid.UUID, theoryIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]int, error) {
	if len(theoryIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListUserTheoryVotes(ctx, sqlcgen.ListUserTheoryVotesParams{
		UserID:  userID,
		Column2: joinUUIDs(theoryIDs),
	})
	if err != nil {
		return nil, fmt.Errorf("batch user theory votes: %w", err)
	}

	votes := make(map[uuid.UUID]int, len(theoryIDs))
	for _, row := range rows {
		votes[row.TargetID] = int(row.Value)
	}

	return votes, nil
}

func (r *theoryDAO) userResponseVotesBatch(ctx context.Context, userID uuid.UUID, responseIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]int, error) {
	if len(responseIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListUserTheoryResponseVotes(ctx, sqlcgen.ListUserTheoryResponseVotesParams{
		UserID:  userID,
		Column2: joinUUIDs(responseIDs),
	})
	if err != nil {
		return nil, fmt.Errorf("batch user response votes: %w", err)
	}

	votes := make(map[uuid.UUID]int, len(responseIDs))
	for _, row := range rows {
		votes[row.TargetID] = int(row.Value)
	}

	return votes, nil
}

func (r *theoryDAO) getResponseSideCounts(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (int, int, error) {
	counts, err := genQueries(r.db, tx).GetTheoryResponseSideCounts(ctx, theoryID)
	if err != nil {
		return 0, 0, err
	}

	return int(counts.WithLoveCount), int(counts.WithoutLoveCount), nil
}

func (r *theoryDAO) responseSideCountsBatch(ctx context.Context, theoryIDs []uuid.UUID, tx ...*sql.Tx) (map[uuid.UUID]theorySideCounts, error) {
	if len(theoryIDs) == 0 {
		return nil, nil
	}

	rows, err := genQueries(r.db, tx).ListTheoryResponseSideCounts(ctx, joinUUIDs(theoryIDs))
	if err != nil {
		return nil, fmt.Errorf("batch response side counts: %w", err)
	}

	counts := make(map[uuid.UUID]theorySideCounts, len(theoryIDs))
	for _, row := range rows {
		counts[row.TheoryID] = theorySideCounts{
			withLove:    int(row.WithLoveCount),
			withoutLove: int(row.WithoutLoveCount),
		}
	}

	return counts, nil
}

func (r *theoryDAO) GetTheoryAuthorID(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, error) {
	userID, err := genQueries(r.db, tx).GetTheoryAuthorID(ctx, theoryID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get theory author: %w", err)
	}

	return userID, nil
}

func (r *theoryDAO) GetResponseInfo(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (uuid.UUID, uuid.UUID, error) {
	row, err := genQueries(r.db, tx).GetTheoryResponseInfo(ctx, responseID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get response info: %w", err)
	}

	return row.UserID, row.TheoryID, nil
}

func (r *theoryDAO) GetTheorySeries(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	series, err := genQueries(r.db, tx).GetTheorySeries(ctx, theoryID)
	if err != nil {
		return "", fmt.Errorf("get theory series: %w", err)
	}

	return series, nil
}

func (r *theoryDAO) GetTheoryTitle(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (string, error) {
	title, err := genQueries(r.db, tx).GetTheoryTitle(ctx, theoryID)
	if err != nil {
		return "", fmt.Errorf("get theory title: %w", err)
	}

	return title, nil
}

func (r *theoryDAO) GetRecentActivityByUser(ctx context.Context, q spec.UserActivityQuery, tx ...*sql.Tx) ([]dto.ActivityItem, int, error) {
	queries := genQueries(r.db, tx)

	total, err := queries.CountUserTheoryActivity(ctx, q.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count activity: %w", err)
	}

	rows, err := queries.ListUserTheoryActivity(ctx, sqlcgen.ListUserTheoryActivityParams{
		UserID: q.UserID,
		Limit:  int32(q.Limit),
		Offset: int32(q.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("get activity: %w", err)
	}

	var items []dto.ActivityItem
	for _, row := range rows {
		items = append(items, dto.ActivityItem{
			Type:        row.Type,
			TheoryID:    row.TheoryID,
			TheoryTitle: row.TheoryTitle,
			Side:        row.Side,
			Body:        row.Body,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return items, int(total), nil
}

func (r *theoryDAO) CountUserTheoriesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUserTheoriesToday(ctx, userID)

	return int(count), err
}

func (r *theoryDAO) CountUserResponsesToday(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (int, error) {
	count, err := genQueries(r.db, tx).CountUserTheoryResponsesToday(ctx, userID)

	return int(count), err
}

func (r *theoryDAO) UpdateCredibilityScore(ctx context.Context, s spec.TheoryCredibilityUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateTheoryCredibilityScore(ctx, sqlcgen.UpdateTheoryCredibilityScoreParams{
		CredibilityScore: float32(s.Score),
		ID:               s.TheoryID,
	})
	if err != nil {
		return fmt.Errorf("update credibility score: %w", err)
	}

	return nil
}

func (r *theoryDAO) GetResponseEvidenceWeights(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) (float64, float64, error) {
	rows, err := genQueries(r.db, tx).ListTheoryResponseEvidenceWeights(ctx, theoryID)
	if err != nil {
		return 0, 0, fmt.Errorf("get evidence weights: %w", err)
	}

	var withLove, withoutLove float64
	for _, row := range rows {
		if row.Side == "with_love" {
			withLove = row.TruthWeight
		} else {
			withoutLove = row.TruthWeight
		}
	}

	return withLove, withoutLove, nil
}

func (r *theoryDAO) SetEvidenceTruthWeight(ctx context.Context, s spec.EvidenceTruthWeightUpdate, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpdateTheoryResponseEvidenceTruthWeight(ctx, sqlcgen.UpdateTheoryResponseEvidenceTruthWeightParams{
		TruthWeight: float32(s.Weight),
		ID:          int64(s.EvidenceID),
	})
	if err != nil {
		return fmt.Errorf("set evidence truth weight: %w", err)
	}

	return nil
}

func langOrDefault(lang string) string {
	if lang == "" {
		return "en"
	}
	return lang
}

func (r *theoryDAO) RecomputeStatus(ctx context.Context, theoryID uuid.UUID, tx ...*sql.Tx) error {
	if err := genQueries(r.db, tx).RecomputeTheoryStatus(ctx, theoryID); err != nil {
		return fmt.Errorf("recompute theory status: %w", err)
	}

	return nil
}

func (r *theoryDAO) MarkRefuted(ctx context.Context, s spec.TheoryRefutation, tx ...*sql.Tx) error {
	affected, err := genQueries(r.db, tx).MarkTheoryRefuted(ctx, sqlcgen.MarkTheoryRefutedParams{
		ID:   s.TheoryID,
		ID_2: s.ResponseID,
	})
	if err != nil {
		return fmt.Errorf("mark theory refuted: %w", err)
	}

	if affected == 0 {
		return ErrRefutationRejected
	}

	return nil
}

func (r *theoryDAO) GetResponseMeta(ctx context.Context, responseID uuid.UUID, tx ...*sql.Tx) (model.ResponseMeta, error) {
	row, err := genQueries(r.db, tx).GetTheoryResponseMeta(ctx, responseID)
	if err != nil {
		return model.ResponseMeta{}, fmt.Errorf("get response meta: %w", err)
	}

	return model.ResponseMeta{
		AuthorID: row.UserID,
		TheoryID: row.TheoryID,
		Side:     row.Side,
		ParentID: row.ParentID,
	}, nil
}
