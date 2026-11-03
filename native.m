#import <Foundation/Foundation.h>
#import <CoreGraphics/CoreGraphics.h>
#import <IOKit/IOKitLib.h>
#import <mach/mach.h>
#import <sys/sysctl.h>
#import <sys/mount.h>
#import <sys/time.h>
#import <stdint.h>
#import <stdio.h>
#import <stdlib.h>
#import <string.h>

static char *copyString(NSString *s) {
    return s ? strdup(s.UTF8String) : strdup("Unknown");
}

char *gf_os_version(void) {
    @autoreleasepool {
        NSOperatingSystemVersion v = NSProcessInfo.processInfo.operatingSystemVersion;
        return copyString([NSString stringWithFormat:@"macOS %ld.%ld.%ld", (long)v.majorVersion, (long)v.minorVersion, (long)v.patchVersion]);
    }
}

char *gf_sysctl_string(const char *key) {
    size_t size = 0;
    if (sysctlbyname(key, NULL, &size, NULL, 0) != 0 || size == 0) return strdup("Unknown");
    char *buf = calloc(size + 1, 1);
    if (!buf) return strdup("Unknown");
    if (sysctlbyname(key, buf, &size, NULL, 0) != 0) { free(buf); return strdup("Unknown"); }
    return buf;
}

char *gf_sysctl_int(const char *key) {
    int64_t value = 0;
    size_t size = sizeof(value);
    if (sysctlbyname(key, &value, &size, NULL, 0) != 0) return strdup("Unknown");
    // Integer sysctls may be 32- or 64-bit; reinterpret based on returned size.
    if (size == sizeof(int32_t)) value = *(int32_t *)&value;
    else if (size != sizeof(int64_t)) return strdup("Unknown");
    char buf[32];
    snprintf(buf, sizeof(buf), "%lld", (long long)value);
    return strdup(buf);
}

char *gf_memory(void) {
    uint64_t total = 0;
    size_t size = sizeof(total);
    if (sysctlbyname("hw.memsize", &total, &size, NULL, 0) != 0 || !total) return strdup("Unknown");
    mach_port_t host = mach_host_self();
    vm_size_t pageSize = 0;
    vm_statistics64_data_t stats = {0};
    mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
    kern_return_t a = host_page_size(host, &pageSize);
    kern_return_t b = host_statistics64(host, HOST_VM_INFO64, (host_info64_t)&stats, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (a != KERN_SUCCESS || b != KERN_SUCCESS) return strdup("Unknown");
    uint64_t used = ((uint64_t)stats.active_count + stats.wire_count + stats.compressor_page_count) * (uint64_t)pageSize;
    if (used > total) used = total;
    return copyString([NSString stringWithFormat:@"%.2f / %.2f GiB", used / 1073741824.0, total / 1073741824.0]);
}

char *gf_gpu(void) {
    @autoreleasepool {
        // IOAccelerator is exposed by both Intel/AMD and Apple Silicon graphics drivers.
        CFMutableDictionaryRef match = IOServiceMatching("IOAccelerator");
        if (!match) return strdup("Unknown");
        io_iterator_t iterator = IO_OBJECT_NULL;
        if (IOServiceGetMatchingServices(kIOMainPortDefault, match, &iterator) != KERN_SUCCESS) return strdup("Unknown");
        io_service_t service;
        NSMutableOrderedSet<NSString *> *names = [NSMutableOrderedSet orderedSet];
        while ((service = IOIteratorNext(iterator))) {
            io_registry_entry_t parent = IO_OBJECT_NULL;
            io_registry_entry_t entry = service;
            // Look for a human-readable model on the accelerator or its parents.
            for (int depth = 0; depth < 5; depth++) {
                CFTypeRef prop = IORegistryEntryCreateCFProperty(entry, CFSTR("model"), kCFAllocatorDefault, 0);
                if (prop) {
                    NSString *model = nil;
                    if (CFGetTypeID(prop) == CFStringGetTypeID()) model = (__bridge NSString *)prop;
                    else if (CFGetTypeID(prop) == CFDataGetTypeID()) {
                        NSData *data = (__bridge NSData *)prop;
                        model = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
                        model = [model stringByTrimmingCharactersInSet:[NSCharacterSet controlCharacterSet]];
                    }
                    if (model.length) [names addObject:model];
                    CFRelease(prop);
                    if (model.length) break;
                }
                if (IORegistryEntryGetParentEntry(entry, kIOServicePlane, &parent) != KERN_SUCCESS) break;
                if (entry != service) IOObjectRelease(entry);
                entry = parent;
            }
            if (entry != service) IOObjectRelease(entry);
            IOObjectRelease(service);
        }
        IOObjectRelease(iterator);
        if (names.count) return copyString([[names array] componentsJoinedByString:@", "]);
        // On some Apple Silicon machines the GPU model is not exposed as a property.
        char *cpu = gf_sysctl_string("machdep.cpu.brand_string");
        if (cpu && strstr(cpu, "Apple")) return cpu;
        free(cpu);
        return strdup("Unknown");
    }
}

char *gf_displays(void) {
    uint32_t count = 0;
    if (CGGetOnlineDisplayList(0, NULL, &count) != kCGErrorSuccess || !count) return strdup("Unknown");
    CGDirectDisplayID *ids = calloc(count, sizeof(CGDirectDisplayID));
    if (!ids) return strdup("Unknown");
    if (CGGetOnlineDisplayList(count, ids, &count) != kCGErrorSuccess) { free(ids); return strdup("Unknown"); }
    NSMutableArray<NSString *> *displays = [NSMutableArray array];
    for (uint32_t i = 0; i < count; i++) {
        CGDisplayModeRef mode = CGDisplayCopyDisplayMode(ids[i]);
        if (mode) {
            // Logical dimensions; may differ from physical pixels on Retina displays.
            [displays addObject:[NSString stringWithFormat:@"%zux%zu", CGDisplayModeGetWidth(mode), CGDisplayModeGetHeight(mode)]];
            CGDisplayModeRelease(mode);
        }
    }
    free(ids);
    return copyString([displays componentsJoinedByString:@", "]);
}

char *gf_disk(void) {
    struct statfs fs;
    if (statfs("/", &fs) != 0) return strdup("Unknown");
    double total = (double)fs.f_blocks * fs.f_bsize;
    double freeBytes = (double)fs.f_bfree * fs.f_bsize;
    double used = total - freeBytes;
    return copyString([NSString stringWithFormat:@"%.1f / %.1f GiB", used / 1073741824.0, total / 1073741824.0]);
}

char *gf_uptime(void) {
    struct timeval boot = {0};
    size_t size = sizeof(boot);
    if (sysctlbyname("kern.boottime", &boot, &size, NULL, 0) != 0) return strdup("Unknown");
    time_t now = time(NULL);
    if (now < boot.tv_sec) return strdup("Unknown");
    uint64_t sec = (uint64_t)(now - boot.tv_sec);
    return copyString([NSString stringWithFormat:@"%llud %lluh %llum", (unsigned long long)(sec / 86400), (unsigned long long)((sec / 3600) % 24), (unsigned long long)((sec / 60) % 60)]);
}
