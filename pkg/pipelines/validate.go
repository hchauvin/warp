package pipelines

import (
	"fmt"
	"github.com/go-playground/validator"
	"reflect"
	"regexp"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	if err := validate.RegisterValidation("name", isName); err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}

var nameRe = regexp.MustCompile(`[A-Za-z0-9_-]+`)

func isName(fl validator.FieldLevel) bool {
	field := fl.Field()

	switch field.Kind() {
	case reflect.String:
		return nameRe.MatchString(field.String())
	case reflect.Slice:
		for i := 0; i < field.Len(); i++ {
			v := field.Index(i)
			if v.Kind() != reflect.String {
				return false
			}
			if !nameRe.MatchString(v.String()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
