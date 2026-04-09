package kafkaProducer

type Config struct {
	Addr string `env:"ADDR,required"`
}
