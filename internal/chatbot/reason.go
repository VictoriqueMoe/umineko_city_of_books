package chatbot

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/repository"
)

type (
	Reason string

	stage string

	outcome struct {
		reason   Reason
		stage    stage
		status   repository.InvocationStatus
		clearsAt time.Time
		detail   string
		err      error
	}
)

const (
	reasonNone            Reason = ""
	reasonNotPermitted    Reason = "not_permitted"
	reasonQuotaUser       Reason = "quota_user"
	reasonQuotaSite       Reason = "quota_site"
	reasonNotConfigured   Reason = "not_configured"
	reasonProviderDown    Reason = "provider_down"
	reasonProviderLimited Reason = "provider_limited"
	reasonTimeout         Reason = "timeout"
	reasonEmptyReply      Reason = "empty_reply"
	reasonFiltered        Reason = "filtered"
	reasonUndeliverable   Reason = "undeliverable"
	reasonInternal        Reason = "internal"

	stagePreTrigger stage = "pre_trigger"
	stagePreModel   stage = "pre_model"
	stagePostModel  stage = "post_model"
)

func (r Reason) policy() bool {
	switch r {
	case reasonNotPermitted, reasonQuotaUser, reasonQuotaSite:
		return true
	default:
		return false
	}
}

func (s *service) noticeText(ctx context.Context, out outcome) string {
	switch out.reason {
	case reasonNotPermitted:
		return s.refusalMessage(ctx)
	case reasonQuotaUser:
		return s.quotaMessage(quotaState{clearsAt: out.clearsAt})
	case reasonQuotaSite:
		return s.quotaMessage(quotaState{global: true, clearsAt: out.clearsAt})
	case reasonProviderLimited:
		return fmt.Sprintf("I am being rate limited and cannot think just now. Try me again %s.", humaniseWait(time.Until(out.clearsAt)))
	case reasonNotConfigured:
		return "I have no voice configured at the moment. A site admin needs to set the chatbot API key before I can answer."
	case reasonTimeout:
		return "I took too long thinking and ran out of time. Try me again."
	case reasonEmptyReply:
		return emptyReplyMessage(out.detail)
	case reasonFiltered:
		return "I wrote an answer that the word filter would not let through. Ask me another way."
	case reasonProviderDown:
		return "I cannot reach my thoughts just now. Try me again in a moment."
	default:
		return "Something went wrong on my side and I could not answer. The staff have been told."
	}
}

func (s *service) refusalMessage(ctx context.Context) string {
	settingsURL := trimBase(s.settingsSvc.Get(ctx, config.SettingBaseURL)) + "/settings"

	return fmt.Sprintf("I am not permitted to answer you yet. Talking to me is opt in, so visit %s and turn it on, then summon me again.", settingsURL)
}
