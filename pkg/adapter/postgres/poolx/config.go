package poolx

import "time"

type Config struct {
	User     string
	Password string
	Host     string
	Port     int
	DB       string
	SslMode  string
	Timeout  time.Duration
}
