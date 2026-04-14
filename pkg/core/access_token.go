package core

import "regexp"

var accessTokenRegexp = regexp.MustCompile("^[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+$")

type AccessToken string

func NewAccessToken(token string) (AccessToken, error) {
	t := AccessToken(token)

	if err := t.Validate(); err != nil {
		return "", err
	}

	return t, nil
}

func (t AccessToken) Validate() error {
	if !accessTokenRegexp.Match([]byte(t)) {
		return ErrInvalidAccessToken
	}
	return nil
}
