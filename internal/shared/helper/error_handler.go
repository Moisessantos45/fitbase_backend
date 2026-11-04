package helper

import (
	"context"
	"errors"
	"fitbase/internal/shared/errorsx"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func TranslateDBError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return errorsx.ErrNotFound
	case errors.Is(err, context.Canceled):
		return errorsx.ErrCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return errorsx.ErrTimeout
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23505":
			return errorsx.ErrAlreadyExists
		case "23503":
			return errorsx.ErrInvalidReference
		case "23502", // not_null_violation
			"23514", // check_violation
			"22001", // string_data_right_truncation (texto demasiado largo)
			"22003", // numeric_value_out_of_range
			"22P02", // invalid_text_representation (p. ej. UUID mal formado)
			"22007", // invalid_datetime_format
			"22008": // datetime_field_overflow
			return errorsx.ErrInvalidData
		}
	}

	return fmt.Errorf("repository operation failed: %w", err)
}
