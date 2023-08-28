package watch

import (
	"errors"

	"github.com/codfrm/cago/pkg/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

func (w *watch) queryRecord(domain, name, value string) (*dnspod.RecordListItem, error) {
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
			logger.Default().Info("record found", zap.Any("record", v))
			return v, nil
		}
	}
	return nil, errors.New("record not found")

}
