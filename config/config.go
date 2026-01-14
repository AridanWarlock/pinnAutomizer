package config

import (
	"pinnAutomizer/internal/adapter/kafka_produce"
	"pinnAutomizer/internal/adapter/postgres"
	"pinnAutomizer/internal/task/update_task_status_after_train"
	"pinnAutomizer/internal/task/update_task_status_on_train"
	"pinnAutomizer/pkg/httpserver"
	"pinnAutomizer/pkg/jwt"
	"pinnAutomizer/pkg/log"

	"github.com/ilyakaznacheev/cleanenv"
)

type App struct {
	Name    string `env:"APP_NAME" required:"true"`
	Version string `env:"APP_VERSION" required:"true"`
	Env     string `env:"ENVIRONMENT" required:"true"`
}

type Config struct {
	App      App
	HTTP     httpserver.Config
	Log      log.Config
	Postgres postgres.Config
	Jwt      jwt.Config

	KafkaProducer           kafka_produce.Config
	KafkaConsumerOnTrain    update_task_status_on_train.Config
	KafkaConsumerAfterTrain update_task_status_after_train.Config
}

func InitConfig() (Config, error) {
	c := Config{}
	err := cleanenv.ReadConfig(".env", &c)

	return c, err
}
