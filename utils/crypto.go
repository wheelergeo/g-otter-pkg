package utils

import "golang.org/x/crypto/bcrypt"

func EncryptPwd(pwd string) (hash string, err error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	hash = string(h)
	return
}

func VerifyPwd(hash string, pwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	if err != nil {
		return false
	}
	return true
}
