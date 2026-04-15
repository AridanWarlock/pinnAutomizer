package fixtures

import (
	"net/netip"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewUserIP() core.UserIP {
	ip := netip.AddrFrom4([4]byte{99, 88, 77, 66})
	return core.UserIP(ip)
}
