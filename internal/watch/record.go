package watch

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/codfrm/cago/pkg/logger"
	"github.com/codfrm/dnspod-watch/pkg/pushcat"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

type record struct {
	w           *watch
	record      *dnspod.RecordListItem
	loadBalance *dnspod.RecordListItem
	isDisable   bool
	domain      *CheckDomain
	logger      *logger.CtxLogger
}

func newRecord(w *watch, r *dnspod.RecordListItem, domain *CheckDomain) (*record, error) {
	ret := &record{
		w:         w,
		record:    r,
		isDisable: false,
		domain:    domain,
		logger: logger.NewCtxLogger(logger.Default()).With(
			zap.String("domain", domain.Domain), zap.String("value", domain.Value),
		),
	}
	if domain.LoadBalance != nil {
		var err error
		ret.loadBalance, err = w.queryRecord(domain.Domain, domain.Name,
			domain.LoadBalance.Value, domain.LoadBalance.Line)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// watch 每分钟检查ip是否可以访问, 无法访问自动暂停记录
func (r *record) watch(ctx context.Context) {
	t := time.NewTicker(time.Second)
	s := newRetry()
	loadBalance := newRetry()
	for {
		select {
		case <-t.C:
			duration, err := r.checkIP(ctx, r.domain.Value)
			if err != nil {
				r.logger.Ctx(ctx).Error("check ip err", zap.Error(err))
			} else {
				r.logger.Ctx(ctx).Info("check ip ok", zap.Duration("duration", duration))
			}
			_ = s.check(err == nil, func() error {
				// 增加负载均衡
				if r.domain.LoadBalance != nil {
					// 判断延迟是否超过200毫秒
					loadBalance.check(duration > time.Millisecond*100, func() error {
						// 开启记录
						msg := fmt.Sprintf("开启负载均衡 域名: %s, 记录: %s",
							r.domain.Domain, r.domain.LoadBalance.Value)
						enableErr := r.w.enable(ctx, r.domain.Domain, *r.loadBalance.RecordId)
						if enableErr != nil {
							r.logger.Ctx(ctx).Error("modify record status err", zap.Error(enableErr))
							msg += "\n记录修改失败: " + enableErr.Error()
						} else {
							r.logger.Ctx(ctx).Info("modify record status success",
								zap.String("status", "ENABLE"))
							r.isDisable = false
						}
						if pushErr := pushcat.Send(ctx, "开启负载均衡", msg); pushErr != nil {
							r.logger.Ctx(ctx).Error("发送通知错误",
								zap.Error(pushErr),
								zap.String("msg", msg))
						}
						return enableErr
					}, func() error {
						// 开启记录
						msg := fmt.Sprintf("关闭负载均衡 域名: %s, 记录: %s",
							r.domain.Domain, r.domain.LoadBalance.Value)
						disableErr := r.w.disable(ctx, r.domain.Domain, *r.loadBalance.RecordId)
						if disableErr != nil {
							r.logger.Ctx(ctx).Error("modify record status err", zap.Error(disableErr))
							msg += "\n记录修改失败: " + disableErr.Error()
						} else {
							r.logger.Ctx(ctx).Info("modify record status success",
								zap.String("status", "DISABLE"))
							r.isDisable = false
						}
						if pushErr := pushcat.Send(ctx, "关闭负载均衡", msg); pushErr != nil {
							r.logger.Ctx(ctx).Error("发送通知错误",
								zap.Error(pushErr),
								zap.String("msg", msg))
						}
						return disableErr
					})
				}
				// 开启记录
				msg := fmt.Sprintf("域名: %s, 记录: %s, ip可以访问,开启记录", r.domain.Domain, r.domain.Value)
				enableErr := r.w.enable(ctx, r.domain.Domain, *r.record.RecordId)
				if enableErr != nil {
					r.logger.Ctx(ctx).Error("modify record status err", zap.Error(enableErr))
					msg += "\n记录修改失败: " + enableErr.Error()
				} else {
					r.logger.Ctx(ctx).Info("modify record status success",
						zap.String("status", "ENABLE"))
					r.isDisable = false
				}
				if pushErr := pushcat.Send(ctx, "ip可以访问,开启记录", msg); pushErr != nil {
					r.logger.Ctx(ctx).Error("发送通知错误",
						zap.Error(pushErr),
						zap.String("msg", msg))
				}
				return enableErr
			}, func() error {
				// 暂停记录
				msg := fmt.Sprintf("域名: %s, 记录: %s, ip无法访问,暂停记录", r.domain.Domain, r.domain.Value)
				disableErr := r.w.disable(ctx, r.domain.Domain, *r.record.RecordId)
				if disableErr != nil {
					r.logger.Ctx(ctx).Error("modify record status err", zap.Error(disableErr))
					msg += "\n记录修改失败: " + disableErr.Error()
				} else {
					r.logger.Ctx(ctx).Info("modify record status success",
						zap.String("status", "DISABLE"))
					r.isDisable = true
				}
				if pushErr := pushcat.Send(ctx, "ip无法访问,暂停记录", msg); pushErr != nil {
					r.logger.Ctx(ctx).Error("发送通知错误",
						zap.Error(pushErr),
						zap.String("msg", msg))
				}
				return disableErr
			})
		case <-ctx.Done():
			t.Stop()
		}
	}
}

func (r *record) checkIP(ctx context.Context, ip string) (time.Duration, error) {
	ts := time.Now()
	con, err := net.DialTimeout("tcp", ip+":80", time.Second*10)
	if err != nil {
		return 0, err
	}
	return time.Since(ts), con.Close()
}
