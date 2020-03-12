package databox

import (
	"fmt"
	"regexp"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/databox/parse"
)

func DataBoxJobID(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return
	}

	if _, err := parse.DataBoxJobID(v); err != nil {
		errors = append(errors, fmt.Errorf("Can not parse %q as a resource id: %v", k, err))
		return
	}

	return warnings, errors
}

func validateDataBoxJobName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\da-zA-Z][-\da-zA-Z]{1,22}[\da-zA-Z]$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be between 3 and 24 characters in length, and it must begin and end with an alphanumeric and can only contain alphanumeric characters and hyphens", k))
	}

	return warnings, errors
}

func validateDataBoxJobContactName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\S][\s\S]{1,32}[\S]$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be between 3 and 34 characters in length", k))
	}

	return warnings, errors
}

func validateDataBoxJobPhoneNumber(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[+][\d]{2,}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must begin with + and may contain only at least 2 numbers", k))
	}

	return warnings, errors
}

func validateDataBoxJobEmail(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be with @", k))
	}

	return warnings, errors
}

func validateDataBoxJobPhoneExtension(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\d]{0,4}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must have maximum 4 characters in length and can only contain numbers", k))
	}

	return warnings, errors
}

func validateDataBoxJobStreetAddress(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\s\S]{1,35}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be between 1 and 35 characters in length", k))
	}

	return warnings, errors
}

func validateDataBoxJobPostCode(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\s\S]{1,9}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be between 1 and 9 characters in length", k))
	}

	return warnings, errors
}

func validateDataBoxJobCity(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\s\S]{2,30}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be between 2 and 30 characters in length", k))
	}

	return warnings, errors
}

func validateDataBoxJobCompanyName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if !regexp.MustCompile(`^[\s\S]{2,35}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be between 2 and 35 characters in length", k))
	}

	return warnings, errors
}

func validateDataBoxJobDiskPassKey(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if len(value) < 12 || len(value) > 32 {
		errors = append(errors, fmt.Errorf(
			"%q must be between 12 and 32 characters: %q", k, value))
	}

	if !regexp.MustCompile(`^.*[\d]+.*[^a-zA-Z0-9 ]+.*|.*[^a-zA-Z0-9 ]+.*[\d]+.*$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%q must be alphanumeric, contain at least one special character and atleast one number", k))
	}

	return warnings, errors
}
