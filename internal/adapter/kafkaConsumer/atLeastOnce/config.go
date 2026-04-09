package kafkaAtLeastOnceConsumer

type Config struct {
	Broker  string `env:"BROKER,required"`
	GroupID string `env:"GROUP_ID,required"`
}
