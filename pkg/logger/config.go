package logger

type Config struct {
	Env    string `env:"LOGGER_ENVIRONMENT"`
	Level  string `env:"LOGGER_LEVEL" env-default:"error"`
	Folder string `env:"LOGGER_FOLDER" env-default:"/app/out/logs"`
}
