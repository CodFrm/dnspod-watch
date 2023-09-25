package watch

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/codfrm/cago/pkg/logger"
	"github.com/codfrm/dnspod-watch/pkg/pushcat"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

type record struct {
	w             *watch
	record        *dnspod.RecordListItem
	isDisable     bool
	domain, value string
	logger        *logger.CtxLogger
}

func newRecord(w *watch, r *dnspod.RecordListItem, domain, value string) *record {
	return &record{
		w:         w,
		record:    r,
		isDisable: false,
		domain:    domain, value: value,
		logger: logger.NewCtxLogger(logger.Default()).With(
			zap.String("domain", domain), zap.String("value", value),
		),
	}
}

// watch 每分钟检查ip是否可以访问, 无法访问自动暂停记录
func (r *record) watch(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	lastSwitch := time.Now()
	count := 0
	for {
		select {
		case <-t.C:
			// 检查ip是否可以访问
			count++
			if err := r.checkIP(ctx, r.value); err != nil {
				r.logger.Ctx(ctx).Error("check ip err", zap.Error(err))
				// 连续3次无法访问,暂停记录
				if !r.isDisable && count > 3 {
					count = 0
					// 暂停记录
					request := dnspod.NewModifyRecordStatusRequest()
					request.SetContext(ctx)
					request.Domain = common.StringPtr(r.domain)
					request.RecordId = common.Uint64Ptr(*r.record.RecordId)
					request.Status = common.StringPtr("DISABLE")
					_, err := r.w.client.ModifyRecordStatus(request)
					msg := fmt.Sprintf("域名: %s, 记录: %s, ip无法访问,暂停记录", r.domain, r.value)
					if err != nil {
						r.logger.Ctx(ctx).Error("modify record status err", zap.Error(err))
						msg += "\n记录修改失败: " + err.Error()
					} else {
						r.logger.Ctx(ctx).Info("modify record status success",
							zap.String("status", "DISABLE"))
						r.isDisable = true
					}
					if err := pushcat.Send(ctx, "ip无法访问,暂停记录", msg); err != nil {
						r.logger.Ctx(ctx).Error("发送通知错误",
							zap.Error(err),
							zap.String("msg", msg))
					}
				}
			} else if r.isDisable && count > 3 {
				// 上次切换时间超过30分钟才能再次切换
				if time.Since(lastSwitch) < time.Minute*30 {
					r.logger.Ctx(ctx).Info("ip可以访问,但是上次切换时间不足30分钟")
					continue
				}
				lastSwitch = time.Now()

				// 检查连续成功3次,开启记录
				count = 0
				request := dnspod.NewModifyRecordStatusRequest()
				request.SetContext(ctx)
				request.Domain = common.StringPtr(r.domain)
				request.RecordId = common.Uint64Ptr(*r.record.RecordId)
				request.Status = common.StringPtr("ENABLE")
				_, err := r.w.client.ModifyRecordStatus(request)
				msg := fmt.Sprintf("域名: %s, 记录: %s, ip可以访问,开启记录", r.domain, r.value)
				if err != nil {
					r.logger.Ctx(ctx).Error("modify record status err", zap.Error(err))
					msg += "\n记录修改失败: " + err.Error()
				} else {
					r.logger.Ctx(ctx).Info("modify record status success",
						zap.String("status", "ENABLE"))
					r.isDisable = false
				}
				if err := pushcat.Send(ctx, "ip可以访问,开启记录", msg); err != nil {
					r.logger.Ctx(ctx).Error("发送通知错误",
						zap.Error(err),
						zap.String("msg", msg))
				}
			} else {
				r.logger.Ctx(ctx).Info("ip is ok")
			}
		case <-ctx.Done():
			t.Stop()
		}
	}
}

func (r *record) checkIP(ctx context.Context, ip string) error {
	con, err := net.DialTimeout("tcp", ip+":80", time.Second*10)
	if err != nil {
		return err
	}
	return con.Close()
}
