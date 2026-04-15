package core

const (
	FingerprintHeader = "X-Internal-Audit-Fingerprint"
	UserIPHeader      = "X-Internal-Audit-Ip"
	UserAgentHeader   = "X-Internal-Audit-User-Agent"

	JtiHeader      = "X-Internal-Auth-Jti"
	UserIDHeader   = "X-Internal-Auth-User-Id"
	RolesHeader    = "X-Internal-Auth-Roles"
	IssuedAtHeader = "X-Internal-Auth-Issued-At"
)

type ToHeadersSerializable interface {
	ToHeaders() map[string]string
}
