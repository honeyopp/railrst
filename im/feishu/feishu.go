package feishu

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
	"webhook-proxy/im"
)

type FeishuClient struct {
	AppID     string
	AppSecret string

	token    string
	tokenExp time.Time
}

func NewFeishuClient(appID, appSecret string) *FeishuClient {
	return &FeishuClient{
		AppID:     appID,
		AppSecret: appSecret,
	}
}

func (f *FeishuClient) getAccessToken() (string, error) {
	if f.token != "" && time.Now().Before(f.tokenExp) {
		return f.token, nil
	}

	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal/"
	payload := map[string]string{
		"app_id":     f.AppID,
		"app_secret": f.AppSecret,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}
	if res.Code != 0 {
		return "", errors.New(res.Msg)
	}
	f.token = res.TenantAccessToken
	f.tokenExp = time.Now().Add(time.Duration(res.Expire-60) * time.Second)
	return f.token, nil
}

func (f *FeishuClient) SendMessage(toUserIDs []string, toDeptIDs []string, msg im.Message) error {
	// 飞书暂不支持直接按部门发消息，暂忽略toDeptIDs
	token, err := f.getAccessToken()
	if err != nil {
		return err
	}
	var userID string
	if len(toUserIDs) == 0 {
		return errors.New("toUserIDs cannot be empty")
	}
	// 飞书要求单用户发送
	userID = toUserIDs[0]

	body := map[string]interface{}{
		"user_id":  userID,
		"msg_type": string(msg.Type),
	}
	content, err := f.convertMessage(msg)
	if err != nil {
		return err
	}
	body["content"] = content

	url := "https://open.feishu.cn/open-apis/message/v4/send/"
	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return err
	}
	if res.Code != 0 {
		return errors.New(res.Msg)
	}
	return nil
}

func (f *FeishuClient) GetDepartments() ([]im.Department, error) {
	token, err := f.getAccessToken()
	if err != nil {
		return nil, err
	}
	// 飞书获取部门列表接口，分页简化，获取全部
	url := "https://open.feishu.cn/open-apis/contact/v3/departments"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				DepartmentID string `json:"department_id"`
				Name         string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.Code != 0 {
		return nil, errors.New(res.Msg)
	}
	var depts []im.Department
	for _, d := range res.Data.Items {
		atoi, _ := strconv.Atoi(d.DepartmentID)
		depts = append(depts, im.Department{
			ID:   atoi,
			Name: d.Name,
		})
	}
	return depts, nil
}

func (f *FeishuClient) GetUsers() ([]im.User, error) {
	token, err := f.getAccessToken()
	if err != nil {
		return nil, err
	}
	// 飞书获取用户接口，分页简化，获取全部
	url := "https://open.feishu.cn/open-apis/contact/v3/users?page_size=100"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				UserID        string   `json:"user_id"`
				Name          string   `json:"name"`
				DepartmentIDs []string `json:"department_ids"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.Code != 0 {
		return nil, errors.New(res.Msg)
	}
	var users []im.User
	for _, u := range res.Data.Items {
		users = append(users, im.User{
			ID:      u.UserID,
			Name:    u.Name,
			DeptIDs: u.DepartmentIDs,
		})
	}
	return users, nil
}

func (f *FeishuClient) convertMessage(msg im.Message) (map[string]interface{}, error) {
	switch msg.Type {
	case im.TextMsg:
		text, ok := msg.Content.(string)
		if !ok {
			return nil, errors.New("invalid content for text message")
		}
		return map[string]interface{}{
			"text": text,
		}, nil
	case im.ImageMsg:
		img, ok := msg.Content.(im.ImageContent)
		if !ok {
			return nil, errors.New("invalid content for image message")
		}
		return map[string]interface{}{
			"image_key": img.MediaID,
		}, nil
	// 其他类型可继续扩展
	default:
		return nil, errors.New("unsupported message type for Feishu")
	}
}
