package img4o

import (
	"errors"
	"strings"

	"github.com/daodao97/xgo/xrequest"
)

// Generate4oImageRequest 生成4o图像请求参数
type Generate4oImageRequest struct {
	Prompt   string   `json:"prompt" binding:"required"`
	FilesURL []string `json:"files_url" binding:"required,max=4"`
	Size     string   `json:"size" binding:"required,oneof=1:1 3:2 2:3"`
}

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Data    T      `json:"data"`
}

type Generate4oImageResponse struct {
	TaskID string `json:"taskId"`
}

var key = "dad2293b6473bb43e7607cb17597ea06"
var generateUrl = "https://kieai.erweima.ai/api/v1/gpt4o-image/generate"
var detailUrl = "https://kieai.erweima.ai/api/v1/gpt4o-image/record-info"

func Generate4oImage(req Generate4oImageRequest) (*Generate4oImageResponse, error) {
	resp, err := xrequest.New().
		SetBody(req).
		SetHeader("Authorization", "Bearer "+key).
		Post(generateUrl)
	if err != nil {
		return nil, err
	}

	if resp.Error() != nil {
		return nil, resp.Error()
	}

	var respData Response[Generate4oImageResponse]
	if err := resp.Scan(&respData); err != nil {
		return nil, err
	}

	if respData.Code != 200 {
		if strings.Contains(respData.Message, "credits are insufficient") {
			return nil, errors.New("系统资源不足, 请联系客服反馈")
		}
		return nil, errors.New(respData.Message)
	}

	return &respData.Data, nil
}

type GetImageDetailResponse struct {
	TaskID       string         `json:"taskId"`
	ParamJSON    string         `json:"paramJson"`
	Response     DetailResponse `json:"response"`
	SuccessFlag  int            `json:"successFlag"`
	Status       string         `json:"status"`
	ErrorCode    int            `json:"errorCode"`
	ErrorMessage string         `json:"errorMessage"`
	Progress     string         `json:"progress"`
}
type DetailResponse struct {
	ResultUrls []string `json:"resultUrls"`
}

func GetImageDetail(taskID string) (*GetImageDetailResponse, error) {
	resp, err := xrequest.New().
		SetQueryParam("taskId", taskID).
		SetHeader("Authorization", "Bearer "+key).
		Get(detailUrl)
	if err != nil {
		return nil, err
	}

	if resp.Error() != nil {
		return nil, resp.Error()
	}

	var respData Response[GetImageDetailResponse]
	if err := resp.Scan(&respData); err != nil {
		return nil, err
	}

	return &respData.Data, nil
}
