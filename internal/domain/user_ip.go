package domain

import (
	"net/netip"
)

type UserIP netip.Addr

func NewUserIP(ip string) (UserIP, error) {
	nip, err := netip.ParseAddr(ip)
	if err != nil {
		return UserIP{}, ErrInvalidIP
	}

	return UserIP(nip), err
}

func (ip UserIP) Validate() error {
	if !netip.Addr(ip).IsValid() {
		return ErrInvalidIP
	}
	return nil
}

func (ip UserIP) String() string {
	return netip.Addr(ip).String()
}
