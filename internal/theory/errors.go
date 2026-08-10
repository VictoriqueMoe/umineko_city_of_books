package theory

import "errors"

var (
	ErrCannotRespondToOwnTheory = errors.New("you cannot respond to your own theory")
	ErrRateLimited              = errors.New("daily limit reached")
	ErrTheoryNotFound           = errors.New("theory not found")
	ErrNotAuthor                = errors.New("only the theory author or a moderator can do this")
	ErrAlreadyRefuted           = errors.New("this theory has already been refuted")
	ErrResponseNotOnTheory      = errors.New("that response is not on this theory")
	ErrRefutationMustOppose     = errors.New("only a response that argues against the theory can refute it")
	ErrRefutationMustBeTopLevel = errors.New("only a top level response can refute a theory")
	ErrCannotRefuteWithOwn      = errors.New("you cannot refute your own theory with your own response")
)
