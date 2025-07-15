package scanner

import (
	"github.com/valyala/fasthttp"
	"time"
	"strings"
	"sync"
	"regexp"
)

// ScanTask 表示一个待扫描的任务
type ScanTask struct {
	BaseURL   string // 目标基础URL
	Path      string // 字典路径
	FullURL   string // 完整URL
}

// ScanResult 表示一次扫描的结果
type ScanResult struct {
	FullURL      string
	StatusCode   int
	Duration     time.Duration
	Err          error
	ContentLength int // 新增字段，内容长度
	Title         string // 新增字段，标题
}

// Scanner 扫描器主结构体
type Scanner struct {
	BaseURLs     []string
	Dict         []string
	ThreadCount  int
	Timeout      time.Duration
	Headers      map[string]string
	ResultChan   chan ScanResult
	TaskChan     chan ScanTask
	StopChan     chan struct{}
}

// NewScanner 创建一个新的Scanner实例
func NewScanner(baseURLs []string, dict []string, threadCount int, timeout time.Duration, headers map[string]string) *Scanner {
	s := &Scanner{
		BaseURLs:    baseURLs,
		Dict:        dict,
		ThreadCount: threadCount,
		Timeout:     timeout,
		Headers:     headers,
		ResultChan:  make(chan ScanResult, 100),
		TaskChan:    make(chan ScanTask, 100),
		StopChan:    make(chan struct{}),
	}
	return s
}

// Start 启动扫描
func (s *Scanner) Start() {
	// 生成任务
	go s.generateTasks()
	// 启动worker
	for i := 0; i < s.ThreadCount; i++ {
		go s.worker()
	}
}

// generateTasks 生成所有扫描任务
func (s *Scanner) generateTasks() {
	var once sync.Once
	var robotsPaths []string
	var robotsLock sync.Mutex

	addRobotsPaths := func(base string) {
		robotsURL := base
		if !strings.HasSuffix(robotsURL, "/") {
			robotsURL += "/"
		}
		robotsURL += "robots.txt"
		client := &fasthttp.Client{}
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		req.SetRequestURI(robotsURL)
		req.Header.SetMethod("GET")
		err := client.DoTimeout(req, resp, 5*time.Second)
		if err == nil && resp.StatusCode() == 200 {
			lines := strings.Split(string(resp.Body()), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Disallow:") {
					path := strings.TrimSpace(strings.TrimPrefix(line, "Disallow:"))
					if path != "" && path != "/" {
						robotsLock.Lock()
						robotsPaths = append(robotsPaths, path)
						robotsLock.Unlock()
					}
				}
			}
		}
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}

	for _, base := range s.BaseURLs {
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}
		// 首次遇到目标，尝试加载robots.txt
		once.Do(func() { addRobotsPaths(base) })
		for _, path := range s.Dict {
			p := strings.TrimLeft(path, "/")
			fullURL := base + p
			task := ScanTask{
				BaseURL: base,
				Path:    path,
				FullURL: fullURL,
			}
			s.TaskChan <- task
		}
	}
	// 将robots.txt中发现的路径加入扫描
	robotsLock.Lock()
	for _, base := range s.BaseURLs {
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}
		for _, path := range robotsPaths {
			p := strings.TrimLeft(path, "/")
			fullURL := base + p
			task := ScanTask{
				BaseURL: base,
				Path:    path,
				FullURL: fullURL,
			}
			s.TaskChan <- task
		}
	}
	robotsLock.Unlock()
	close(s.TaskChan)
}

// worker 执行扫描任务
func (s *Scanner) worker() {
	client := &fasthttp.Client{}
	for task := range s.TaskChan {
		select {
		case <-s.StopChan:
			return
		default:
			start := time.Now()
			req := fasthttp.AcquireRequest()
			resp := fasthttp.AcquireResponse()
			req.SetRequestURI(task.FullURL)
			req.Header.SetMethod("GET")
			uaSet := false
			for k, v := range s.Headers {
				req.Header.Set(k, v)
				if strings.ToLower(k) == "user-agent" {
					uaSet = true
				}
			}
			if !uaSet {
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")
			}
			err := client.DoTimeout(req, resp, s.Timeout)
			duration := time.Since(start)
			statusCode := 0
			contentLength := 0
			title := ""
			var titleRegex = regexp.MustCompile(`(?i)<title.*?>(.*?)</title>`)

			if err == nil {
				statusCode = resp.StatusCode()
				contentLength = len(resp.Body())
				body := resp.Body()
				matches := titleRegex.FindStringSubmatch(string(body))
				
				if len(matches) > 2 {
					title = strings.TrimSpace(matches[1])
					title = strings.NewReplacer(
						"\n", " ",
						"\r", " ",
						"\t", " ",
					).Replace(title)
				}
				
			}
			s.ResultChan <- ScanResult{
				FullURL:    task.FullURL,
				StatusCode: statusCode,
				Duration:   duration,
				Err:        err,
				ContentLength: contentLength,
				Title: title,
			}
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}
	}
}

// Stop 停止扫描
func (s *Scanner) Stop() {
	close(s.StopChan)
} 
