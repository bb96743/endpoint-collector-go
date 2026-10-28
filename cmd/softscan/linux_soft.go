//go:build linux

package main

import (
	"os"
	"strings"
)

// installedSoftware 解析 dpkg 包库（/var/lib/dpkg/status）生成软件清单。
// rpm 系（麒麟）可按同样思路解析 `rpm -qa` 输出，此处先覆盖 deb 系。
func installedSoftware() []Software {
	b, err := os.ReadFile("/var/lib/dpkg/status")
	if err != nil {
		return nil
	}
	var out []Software
	var cur Software
	flush := func() {
		if cur.Name != "" {
			out = append(out, cur)
			cur = Software{}
		}
	}
	for _, line := range strings.Split(string(b), "\n") {
		switch {
		case strings.HasPrefix(line, "Package: "):
			flush()
			cur.Name = strings.TrimPrefix(line, "Package: ")
		case strings.HasPrefix(line, "Version: "):
			cur.Version = strings.TrimPrefix(line, "Version: ")
		case strings.HasPrefix(line, "Maintainer: "):
			cur.Publisher = strings.TrimPrefix(line, "Maintainer: ")
		case line == "":
			flush()
		}
	}
	flush()
	return out
}
