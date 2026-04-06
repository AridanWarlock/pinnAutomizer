package kafka_produce

type Config struct {
	Addr []string `env:"WRITER_ADDR"`
}
