//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
)

func listProcesses() ([]Proc, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []Proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // 非 /proc/<pid> 目录
		}
		p := Proc{PID: pid}
		if b, err := os.ReadFile("/proc/" + e.Name() + "/comm"); err == nil {
			p.Name = strings.TrimSpace(string(b))
		}
		if b, err := os.ReadFile("/proc/" + e.Name() + "/stat"); err == nil {
			// stat 格式：pid (comm) state ppid ...——comm 可能含空格，取右括号之后
			if i := strings.LastIndexByte(string(b), ')'); i >= 0 {
				fields := strings.Fields(string(b)[i+1:])
				if len(fields) > 1 {
					p.PPID, _ = strconv.Atoi(fields[1])
				}
			}
		}
		if p.Name != "" {
			out = append(out, p)
		}
	}
	return out, nil
}
