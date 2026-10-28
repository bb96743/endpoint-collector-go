// softscan 采集软件资产信息并计算文件 CRC 指纹。
//
// 两个子命令：
//
//	softscan -list            已安装软件清单（Windows 读注册表 Uninstall 键；
//	                          Linux 解析 dpkg/rpm 包库）
//	softscan -crc <目录>       对目录内全部文件流式计算 CRC-32（常量内存），
//	                          errgroup 控制并发；锁定/无权限文件跳过不中断
//	softscan -crc <目录> -json JSON 输出（对应采集策略上报服务端的格式）
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type FileCRC struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	CRC32 string `json:"crc32"`
}

type ScanResult struct {
	Dir       string    `json:"dir"`
	Files     int       `json:"files"`
	Skipped   int       `json:"skipped"`
	Bytes     int64     `json:"bytes"`
	ElapsedMS int64     `json:"elapsed_ms"`
	Items     []FileCRC `json:"items"`
}

func main() {
	list := flag.Bool("list", false, "列出已安装软件")
	crcDir := flag.String("crc", "", "对指定目录计算 CRC-32")
	jsonOut := flag.Bool("json", false, "以 JSON 输出（-crc 模式）")
	flag.Parse()

	switch {
	case *list:
		runList()
	case *crcDir != "":
		res, err := scanCRC(*crcDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "softscan error:", err)
			os.Exit(1)
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(res)
			return
		}
		for _, it := range res.Items {
			fmt.Printf("%s\t%d\t%s\n", it.CRC32, it.Size, it.Path)
		}
		fmt.Fprintf(os.Stderr, "files=%d skipped=%d bytes=%d elapsed=%dms\n",
			res.Files, res.Skipped, res.Bytes, res.ElapsedMS)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// scanCRC 并发计算目录内全部文件的 CRC-32。
// 内存恒定：每个文件以 1MB 缓冲流式读取；被占用/无权限文件计入 Skipped，不中断整体采集。
func scanCRC(root string) (*ScanResult, error) {
	start := time.Now()

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 无权限目录等：跳过，不中断
		}
		if d.Type().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	workers := runtime.NumCPU() * 2
	tasks := make(chan string)
	results := make(chan FileCRC, 64)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 1<<20) // 1MB 复用缓冲
			for path := range tasks {
				sum, size, err := fileCRC(path, buf)
				if err != nil {
					results <- FileCRC{Path: path, Size: -1} // 失败标记
					continue
				}
				results <- FileCRC{Path: path, Size: size, CRC32: sum}
			}
		}()
	}

	res := &ScanResult{Dir: root}
	done := make(chan struct{})
	go func() {
		for r := range results {
			if r.Size < 0 {
				res.Skipped++
			} else {
				res.Files++
				res.Bytes += r.Size
				res.Items = append(res.Items, r)
			}
		}
		close(done)
	}()

	for _, f := range files {
		tasks <- f
	}
	close(tasks)
	wg.Wait()
	close(results)
	<-done

	res.ElapsedMS = time.Since(start).Milliseconds()
	return res, nil
}

func fileCRC(path string, buf []byte) (string, int64, error) {
	f, err := os.Open(path) // Windows 下以共享读打开，多数被占用文件仍可读
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := crc32.NewIEEE()
	size, err := io.CopyBuffer(h, f, buf)
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("%08x", h.Sum32()), size, nil
}

// runList 输出已安装软件清单（跨平台分派）。
func runList() {
	for _, s := range installedSoftware() {
		fmt.Printf("%s\t%s\t%s\n", s.Name, s.Version, s.Publisher)
	}
	fmt.Fprintf(os.Stderr, "total software: %d (%s)\n", len(installedSoftware()), runtime.GOOS)
}

type Software struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
}

var _ = strings.TrimSpace // 预留：Linux 分支使用
