package watch

import (
	"errors"

	"github.com/codfrm/cago/pkg/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"go.uber.org/zap"
)

func (w *watch) init() (*dnspod.RecordListItem, error) {
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
		return nil, err
	}

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := dnspod.NewDescribeRecordListRequest()

	request.Domain = common.StringPtr(w.config.Domain)

	response, err := w.client.DescribeRecordList(request)
	if err != nil {
		logger.Default().Error("DescribeRecordList error", zap.Error(err))
		return nil, err
	}
	for _, v := range response.Response.RecordList {
		if *v.Name == w.config.Name && *v.Value == w.config.Value {
			logger.Default().Info("record found", zap.Any("record", v))
			return v, nil
		}
	}
	return nil, errors.New("record not found")

}
