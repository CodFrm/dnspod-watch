package watch

import (
	"context"
	"net"
	"time"

	"github.com/codfrm/cago"
	"github.com/codfrm/cago/configs"
	"github.com/codfrm/cago/pkg/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

type Config struct {
	SecretID  string `yaml:"secretID"`  // 账号 secret id
	SecretKey string `yaml:"secretKey"` // 账号 secret key
	Domain    string // 域名
	Name      string // 记录名
	Value     string // 记录值
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
	if err := cfg.Scan("watch", config); err != nil {
		return err
	}
	// 获取记录列表
	w.config = config
	record, err := w.init()
	if err != nil {
		return err
	}
	go w.watch(ctx, record)
	return nil
}

func (w *watch) CloseHandle() {

}

// watch 每分钟检查ip是否可以访问, 无法访问自动暂停记录
func (w *watch) watch(ctx context.Context, record *dnspod.RecordListItem) {
	t := time.NewTicker(time.Minute)
	count := 0
	for {
		select {
		case <-t.C:
			// 检查ip是否可以访问
			count++
			if err := w.checkIP(ctx, w.config.Value); err != nil {
				// 连续3次无法访问,暂停记录
				if !w.isDisable && count > 3 {
					count = 0
					// 暂停记录
					request := dnspod.NewModifyRecordStatusRequest()
					request.SetContext(ctx)
					request.Domain = common.StringPtr(w.config.Domain)
					request.RecordId = common.Uint64Ptr(*record.RecordId)
					request.Status = common.StringPtr("DISABLE")
					_, err := w.client.ModifyRecordStatus(request)
					if err != nil {
						logger.Ctx(ctx).Error("modify record status err", zap.Error(err))
					} else {
						logger.Ctx(ctx).Info("modify record status success",
							zap.String("status", "DISABLE"))
						w.isDisable = true
					}
				}
			} else if w.isDisable && count > 3 {
				// 检查连续成功3次,开启记录
				count = 0
				request := dnspod.NewModifyRecordStatusRequest()
				request.SetContext(ctx)
				request.Domain = common.StringPtr(w.config.Domain)
				request.RecordId = common.Uint64Ptr(*record.RecordId)
				request.Status = common.StringPtr("ENABLE")
				_, err := w.client.ModifyRecordStatus(request)
				if err != nil {
					logger.Ctx(ctx).Error("modify record status err", zap.Error(err))
				} else {
					logger.Ctx(ctx).Info("modify record status success",
						zap.String("status", "ENABLE"))
					w.isDisable = false
				}
			} else {
				logger.Ctx(ctx).Info("ip is ok")
			}
		case <-ctx.Done():
			t.Stop()
		}
	}
}

func (w *watch) checkIP(ctx context.Context, ip string) error {
	con, err := net.DialTimeout("tcp", ip+":80", time.Second*10)
	if err != nil {
		return err
	}
	return con.Close()
}
