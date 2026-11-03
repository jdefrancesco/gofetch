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

const reset = "\x1b[0m"
const cyan = "\x1b[36m"
const green = "\x1b[32m"
const bold = "\x1b[1m"
const logoGap = 6

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

func fromNative(p *C.char) string {
    if p == nil { return "Unknown" }
    defer C.free(unsafe.Pointer(p))
    s := strings.TrimSpace(C.GoString(p))
    if s == "" { return "Unknown" }
    return s
}

func sysctl(key string) string {
    k := C.CString(key)
    defer C.free(unsafe.Pointer(k))
    return fromNative(C.gf_sysctl_string(k))
}

func sysctlInt(key string) string {
    k := C.CString(key)
    defer C.free(unsafe.Pointer(k))
    return fromNative(C.gf_sysctl_int(k))
}

func envBase(key string) string {
    v := os.Getenv(key)
    if v == "" { return "Unknown" }
    return filepath.Base(v)
}

func main() {
    if runtime.GOOS != "darwin" { fmt.Fprintln(os.Stderr, "gofetch requires macOS"); os.Exit(1) }
    host, err := os.Hostname()
    if err != nil { host = "Unknown" }
    name := os.Getenv("USER")
    if u, err := user.Current(); err == nil { name = u.Username }
    if name == "" { name = "Unknown" }

    cpu := sysctl("machdep.cpu.brand_string")
    if cpu == "Unknown" { cpu = sysctl("hw.model") }
    cores := sysctlInt("hw.physicalcpu")
    if cores != "Unknown" { cpu += " (" + cores + " cores)" }
    terminal := os.Getenv("TERM_PROGRAM")
    if terminal == "" { terminal = os.Getenv("TERM") }
    if terminal == "" { terminal = "Unknown" }

    rows := []string{
        bold + green + name + "@" + host + reset,
        strings.Repeat("─", 34),
        field("OS", fromNative(C.gf_os_version())),
        field("Host", sysctl("hw.model")),
        field("Kernel", sysctl("kern.osrelease")),
        field("Arch", runtime.GOARCH),
        field("CPU", cpu),
        field("GPU", fromNative(C.gf_gpu())),
        field("Memory", fromNative(C.gf_memory())),
        field("Disk", fromNative(C.gf_disk())),
        field("Uptime", fromNative(C.gf_uptime())),
        field("Shell", envBase("SHELL")),
        field("Terminal", terminal),
        field("Display", fromNative(C.gf_displays())),
        "",
        "\x1b[31m● \x1b[32m● \x1b[33m● \x1b[34m● \x1b[35m● \x1b[36m● \x1b[37m●" + reset,
    }
    fmt.Println()
    artWidth := 0
    for _, line := range logo { artWidth = max(artWidth, visibleWidth(line)) }
    for i := 0; i < max(len(rows), len(logo)); i++ {
        art, row := "", ""
        if i < len(logo) { art = logo[i] }
        if i < len(rows) { row = rows[i] }
        fmt.Printf("%s%s%s\n", art, strings.Repeat(" ", artWidth+logoGap-visibleWidth(art)), row)
    }
    fmt.Println()
}

func field(k, v string) string { return fmt.Sprintf("%s%s%-10s%s %s", bold, cyan, k, reset, v) }

func visibleWidth(s string) int {
    width := 0
    esc := false
    for _, r := range s {
        if r == '\x1b' { esc = true; continue }
        if esc { if r == 'm' { esc = false }; continue }
        width++
    }
    return width
}
