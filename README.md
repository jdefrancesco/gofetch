# gofetch

Native macOS system information CLI written in Go and Objective-C (cgo).

## Build

Requires macOS, Xcode Command Line Tools, and Go 1.22+.

```sh
xcode-select --install  # if not installed
CGO_ENABLED=1 go build -o gofetch .
./gofetch
```

No third-party Go modules are required. Frameworks: Foundation, IOKit, CoreGraphics, CoreFoundation.

## Notes

- macOS only; must build on macOS (the Linux container cannot compile Apple frameworks).
- GPU detection depends on IORegistry properties and may fall back to the CPU brand on Apple Silicon.
- Memory usage is an approximation: active + wired + compressor pages, not Activity Monitor's memory pressure.
- Disk statistics describe the root filesystem; APFS volumes share container space.
- Display dimensions are CoreGraphics mode dimensions, not necessarily native panel pixel counts.
- cgo adds Objective-C runtime/framework linkage, but no subprocesses are spawned.
