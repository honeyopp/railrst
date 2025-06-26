package im

type Department struct {
	ID   int
	Name string
	//其他字段
}
type User struct {
	ID      string
	Name    string
	Phone   string
	DeptIDs []string
	// 其他字段
}

type MessageType string

const (
	TextMsg     MessageType = "text"
	ImageMsg    MessageType = "image"
	VoiceMsg    MessageType = "voice"
	VideoMsg    MessageType = "video"
	FileMsg     MessageType = "file"
	TextCardMsg MessageType = "textcard"
	NewsMsg     MessageType = "news" // 图文
	MarkdownMsg MessageType = "markdown"
)

// Message 统一消息结构
type Message struct {
	Type    MessageType
	Content interface{}
}

// Client 统一接口
type Client interface {
	SendMessage(toUserIDs []string, toDeptIDs []string, msg Message) error
	GetDepartments() ([]Department, error)
	GetUsers() ([]User, error)
}

// Article 图文消息结构
type Article struct {
	Title       string
	Description string
	Url         string
	PicUrl      string
}
