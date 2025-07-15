darwin-amd64：mac+intel cpu版本
darwin-arm64：mac +m系列cpu版本
linux-amd64 ：Linux版本
windows-amd64.exe ：Windows版本

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
