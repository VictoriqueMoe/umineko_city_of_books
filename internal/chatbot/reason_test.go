package chatbot

import (
	"context"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func allReasons() []Reason {
	return []Reason{
		reasonNotPermitted,
		reasonQuotaUser,
		reasonQuotaSite,
		reasonNotConfigured,
		reasonProviderDown,
		reasonProviderLimited,
		reasonTimeout,
		reasonEmptyReply,
		reasonFiltered,
		reasonUndeliverable,
		reasonInternal,
	}
}

func TestReason_PolicyCoversOnlyTheOnesTheMemberCanActOn(t *testing.T) {
	policy := map[Reason]bool{
		reasonNotPermitted: true,
		reasonQuotaUser:    true,
		reasonQuotaSite:    true,
	}

	for _, reason := range append(allReasons(), reasonNone) {
		t.Run(string(reason), func(t *testing.T) {
			// given the reason from the table

			// when
			got := reason.policy()

			// then
			assert.Equal(t, policy[reason], got)
		})
	}
}

func TestNoticeText_EveryReasonExplainsItself(t *testing.T) {
	for _, reason := range allReasons() {
		t.Run(string(reason), func(t *testing.T) {
			// given
			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return("https://whentheycry.social").Maybe()
			svc := &service{settingsSvc: settingsSvc}

			// when
			got := svc.noticeText(context.Background(), outcome{reason: reason, clearsAt: time.Now().Add(time.Hour)})

			// then
			assert.NotEmpty(t, got, "a member must never be told nothing")
			assert.NotContains(t, got, "%!", "the copy must not carry a formatting fault")
		})
	}
}

func TestNoticeText_TheReasonsAMemberCanActOnAreWordedDistinctly(t *testing.T) {
	svc := new(service)
	clearsAt := time.Now().Add(time.Hour)

	distinct := []Reason{
		reasonQuotaUser,
		reasonQuotaSite,
		reasonProviderLimited,
		reasonNotConfigured,
		reasonTimeout,
		reasonEmptyReply,
		reasonFiltered,
		reasonProviderDown,
	}

	seen := make(map[string]Reason, len(distinct))
	for _, reason := range distinct {
		body := svc.noticeText(context.Background(), outcome{reason: reason, clearsAt: clearsAt})

		earlier, clash := seen[body]
		assert.Falsef(t, clash, "%q and %q share wording, so a member cannot tell them apart", earlier, reason)

		seen[body] = reason
	}
}

func TestNoticeText_AnUnknownReasonStillApologises(t *testing.T) {
	// given
	svc := new(service)

	// when
	got := svc.noticeText(context.Background(), outcome{reason: Reason("something_new_nobody_wrote_copy_for")})

	// then
	assert.Equal(t, "Something went wrong on my side and I could not answer. The staff have been told.", got)
}

func TestNoticeText_QuotaCopyDistinguishesTheMemberFromTheSite(t *testing.T) {
	// given
	svc := new(service)
	clearsAt := time.Now().Add(2 * time.Hour)

	// when
	mine := svc.noticeText(context.Background(), outcome{reason: reasonQuotaUser, clearsAt: clearsAt})
	everyones := svc.noticeText(context.Background(), outcome{reason: reasonQuotaSite, clearsAt: clearsAt})

	// then
	assert.Contains(t, mine, "your message limit")
	assert.Contains(t, everyones, "whole site")
	assert.Contains(t, mine, "in about 2 hours")
	assert.Contains(t, everyones, "in about 2 hours")
}

func TestEmptyReplyMessage_SaysWhyThereWasNothing(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"the provider filtered it", openai.IncompleteContentFilter, "thought better of it"},
		{"it ran out of room", openai.IncompleteMaxOutputTokens, "ran out of room"},
		{"no reason given", "", "nothing to say"},
		{"a reason we do not recognise", "some_future_reason", "nothing to say"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given the reason from the table

			// when
			got := emptyReplyMessage(tc.in)

			// then
			assert.Contains(t, got, tc.want)
		})
	}
}

func TestTrimBase(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"already clean", "https://whentheycry.social", "https://whentheycry.social"},
		{"a trailing slash", "https://whentheycry.social/", "https://whentheycry.social"},
		{"surrounding whitespace", "  https://whentheycry.social  ", "https://whentheycry.social"},
		{"whitespace and a slash", "  https://whentheycry.social/  ", "https://whentheycry.social"},
		{"empty", "", ""},
		{"only a slash", "/", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given the raw value from the table

			// when
			got := trimBase(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHumaniseWait(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"already clear", -time.Minute, "shortly"},
		{"seconds away", 30 * time.Second, "in less than a minute"},
		{"a single minute", 70 * time.Second, "in about a minute"},
		{"some minutes", 20 * time.Minute, "in about 20 minutes"},
		{"just under an hour", 59 * time.Minute, "in about 59 minutes"},
		{"an hour", 62 * time.Minute, "in about an hour"},
		{"several hours", 3*time.Hour + 10*time.Minute, "in about 3 hours"},
		{"most of a day", 23 * time.Hour, "in about 23 hours"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given the duration from the table

			// when
			got := humaniseWait(tc.in)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestQuotaMessage_SaysWhoseLimitAndWhenItClears(t *testing.T) {
	cases := []struct {
		name        string
		state       quotaState
		wantPhrases []string
	}{
		{
			name:        "a member out of their own allowance",
			state:       quotaState{over: true, clearsAt: time.Now().Add(2 * time.Hour)},
			wantPhrases: []string{"your message limit", "in about 2 hours"},
		},
		{
			name:        "the whole site out of allowance",
			state:       quotaState{over: true, global: true, clearsAt: time.Now().Add(30 * time.Minute)},
			wantPhrases: []string{"whole site", "in about 30 minutes"},
		},
		{
			name:        "an unknown clearing time still says something useful",
			state:       quotaState{over: true},
			wantPhrases: []string{"shortly"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			svc := new(service)

			// when
			got := svc.quotaMessage(tc.state)

			// then
			for _, phrase := range tc.wantPhrases {
				assert.Contains(t, got, phrase)
			}
		})
	}
}

func TestRefusalMessage_PointsAtTheSettingsPage(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{"a plain base url", "https://whentheycry.social", "https://whentheycry.social/settings"},
		{"a trailing slash is not doubled", "https://whentheycry.social/", "https://whentheycry.social/settings"},
		{"surrounding whitespace is trimmed", "  https://whentheycry.social  ", "https://whentheycry.social/settings"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			settingsSvc := settings.NewMockService(t)
			settingsSvc.EXPECT().Get(mock.Anything, config.SettingBaseURL).Return(tc.baseURL)
			svc := &service{settingsSvc: settingsSvc}

			// when
			got := svc.refusalMessage(context.Background())

			// then
			assert.Contains(t, got, tc.want)
			assert.NotContains(t, got, "//settings")
		})
	}
}
