package config

/*
Gift服务起服所需配置。
*/
type Config struct {
	GiftCfg GiftConfig
}

type GiftConfig struct {
	// DB 数据库相关的配置。
	DBDriver       string `toml:"db_driver"`             // 使用的数据库驱动。
	DBUrl          string `toml:"gift_mysql_url"`        // DB连接完整URL。
	DBMaxOpenConns int    `toml:"db_max_open_conns"`     // 与DB间的最大允许连接数。
	DBMaxIdleConns int    `toml:"db_max_idle_conns"`     // 与DB间的最大允许闲置数。
	DBMaxLifeTime  int    `toml:"db_conn_max_life_time"` // 与DB间的连接最大允许保持时间。

	// Code 服务器参数相关配置。
	CodeProjectNum int    `toml:"code_project_num"` // 当前项目的编号（用于兑换码生成）。
	ServerID       string `toml:"serverid"`         // 当前服务器编号。
	ProjID         string `toml:"proj"`             // 当前项目的编号。

	// OSS 数据库相关配置。
	OSSEndPoint   string `toml:"oss_endpoint"`    // OSS外网访问地域节点。
	OSSDataBucket string `toml:"oss_data_bucket"` // 存储数据的远端桶。
	OSSCloudRoot  string `toml:"oss_cloud_root"`  // 存储数据的远端桶内的根目录
	OSSAccessKey  string `toml:"oss_accesskey"`   // OSS链接Key。
	OSSSecretKey  string `toml:"oss_secretkey"`   // OSS秘钥Key。

	// Gin 对外服务参数。
	GinPort        string `toml:"httpport"`        // 服务器Gin服务监听端口。
	ServerCertFile string `toml:"server_crt_file"` // 服务器证书文件。
	ServerKeyFile  string `toml:"server_key_file"` // 服务器秘钥文件。
	CACertFile     string `toml:"ca_crt_file"`     // CA证书文件。
}
