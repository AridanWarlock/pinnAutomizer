package segmentio

type ReaderConfig struct {
	Broker  string `env:"BROKER,required"`
	GroupID string `env:"GROUP_ID,required"`
}

type WriterConfig struct {
	Broker string `env:"BROKER,required"`
}
