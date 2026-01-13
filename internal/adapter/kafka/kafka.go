package kafka

type Config struct {
}

type Kafka struct {
}

func New(c Config) (*Kafka, error) {
	return &Kafka{}, nil
}

func (k *Kafka) Close() {

}
