package ssk

type Env struct {
	KMSKeyID   string `env:"SAKURACLOUD_KMS_KEY_ID" required:""`
	ServerOnly bool   `env:"SSK_SERVER_ONLY" default:"false"`
	ServerAddr string `env:"SSK_SERVER_ADDR" default:"127.0.0.1:8200"`
	SOPSPath   string `env:"SSK_SOPS_PATH" default:"sops"`
}
