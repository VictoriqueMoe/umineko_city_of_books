package admin

import "errors"

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrProtectedUser           = errors.New("this user cannot be modified")
	ErrBotAccountProtected     = errors.New("this action is not available for bot accounts")
	ErrUnknownRole             = errors.New("unknown role")
	ErrRoleOutranksActor       = errors.New("cannot grant a role equal to or above your own")
	ErrSystemRole              = errors.New("cannot modify system role assignments")
	ErrVanityRoleNotFound      = errors.New("vanity role not found")
	ErrImmutableRole           = errors.New("this role's permissions cannot be edited")
	ErrUnknownPermission       = errors.New("unknown permission")
	ErrStaffPermission         = errors.New("this permission cannot be granted to a vanity role")
	ErrRestrictedPermission    = errors.New("this permission is reserved for admins and cannot be granted to another role")
	ErrVanityRoleOptInLocked   = errors.New("this role is the one members opt in to for characters; turn off Restrict To Chatbot Permission, or switch characters off entirely, before deleting it")
	ErrNoEmailAddress          = errors.New("your account has no email address set")
	ErrEmptyDisplayName        = errors.New("display name is required")
	ErrBannedGiphyInvalidInput = errors.New("could not recognise a Giphy URL or ID in the input")
	ErrBannedGiphyKindMismatch = errors.New("supplied kind does not match what was extracted from the URL")
)
