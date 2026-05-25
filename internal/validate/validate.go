package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
    validate *validator.Validate
}

type validationError map[string]string

func (v validationError) Error() string {
    parts := make([]string, 0, len(v))
    for field, msg := range v {
        parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
    }
    return strings.Join(parts, ", ")
}

func New() *Validator {
    v := validator.New(validator.WithRequiredStructEnabled())

    v.RegisterTagNameFunc(func(f reflect.StructField) string {
        tag := f.Tag.Get("json")
        name, _, _ := strings.Cut(tag, ",")
        if name == "-" || name == "" {
            return f.Name
        }
        return name
    })

    return &Validator{validate: v}
}

func (v *Validator) Validate(data any) error {
    err := v.validate.Struct(data)
    if err == nil {
        return nil
    }

    var validationErrors validator.ValidationErrors
    if !errors.As(err, &validationErrors) {
        return err
    }

    errs := validationError{}
    for _, e := range validationErrors {
        field := e.Field()
        switch e.Tag() {
        case "required":
            errs[field] = "required camp"
        case "email":
            errs[field] = "invalid email address"
        case "gte":
            errs[field] = fmt.Sprintf("must be greater or equal than %s", e.Param())
        default:
            errs[field] = fmt.Sprintf("invalid value (rule: %s)", e.Tag())
        }
    }

    return errs
}