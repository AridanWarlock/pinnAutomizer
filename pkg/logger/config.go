package logger

type Config struct {
	Env    string `env:"ENVIRONMENT,required"`
	Level  string `env:"LEVEL,required"`
	Folder string `env:"FOLDER,required"`
}
