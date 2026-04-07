package kafkaProducer

type Config struct {
	Addr []string `env:"WRITER_ADDR"`
}
