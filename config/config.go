package config

import (
	"fmt"
	"pinnAutomizer/internal/adapter/kafka_produce"
	"pinnAutomizer/internal/adapter/postgres"
	"pinnAutomizer/internal/adapter/redis"
	"pinnAutomizer/pkg/auth/jwt/access_token"
	"pinnAutomizer/pkg/auth/refresh_token"
	"pinnAutomizer/pkg/httpserver"
	"pinnAutomizer/pkg/log"

	"github.com/ilyakaznacheev/cleanenv"
)

type App struct {
	Env string `env:"ENVIRONMENT" env-required:"true"`
}

type Config struct {
	App                   App
	HTTP                  httpserver.Config
	Log                   log.Config
	Postgres              postgres.Config
	Redis                 redis.Config
	AccessTokenGenerator  access_token.Config
	RefreshTokenGenerator refresh_token.Config

	KafkaProducer kafka_produce.Config
	//KafkaConsumerOnTrain    update_task_status_on_train.Config
	//KafkaConsumerAfterTrain update_task_status_after_train.Config
}

func InitConfig() (Config, error) {
	c := Config{}
	err := cleanenv.ReadEnv(&c)

	fmt.Println(c)

	return c, err
}
