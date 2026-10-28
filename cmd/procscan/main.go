// procscan 采集系统进程列表（PID / PPID / 进程名）。
// 跨平台：Windows 走 CreateToolhelp32Snapshot 系列系统调用，
// Linux 直接解析 /proc 伪文件系统（麒麟 / 鲲鹏 arm64 可直接交叉编译运行）。
package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type Proc struct {
	PID  int
	PPID int
	Name string
}

func main() {
	list, err := listProcesses()
	if err != nil {
		fmt.Fprintln(os.Stderr, "procscan error:", err)
		os.Exit(1)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].PID < list[j].PID })
	w := os.Stdout
	fmt.Fprintln(w, "PID\tPPID\tName")
	for _, p := range list {
		fmt.Fprintf(w, "%d\t%d\t%s\n", p.PID, p.PPID, p.Name)
	}
	fmt.Fprintf(os.Stderr, "total processes: %d (%s/%s)\n", len(list), runtime.GOOS, runtime.GOARCH)
}

func itoa(n int) string { return strconv.Itoa(n) }

func trimProc(s string) string { return strings.Trim(s, "\x00") }
