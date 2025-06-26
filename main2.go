package main

import (
	"fmt"
	"webhook-proxy/im"
	"webhook-proxy/im/wecom"
)

func main() {
	client := wecom.NewWeComClient("your_corp_id", "your_corp_secret")

	// 发送文本消息
	err := client.SendMessage([]string{"userid1", "userid2"}, nil, im.Message{
		Type:    im.TextMsg,
		Content: "Hello from Go SDK!",
	})
	if err != nil {
		fmt.Println("SendMessage error:", err)
	} else {
		fmt.Println("SendMessage success")
	}

	// 获取部门
	depts, err := client.GetDepartments()
	if err != nil {
		fmt.Println("GetDepartments error:", err)
	} else {
		fmt.Printf("Departments: %+v\n", depts)
	}

	// 获取用户
	users, err := client.GetUsers()
	if err != nil {
		fmt.Println("GetUsers error:", err)
	} else {
		fmt.Printf("Users: %+v\n", users)
	}

}
