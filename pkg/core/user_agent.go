package core

type UserAgent string

func NewUserAgent(agent string) (UserAgent, error) {
	a := UserAgent(agent)
	if err := a.Validate(); err != nil {
		return "", err
	}

	return UserAgent(agent), nil
}

func (a UserAgent) Validate() error {
	if a == "" {
		return ErrInvalidUserAgent
	}
	return nil
}
