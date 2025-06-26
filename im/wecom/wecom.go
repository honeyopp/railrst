package wecom

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	"webhook-proxy/im"
	"webhook-proxy/utils"
)

// WeComClient 实现 im.IMClient
type WeComClient struct {
	CorpID     string
	CorpSecret string

	token    string
	tokenExp time.Time
}

// NewWeComClient 构造函数
func NewWeComClient(corpID, corpSecret string) *WeComClient {
	return &WeComClient{
		CorpID:     corpID,
		CorpSecret: corpSecret,
	}
}

func (w *WeComClient) getAccessToken() (string, error) {
	if w.token != "" && time.Now().Before(w.tokenExp) {
		return w.token, nil
	}
	// 请求获取access_token，示例简单写法，真实项目建议加重试和错误处理
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", w.CorpID, w.CorpSecret)
	var res struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	err := utils.HttpGetJSON(url, &res)
	if err != nil {
		return "", err
	}
	if res.ErrCode != 0 {
		return "", errors.New(res.ErrMsg)
	}
	w.token = res.AccessToken
	w.tokenExp = time.Now().Add(time.Duration(res.ExpiresIn-60) * time.Second)
	return w.token, nil
}

// 发送消息实现
func (w *WeComClient) SendMessage(toUserIDs []string, toDeptIDs []string, msg im.Message) error {
	token, err := w.getAccessToken()
	if err != nil {
		return err
	}
	payload, err := w.convertMessage(msg)
	if err != nil {
		return err
	}

	// 构造请求body
	body := map[string]interface{}{
		"touser":  utils.JoinIDs(toUserIDs),
		"toparty": utils.JoinIDs(toDeptIDs),
		"msgtype": string(msg.Type),
	}
	for k, v := range payload {
		body[k] = v
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	var res struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	err = utils.HttpPostJSON(url, body, &res, make(map[string]string))
	if err != nil {
		return err
	}
	if res.ErrCode != 0 {
		return errors.New(res.ErrMsg)
	}
	return nil
}

// 获取所有部门
func (w *WeComClient) GetDepartments() ([]im.Department, error) {
	token, err := w.getAccessToken()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/department/list?access_token=%s", token)
	var res struct {
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
		Department []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			ParentID int    `json:"parentid"`
		} `json:"department"`
	}
	err = utils.HttpGetJSON(url, &res)
	if err != nil {
		return nil, err
	}
	if res.ErrCode != 0 {
		return nil, errors.New(res.ErrMsg)
	}

	var depts []im.Department

	for _, d := range res.Department {
		atoi, _ := strconv.Atoi(fmt.Sprintf("%d", d.ID))
		depts = append(depts, im.Department{
			ID:   atoi,
			Name: d.Name,
		})
	}
	return depts, nil
}

// 获取所有用户，递归或分页这里简化，只取根部门下全部用户
func (w *WeComClient) GetUsers() ([]im.User, error) {
	token, err := w.getAccessToken()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/user/list?access_token=%s&department_id=1&fetch_child=1", token)
	var res struct {
		ErrCode  int    `json:"errcode"`
		ErrMsg   string `json:"errmsg"`
		UserList []struct {
			UserID     string `json:"userid"`
			Name       string `json:"name"`
			Department []int  `json:"department"`
		} `json:"userlist"`
	}
	err = utils.HttpGetJSON(url, &res)
	if err != nil {
		return nil, err
	}
	if res.ErrCode != 0 {
		return nil, errors.New(res.ErrMsg)
	}
	var users []im.User
	for _, u := range res.UserList {
		var depts []string
		for _, d := range u.Department {
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

// convertMessage 将统一消息转成企业微信格式payload
func (w *WeComClient) convertMessage(msg im.Message) (map[string]interface{}, error) {
	switch msg.Type {
	case im.TextMsg:
		text, ok := msg.Content.(string)
		if !ok {
			return nil, errors.New("invalid content for text message")
		}
		return map[string]interface{}{
			"text": map[string]string{"content": text},
		}, nil
	case im.ImageMsg:
		img, ok := msg.Content.(im.ImageContent)
		if !ok {
			return nil, errors.New("invalid content for image message")
		}
		return map[string]interface{}{
			"image": map[string]string{"media_id": img.MediaID},
		}, nil
	case im.VoiceMsg:
		voice, ok := msg.Content.(im.VoiceContent)
		if !ok {
			return nil, errors.New("invalid content for voice message")
		}
		return map[string]interface{}{
			"voice": map[string]string{"media_id": voice.MediaID},
		}, nil
	case im.VideoMsg:
		video, ok := msg.Content.(im.VideoContent)
		if !ok {
			return nil, errors.New("invalid content for video message")
		}
		return map[string]interface{}{
			"video": map[string]string{
				"media_id":    video.MediaID,
				"title":       video.Title,
				"description": video.Description,
			},
		}, nil
	case im.FileMsg:
		file, ok := msg.Content.(im.FileContent)
		if !ok {
			return nil, errors.New("invalid content for file message")
		}
		return map[string]interface{}{
			"file": map[string]string{"media_id": file.MediaID},
		}, nil
	case im.TextCardMsg:
		card, ok := msg.Content.(im.TextCardContent)
		if !ok {
			return nil, errors.New("invalid content for textcard message")
		}
		return map[string]interface{}{
			"textcard": map[string]string{
				"title":       card.Title,
				"description": card.Description,
				"url":         card.URL,
				"btntxt":      card.ButtonText,
			},
		}, nil
	case im.NewsMsg:
		news, ok := msg.Content.(im.NewsContent)
		if !ok {
			return nil, errors.New("invalid content for news message")
		}
		var articles []map[string]string
		for _, art := range news.Articles {
			articles = append(articles, map[string]string{
				"title":       art.Title,
				"description": art.Description,
				"url":         art.URL,
				"picurl":      art.PicURL,
			})
		}
		return map[string]interface{}{
			"news": map[string]interface{}{
				"articles": articles,
			},
		}, nil
	case im.MarkdownMsg:
		md, ok := msg.Content.(im.MarkdownContent)
		if !ok {
			return nil, errors.New("invalid content for markdown message")
		}
		return map[string]interface{}{
			"markdown": map[string]string{"content": md.Content},
		}, nil
	default:
		return nil, errors.New("unsupported message type")
	}
}
