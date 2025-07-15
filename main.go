package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"webscan/scanner"
)

// ANSI颜色代码
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorPurple = "\033[35m"
	ColorOrange = "\033[38;5;208m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	SaveCursor    = "\033[s"
	RestoreCursor = "\033[u"
	ClearLine     = "\033[K"
	MoveUp        = "\033[1A"
)

func printHelp() {
	help := `
高速网站目录和后台扫描器

用法:
  webscan [参数]

参数:
  -h, --help           显示帮助信息
  -u, --url URL        目标网站URL（如：https://example.com）
  -U, --urlfile FILE   目标URL文件，每行一个
  -t, --threads N      扫描线程数（默认100）
  -to, --timeout MS    超时时间（毫秒，默认10000）
  -d, --dict FILE      字典文件路径
  -H, --header KEY:VAL  自定义请求头，可多次指定，如-H 'User-Agent: xxx'

示例:
  webscan -u https://example.com -U urls.txt -t 200 -to 5000 -d dict.txt -H 'User-Agent: xxx'
`
	fmt.Println(help)
}

func loadDict(dictPath string) ([]string, error) {
	file, err := os.Open(dictPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dict []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			dict = append(dict, line)
		}
	}
	return dict, scanner.Err()
}

func colorForStatus(status int) string {
	switch status {
	case 200:
		return ColorGreen
	case 302:
		return ColorPurple
	case 403:
		return ColorOrange
	case 404:
		return ColorRed
	case 500:
		return ColorYellow
	default:
		return ColorReset
	}
}

func printProgress(current, total int) {
	percent := float64(current) / float64(total) * 100
	barLen := 40
	filled := int(float64(barLen) * percent / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("-", barLen-filled)
	fmt.Printf("%s\033[999;1H%s进度: [%s] %.1f%% (%d/%d)%s\r%s",
		SaveCursor,
		ClearLine,
		bar,
		percent,
		current,
		total,
		ClearLine,
		RestoreCursor)
	if current == total {
		fmt.Println()
	}
}

func parseHeaders(headerArgs []string) map[string]string {
	headers := make(map[string]string)
	for _, h := range headerArgs {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			headers[key] = val
		}
	}
	return headers
}

func loadTargets(urlArg string, urlFile string) ([]string, error) {
	targets := make(map[string]struct{})
	if urlArg != "" {
		targets[urlArg] = struct{}{}
	}
	if urlFile != "" {
		file, err := os.Open(urlFile)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				targets[line] = struct{}{}
			}
		}
		file.Close()
	}
	var result []string
	for k := range targets {
		result = append(result, k)
	}
	return result, nil
}

func formatDuration(d time.Duration) string {
	sec := float64(d.Microseconds()) / 1e6
	return fmt.Sprintf("%.2fs", sec)
}

func main() {
	url := flag.String("u", "", "目标网站URL")
	urlFile := flag.String("U", "", "目标URL文件，每行一个")
	threads := flag.Int("t", 100, "扫描线程数")
	timeout := flag.Int("to", 10000, "超时时间(毫秒)")
	dictPath := flag.String("d", "", "字典文件路径")
	var headerList []string
	flag.Var((*stringSlice)(&headerList), "H", "自定义请求头，可多次指定，如-H 'User-Agent: xxx'")
	flag.Parse()

	if len(os.Args) == 1 {
		printHelp()
		return
	}
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printHelp()
			return
		}
	}

	targets, err := loadTargets(*url, *urlFile)
	if err != nil || len(targets) == 0 {
		fmt.Println("[!] 目标URL不能为空，且文件需存在！")
		printHelp()
		return
	}

	dict, err := loadDict(*dictPath)
	if err != nil {
		fmt.Printf("[!] 加载字典失败: %v\n", err)
		return
	}

	headers := parseHeaders(headerList)

	s := scanner.NewScanner(targets, dict, *threads, time.Duration(*timeout)*time.Millisecond, headers)
	go s.Start()

	fmt.Println()

	total := len(dict) * len(targets)
	for i := 0; i < total; i++ {
		result := <-s.ResultChan
		if result.StatusCode == 404 {
			printProgress(i+1, total)
			continue
		}
		color := colorForStatus(result.StatusCode)
		durStr := formatDuration(result.Duration)
		if result.Err != nil {
			fmt.Printf("%sURL: %s | 状态码: %d | 耗时: %s | 错误: %v%s\n", color, result.FullURL, result.StatusCode, durStr, result.Err, ColorReset)
		} else {
			fmt.Printf("%sURL: %s | 状态码: %d | 耗时: %s | 大小: %d 字节 | 标题: %s%s\n", color, result.FullURL, result.StatusCode, durStr, result.ContentLength, result.Title, ColorReset)
		}
		printProgress(i+1, total)
	}

	s.Stop()
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
} 