package profile

import (
	"context"
	"testing"
	"unicode/utf8"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateProfile_ClampsPronounsByRunesNotBytes(t *testing.T) {
	cases := []struct {
		name           string
		subject        string
		possessive     string
		wantSubject    string
		wantPossessive string
	}{
		{
			name:           "ascii pronouns within the allowance are untouched",
			subject:        "they",
			possessive:     "their",
			wantSubject:    "they",
			wantPossessive: "their",
		},
		{
			name:           "ascii pronouns over the allowance keep ten characters",
			subject:        "theythemtheirs",
			possessive:     "theirstheirs",
			wantSubject:    "theythemth",
			wantPossessive: "theirsthei",
		},
		{
			name:           "japanese pronouns within the allowance survive whole",
			subject:        "あたくし",
			possessive:     "あたくしの",
			wantSubject:    "あたくし",
			wantPossessive: "あたくしの",
		},
		{
			name:           "japanese pronouns over the allowance keep ten characters",
			subject:        "あたくしあたくしあたくし",
			possessive:     "わたくしわたくしわたくし",
			wantSubject:    "あたくしあたくしあた",
			wantPossessive: "わたくしわたくしわた",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc, userRepo, _, _, _, _ := newTestService(t)
			userID := uuid.New()
			req := dto.UpdateProfileRequest{DisplayName: "Beatrice", PronounSubject: tc.subject, PronounPossessive: tc.possessive}

			expected := req
			expected.DefaultProfileTab = "posts"
			expected.PronounSubject = tc.wantSubject
			expected.PronounPossessive = tc.wantPossessive

			userRepo.EXPECT().GetByID(mock.Anything, userID).Return(&model.User{ID: userID}, nil)
			userRepo.EXPECT().UpdateProfile(mock.Anything, userID, expected).Return(nil)

			// when
			err := svc.UpdateProfile(context.Background(), userID, req)

			// then
			require.NoError(t, err)
			assert.True(t, utf8.ValidString(expected.PronounSubject))
			assert.True(t, utf8.ValidString(expected.PronounPossessive))
			assert.LessOrEqual(t, utf8.RuneCountInString(expected.PronounSubject), maxPronounLength)
			assert.LessOrEqual(t, utf8.RuneCountInString(expected.PronounPossessive), maxPronounLength)
		})
	}
}
