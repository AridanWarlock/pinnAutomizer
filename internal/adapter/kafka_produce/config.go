package kafka_produce

type Config struct {
	Addr []string `env:"KAFKA_WRITER_ADDR"`
}
