package config

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/kafka_produce"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/redis"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/auth/jwt/access_token"
	"github.com/AridanWarlock/pinnAutomizer/pkg/auth/refresh_token"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/ilyakaznacheev/cleanenv"
)

type App struct {
	Env string `env:"ENVIRONMENT" env-required:"true"`
}

type Config struct {
	App                   App
	HTTP                  core_http_server.Config
	Log                   logger.Config
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

	return c, err
}
