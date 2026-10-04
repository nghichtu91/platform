package email

var (
	EMailCfg EMailConfig
)

type EMailConfig struct {
	EMailJPCfg EMailZoneConfig `toml:"EMailJPConfig"` // 日本邮箱
}

type EMailZoneConfig struct {
	ServerHost string `toml:"serverHost"`
	ServerPort int    `toml:"serverPort"`
	FromEmail  string `toml:"fromEmail"`
	FromPasswd string `toml:"fromPasswd"`
}

func GetEMailConfig() EMailZoneConfig {
	return EMailCfg.EMailJPCfg
}
