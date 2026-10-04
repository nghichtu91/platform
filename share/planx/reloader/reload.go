package reloader

type ReloadHandlerConfig struct {
	ReloadHandler []string `toml:"reload_handler"`
}

var (
	ReLoadConfig   ReloadHandlerConfig
	ReloadFileName = "reload.toml"
)
