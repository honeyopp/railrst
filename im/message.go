package im

// 图片消息结构
type ImageContent struct {
	MediaID string // 媒体文件ID
}

// 语音消息结构
type VoiceContent struct {
	MediaID  string
	Duration int // 时长，秒
}

// 视频消息结构
type VideoContent struct {
	MediaID     string
	Title       string
	Description string
}

// 文件消息结构
type FileContent struct {
	MediaID string
}

// 文本卡片结构
type TextCardContent struct {
	Title       string
	Description string
	URL         string
	ButtonText  string
}

// 图文消息结构（新闻）
type NewsArticle struct {
	Title       string
	Description string
	URL         string
	PicURL      string
}

type NewsContent struct {
	Articles []NewsArticle
}

// Markdown消息结构
type MarkdownContent struct {
	Content string
}
