# opsec

> Tool for larp kings — now in pure x86_64 Assembly.

Zero dependencies, zero `libc`, pure Linux syscalls. 100% OPSEC.

## Features

- **Embedded Assets:** `elliot.jpg` is embedded directly into the binary `.rodata` via `.incbin`.
- **Pure Linux Syscalls:**
  - `sys_openat` (257) & `sys_write` (1) to drop `/tmp/elliot.jpg`.
  - `sys_fork` (57) & `sys_execve` (59) to set wallpaper using `feh`.
  - `sys_wait4` (61) to wait for wallpaper update.
  - `sys_getrandom` (318) & `sys_nanosleep` (35) for true random terminal matrix rain.
- **Binary Size:** ~323 KB (of which 321 KB is `elliot.jpg`).

## Building

```bash
make
./opsec
```
