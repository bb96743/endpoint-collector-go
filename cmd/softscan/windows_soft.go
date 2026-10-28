//go:build windows

package main

import (
	"golang.org/x/sys/windows/registry"
)

// installedSoftware 读取注册表 Uninstall 键生成软件资产清单。
// 覆盖 HKLM 64 位视图与 WOW6432Node（32 位程序），字段：DisplayName / DisplayVersion / Publisher。
func installedSoftware() []Software {
	views := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	seen := map[string]bool{}
	var out []Software
	for _, v := range views {
		k, err := registry.OpenKey(v.root, v.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadSubKeyNames(-1)
		k.Close()

		for _, name := range names {
			sk, err := registry.OpenKey(v.root, v.path+`\`+name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			display, _, _ := sk.GetStringValue("DisplayName")
			version, _, _ := sk.GetStringValue("DisplayVersion")
			publisher, _, _ := sk.GetStringValue("Publisher")
			sk.Close()

			if display == "" || seen[display+"\x00"+version] {
				continue
			}
			seen[display+"\x00"+version] = true
			out = append(out, Software{Name: display, Version: version, Publisher: publisher})
		}
	}
	return out
}
