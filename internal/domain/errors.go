package domain

import "errors"

var (
	ErrAmountInvalid          = errors.New("amount must be greater than 0")
	ErrTypeInvalid            = errors.New("type must be either 'income' or 'expense'")
	ErrDateFormatInvalid      = errors.New("invalid date format, use ISO 8601")
	ErrDateRangeStartInvalid  = errors.New("invalid start date format, use YYYY-MM-DD")
	ErrDateRangeEndInvalid    = errors.New("invalid end date format, use YYYY-MM-DD")
	ErrDateRangeOrder         = errors.New("end date must be greater than or equal to start date")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrCategoryWrongFamily    = errors.New("category does not belong to this family")
	ErrTransactionNotOwner    = errors.New("unauthorized: you can only update/delete your own transactions")
	ErrFamilyNameEmpty        = errors.New("family name cannot be empty")
	ErrAlreadyInFamily        = errors.New("user already in a family")
	ErrAlreadyAMemberOfFamily = errors.New("already a member of a family")
	ErrInvalidInviteCode      = errors.New("invalid invite code")
)
