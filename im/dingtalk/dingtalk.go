package dingtalk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"webhook-proxy/im"
)

type DingTalkClient struct {
	AppKey    string
	AppSecret string

	token    string
	tokenExp time.Time
}

func NewDingTalkClient(appKey, appSecret string) *DingTalkClient {
	return &DingTalkClient{
		AppKey:    appKey,
		AppSecret: appSecret,
	}
}

func (d *DingTalkClient) getAccessToken() (string, error) {
	if d.token != "" && time.Now().Before(d.tokenExp) {
		return d.token, nil
	}
	url := fmt.Sprintf("https://oapi.dingtalk.com/gettoken?appkey=%s&appsecret=%s", d.AppKey, d.AppSecret)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var res struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}
	if res.ErrCode != 0 {
		return "", errors.New(res.ErrMsg)
	}
	d.token = res.AccessToken
	d.tokenExp = time.Now().Add(time.Duration(res.ExpiresIn-60) * time.Second)
	return d.token, nil
}

func (d *DingTalkClient) SendMessage(toUserIDs []string, toDeptIDs []string, msg im.Message) error {
	token, err := d.getAccessToken()
	if err != nil {
		return err
	}
	if len(toUserIDs) == 0 {
		return errors.New("toUserIDs cannot be empty")
	}
	// 钉钉支持多个userid逗号分隔
	toUserIDStr := strings.Join(toUserIDs, ",")

	body := map[string]interface{}{
		"agent_id":    123456789, // 这里需要你自己传入或设置
		"userid_list": toUserIDStr,
		"msg":         nil,
	}
	msgPayload, err := d.convertMessage(msg)
	if err != nil {
		return err
	}
	body["msg"] = msgPayload

	url := fmt.Sprintf("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token=%s", token)
	b, _ := json.Marshal(body)

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return err
	}
	if res.ErrCode != 0 {
		return errors.New(res.ErrMsg)
	}
	return nil
}

func (d *DingTalkClient) GetDepartments() ([]im.Department, error) {
	token, err := d.getAccessToken()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://oapi.dingtalk.com/department/list?access_token=%s", token)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var res struct {
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
		Department []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			ParentID int    `json:"parentid"`
		} `json:"department"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.ErrCode != 0 {
		return nil, errors.New(res.ErrMsg)
	}
	var depts []im.Department
	for _, dd := range res.Department {
		depts = append(depts, im.Department{
			ID:   dd.ID,
			Name: dd.Name,
		})
	}
	return depts, nil
}

func (d *DingTalkClient) GetUsers() ([]im.User, error) {
	token, err := d.getAccessToken()
	if err != nil {
		return nil, err
	}
	// 钉钉获取部门用户列表接口，默认取根部门ID=1
	url := fmt.Sprintf("https://oapi.dingtalk.com/user/listbypage?access_token=%s&department_id=1&offset=0&size=100", token)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var res struct {
		ErrCode  int    `json:"errcode"`
		ErrMsg   string `json:"errmsg"`
		Userlist []struct {
			UserID     string `json:"userid"`
			Name       string `json:"name"`
			DeptIDList []int  `json:"department"`
		} `json:"userlist"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.ErrCode != 0 {
		return nil, errors.New(res.ErrMsg)
	}

	var users []im.User
	for _, u := range res.Userlist {
		var depts []string
		for _, d := range u.DeptIDList {
			depts = append(depts, fmt.Sprintf("%d", d))
		}
		users = append(users, im.User{
			ID:      u.UserID,
			Name:    u.Name,
			DeptIDs: depts,
		})
	}
	return users, nil
}

func (d *DingTalkClient) convertMessage(msg im.Message) (map[string]interface{}, error) {
	switch msg.Type {
	case im.TextMsg:
		text, ok := msg.Content.(string)
		if !ok {
			return nil, errors.New("invalid content for text message")
		}
		return map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": text,
			},
		}, nil
	case im.ImageMsg:
		img, ok := msg.Content.(im.ImageContent)
		if !ok {
			return nil, errors.New("invalid content for image message")
		}
		// 钉钉图片需要media_id
		return map[string]interface{}{
			"msgtype": "image",
			"image": map[string]string{
				"media_id": img.MediaID,
			},
		}, nil
	// 其他消息类型可以继续拓展
	default:
		return nil, errors.New("unsupported message type for DingTalk")
	}
}
