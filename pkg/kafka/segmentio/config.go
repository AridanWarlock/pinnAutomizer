package segmentio

type ConsumerConfig struct {
	Broker  string `env:"BROKER,required"`
	GroupID string `env:"GROUP_ID,required"`
}

type ProducerConfig struct {
	Broker string `env:"BROKER,required"`
}
