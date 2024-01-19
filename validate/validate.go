package validate

import (
	"fmt"

	"github.com/dlclark/regexp2"

	"github.com/cloudwego/hertz/pkg/app/server/binding"
)

// Only for hertz
func Config() *binding.ValidateConfig {
	validateConfig := &binding.ValidateConfig{}
	validateConfig.MustRegValidateFunc("weakPasswd", weakPassword)
	validateConfig.MustRegValidateFunc("mediumPasswd", mediumPassword)
	validateConfig.MustRegValidateFunc("strongPasswd", strongPassword)

	return validateConfig
}

func weakPassword(args ...interface{}) error {
	regex := "^(?=.*[a-zA-Z0-9])[A-Za-z0-9]{8,18}$"
	if len(args) != 1 {
		return fmt.Errorf("The password is null")
	}
	re, err := regexp2.Compile(regex, 0)
	if err != nil {
		return err
	}
	s, _ := args[0].(string)
	existed, err := re.MatchString(s)
	if existed {
		return nil
	}
	return fmt.Errorf("The password does not comply with the rules")
}

func mediumPassword(args ...interface{}) error {
	regex := "^(?=.*[a-zA-Z])(?=.*[0-9])[A-Za-z0-9]{8,18}$"
	if len(args) != 1 {
		return fmt.Errorf("The password is null")
	}
	re, err := regexp2.Compile(regex, 0)
	if err != nil {
		return err
	}
	s, _ := args[0].(string)
	existed, err := re.MatchString(s)
	if existed {
		return nil
	}
	return fmt.Errorf("The password does not comply with the rules")
}

func strongPassword(args ...interface{}) error {
	regex := "^(?=.*[a-zA-Z])(?=.*[0-9])(?=.*[._~!@#$^&*])[A-Za-z0-9._~!@#$^&*]{8,20}$"
	if len(args) != 1 {
		return fmt.Errorf("The password is null")
	}
	re, err := regexp2.Compile(regex, 0)
	if err != nil {
		return err
	}
	s, _ := args[0].(string)
	existed, err := re.MatchString(s)
	if existed {
		return nil
	}
	return fmt.Errorf("The password does not comply with the rules")
}
