package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func NewAccessToken(mods ...mod[domain.AccessToken]) domain.AccessToken {
	token := domain.AccessToken("new.access.token")

	return fixture(token, mods)
}
