package watch

import (
	"context"
	"errors"
	"github.com/codfrm/cago/pkg/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

func (w *watch) queryRecord(domain, name, value, line string) (*dnspod.RecordListItem, error) {
	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := dnspod.NewDescribeRecordListRequest()

	request.Domain = common.StringPtr(domain)

	response, err := w.client.DescribeRecordList(request)
	if err != nil {
		logger.Default().Error("DescribeRecordList error", zap.Error(err))
		return nil, err
	}
	for _, v := range response.Response.RecordList {
		if *v.Name == name && *v.Value == value {
			if line != "" && *v.Line != line {
				continue
			}
			logger.Default().Info("record found", zap.Any("record", v))
			return v, nil
		}
	}
	return nil, errors.New("record not found")
}

func (w *watch) enable(ctx context.Context, domain string, recordId uint64) error {
	// 开启记录
	request := dnspod.NewModifyRecordStatusRequest()
	request.SetContext(ctx)
	request.Domain = common.StringPtr(domain)
	request.RecordId = common.Uint64Ptr(recordId)
	request.Status = common.StringPtr("ENABLE")
	_, err := w.client.ModifyRecordStatus(request)
	if err != nil {
		return err
	}
	return nil
}

func (w *watch) disable(ctx context.Context, domain string, recordId uint64) error {
	// 开启记录
	request := dnspod.NewModifyRecordStatusRequest()
	request.SetContext(ctx)
	request.Domain = common.StringPtr(domain)
	request.RecordId = common.Uint64Ptr(recordId)
	request.Status = common.StringPtr("DISABLE")
	_, err := w.client.ModifyRecordStatus(request)
	if err != nil {
		return err
	}
	return nil
}
