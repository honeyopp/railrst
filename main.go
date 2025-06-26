package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Hook struct {
	TargetURL string
	Logs      []Log
}

type Log struct {
	Timestamp  time.Time `json:"timestamp"`
	Body       string    `json:"body"`
	StatusCode int       `json:"status_code"`
}

var (
	hooks = make(map[string]*Hook)
	mu    sync.Mutex
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/hook/", hookHandler)
	http.HandleFunc("/logs/", logsHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := ":8080"
	fmt.Println("Listening on", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, nil)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	target := r.FormValue("target_url")
	if target == "" {
		http.Error(w, "请输入目标 URL", http.StatusBadRequest)
		return
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())

	mu.Lock()
	hooks[id] = &Hook{TargetURL: target}
	mu.Unlock()

	resp := fmt.Sprintf(`✅ Webhook 已创建！

📥 请求地址：/hook/%s  
📊 日志查看：/logs/%s`, id, id)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(resp))
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/hook/")
	mu.Lock()
	hook, exists := hooks[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "Webhook 不存在", http.StatusNotFound)
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	resp, err := http.Post(hook.TargetURL, "application/json", strings.NewReader(string(body)))
	status := 0
	if err != nil {
		status = 500
	} else {
		status = resp.StatusCode
	}

	logEntry := Log{
		Timestamp:  time.Now(),
		Body:       string(body),
		StatusCode: status,
	}

	mu.Lock()
	hook.Logs = append([]Log{logEntry}, hook.Logs...)
	if len(hook.Logs) > 10 {
		hook.Logs = hook.Logs[:10]
	}
	mu.Unlock()

	w.Write([]byte(fmt.Sprintf("转发完成，状态码：%d", status)))
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/logs/")
	mu.Lock()
	hook, exists := hooks[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "Webhook 不存在", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hook.Logs)
}