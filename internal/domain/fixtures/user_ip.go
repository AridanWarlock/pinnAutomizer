package fixtures

import (
	"net/netip"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func NewUserIP() domain.UserIP {
	ip := netip.AddrFrom4([4]byte{99, 88, 77, 66})
	return domain.UserIP(ip)
}
