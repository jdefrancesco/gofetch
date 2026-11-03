package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Foundation -framework IOKit -framework CoreGraphics -framework CoreFoundation
#include <stdlib.h>
char *gf_os_version(void);
char *gf_sysctl_string(const char *key);
char *gf_sysctl_int(const char *key);
char *gf_memory(void);
char *gf_gpu(void);
char *gf_displays(void);
char *gf_disk(void);
char *gf_uptime(void);
*/
import "C"

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

const (
	reset   = "\x1b[0m"
	cyan    = "\x1b[36m"
	green   = "\x1b[32m"
	bold    = "\x1b[1m"
	logoGap = 6
	unknown = "Unknown"
)

var logo = []string{
	"\x1b[32m                    c.'\x1b[0m",
	"\x1b[32m                 ,xNMM.\x1b[0m",
	"\x1b[32m               .OMMMMo\x1b[0m",
	"\x1b[32m               OMMM0,\x1b[0m",
	"\x1b[32m     .;loddo:' loolloddol;.\x1b[0m",
	"\x1b[32m   cKMMMMMMMMMMNWMMMMMMMMMM0:\x1b[0m",
	"\x1b[33m .KMMMMMMMMMMMMMMMMMMMMMMMWd.\x1b[0m",
	"\x1b[33m XMMMMMMMMMMMMMMMMMMMMMMMX.\x1b[0m",
	"\x1b[31m;MMMMMMMMMMMMMMMMMMMMMMMM:\x1b[0m",
	"\x1b[31m:MMMMMMMMMMMMMMMMMMMMMMMM:\x1b[0m",
	"\x1b[31m.MMMMMMMMMMMMMMMMMMMMMMMMX.\x1b[0m",
	"\x1b[35m kMMMMMMMMMMMMMMMMMMMMMMMMWd.\x1b[0m",
	"\x1b[35m .XMMMMMMMMMMMMMMMMMMMMMMMMMMk\x1b[0m",
	"\x1b[34m  .XMMMMMMMMMMMMMMMMMMMMMMMMK.\x1b[0m",
	"\x1b[34m    kMMMMMMMMMMMMMMMMMMMMMMd\x1b[0m",
	"\x1b[34m     ;KMMMMMMMWXXWMMMMMMMk.\x1b[0m",
	"\x1b[34m       .cooc,.    .,coo:.\x1b[0m",
}

// fromNative copies a malloc-allocated native string and frees it.
// It trims surrounding whitespace and returns unknown for nil or empty strings.
func fromNative(p *C.char) string {
	if p == nil {
		return unknown
	}
	defer C.free(unsafe.Pointer(p))
	s := strings.TrimSpace(C.GoString(p))
	if s == "" {
		return unknown
	}
	return s
}

// sysctl reads a string sysctl value, returning unknown when unavailable.
func sysctl(key string) string {
	k := C.CString(key)
	defer C.free(unsafe.Pointer(k))
	return fromNative(C.gf_sysctl_string(k))
}

// sysctlInt reads an integer sysctl value as decimal text, returning unknown when unavailable.
func sysctlInt(key string) string {
	k := C.CString(key)
	defer C.free(unsafe.Pointer(k))
	return fromNative(C.gf_sysctl_int(k))
}

// envBase returns the base name of an environment variable's path, or unknown if empty.
func envBase(key string) string {
	v := os.Getenv(key)
	if v == "" {
		return unknown
	}
	return filepath.Base(v)
}

// main checks for macOS and prints the system information report.
func main() {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "gofetch requires macOS")
		os.Exit(1)
	}
	printReport(systemRows())
}

// username returns the current user's name, falling back to USER if lookup fails.
// It returns unknown if the resulting name is empty.
func username() string {
	name := os.Getenv("USER")
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	if name == "" {
		return unknown
	}
	return name
}

// cpuDescription returns the CPU brand, or hardware model, with the physical core count when available.
func cpuDescription() string {
	cpu := sysctl("machdep.cpu.brand_string")
	if cpu == unknown {
		cpu = sysctl("hw.model")
	}
	if cores := sysctlInt("hw.physicalcpu"); cores != unknown {
		cpu += " (" + cores + " cores)"
	}
	return cpu
}

// terminalName returns TERM_PROGRAM, falling back to TERM and then unknown.
func terminalName() string {
	for _, key := range []string{"TERM_PROGRAM", "TERM"} {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return unknown
}

// systemRows collects system information and formats the report's text rows.
func systemRows() []string {
	host, err := os.Hostname()
	if err != nil {
		host = unknown
	}
	return []string{
		bold + green + username() + "@" + host + reset,
		strings.Repeat("─", 34),
		field("OS", fromNative(C.gf_os_version())),
		field("Host", sysctl("hw.model")),
		field("Kernel", sysctl("kern.osrelease")),
		field("Arch", runtime.GOARCH),
		field("CPU", cpuDescription()),
		field("GPU", fromNative(C.gf_gpu())),
		field("Memory", fromNative(C.gf_memory())),
		field("Disk", fromNative(C.gf_disk())),
		field("Uptime", fromNative(C.gf_uptime())),
		field("Shell", envBase("SHELL")),
		field("Terminal", terminalName()),
		field("Display", fromNative(C.gf_displays())),
		"",
		"\x1b[31m● \x1b[32m● \x1b[33m● \x1b[34m● \x1b[35m● \x1b[36m● \x1b[37m●" + reset,
	}
}

// printReport writes the logo and aligned report rows to standard output.
func printReport(rows []string) {
	fmt.Println()
	artWidth := 0
	for _, line := range logo {
		artWidth = max(artWidth, visibleWidth(line))
	}
	for i := 0; i < max(len(rows), len(logo)); i++ {
		art, row := "", ""
		if i < len(logo) {
			art = logo[i]
		}
		if i < len(rows) {
			row = rows[i]
		}
		fmt.Printf("%s%s%s\n", art, strings.Repeat(" ", artWidth+logoGap-visibleWidth(art)), row)
	}
	fmt.Println()
}

// field formats a value with a colored label padded to at least ten characters.
func field(label, value string) string {
	return fmt.Sprintf("%s%s%-10s%s %s", bold, cyan, label, reset, value)
}

// visibleWidth counts runes excluding the ANSI color sequences used by the logo.
// It assumes each visible rune occupies one terminal column.
func visibleWidth(s string) int {
	width := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		width++
	}
	return width
}
