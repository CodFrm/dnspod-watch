package watch

import (
	"context"

	"github.com/codfrm/cago"
	"github.com/codfrm/cago/configs"
	"github.com/codfrm/cago/pkg/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

type Config struct {
	SecretID  string   `yaml:"secretID"`  // 账号 secret id
	SecretKey string   `yaml:"secretKey"` // 账号 secret key
	Domain    string   // 域名
	Name      string   // 记录名
	Value     []string // 记录值
}

type watch struct {
	config     *Config
	credential *common.Credential
	client     *dnspod.Client
	isDisable  bool
}

func Watch() cago.Component {
	return &watch{
		isDisable: false,
	}
}

func (w *watch) Start(ctx context.Context, cfg *configs.Config) error {
	config := &Config{}
	if err := cfg.Scan("w", config); err != nil {
		return err
	}
	// 获取记录列表
	w.config = config
	w.credential = common.NewCredential(
		w.config.SecretID,
		w.config.SecretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	var err error
	w.client, err = dnspod.NewClient(w.credential, "", cpf)
	if err != nil {
		logger.Default().Error("NewClient error", zap.Error(err))
		return err
	}
	for _, v := range w.config.Value {
		r, err := w.queryRecord(v)
		if err != nil {
			return err
		}
		record := newRecord(w, r, config.Domain, v)
		go record.watch(ctx)
	}
	return nil
}

func (w *watch) CloseHandle() {

}
