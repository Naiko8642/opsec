.global _start

.equ SYS_WRITE,     1
.equ SYS_CLOSE,     3
.equ SYS_NANOSLEEP, 35
.equ SYS_FORK,      57
.equ SYS_EXIT,      60
.equ SYS_WAIT4,     61
.equ SYS_EXECVE,    59
.equ SYS_OPENAT,    257
.equ SYS_GETRANDOM, 318

.equ AT_FDCWD,      -100
.equ O_FLAGS,       577 /* O_WRONLY | O_CREAT | O_TRUNC (0x241) */
.equ MODE_644,      420 /* 0644 */
.equ STDOUT,        1

.section .rodata
tmp_img_path:
    .asciz "/tmp/elliot.jpg"

feh_path:
    .asciz "/usr/bin/feh"
feh_arg0:
    .asciz "feh"
feh_arg1:
    .asciz "--bg-fill"
feh_arg2:
    .asciz "/tmp/elliot.jpg"

.align 8
feh_argv:
    .quad feh_arg0
    .quad feh_arg1
    .quad feh_arg2
    .quad 0

/* ANSI escape sequences */
init_term:
    .ascii "\033[?25l\033[2J\033[H\033[1;32m"
init_len = . - init_term

charset:
    .ascii "0123456789ABCDEFabcdefghijklmnopqrstuvwxyz@#$%&*+=-/\\|<>[]"
charset_len = . - charset

space:
    .ascii " "

sleep_req:
    .quad 0
    .quad 35000000

/* Embed elliot.jpg */
elliot_data:
    .incbin "elliot.jpg"
elliot_len = . - elliot_data

.section .bss
.align 8
saved_envp:
    .quad 0
.lcomm rand_buf, 32

.section .text
_start:
    /* Grab envp pointer from initial stack: envp = rsp + 16 + 8*argc */
    mov (%rsp), %rax
    lea 16(%rsp, %rax, 8), %rbx
    mov %rbx, saved_envp(%rip)

    /* 1. Extract embedded image to /tmp/elliot.jpg via sys_openat */
    mov $SYS_OPENAT, %rax
    mov $AT_FDCWD, %rdi
    lea tmp_img_path(%rip), %rsi
    mov $O_FLAGS, %rdx
    mov $MODE_644, %r10
    syscall

    test %rax, %rax
    js skip_wallpaper
    mov %rax, %r14                  /* fd */

    /* Write embedded data */
    mov $SYS_WRITE, %rax
    mov %r14, %rdi
    lea elliot_data(%rip), %rsi
    mov $elliot_len, %rdx
    syscall

    /* Close fd */
    mov $SYS_CLOSE, %rax
    mov %r14, %rdi
    syscall

    /* 2. Fork and execve feh to set wallpaper */
    mov $SYS_FORK, %rax
    syscall
    test %rax, %rax
    js skip_wallpaper
    jnz parent_process

    /* Child: execve /usr/bin/feh with caller's environment (inheriting DISPLAY) */
    mov $SYS_EXECVE, %rax
    lea feh_path(%rip), %rdi
    lea feh_argv(%rip), %rsi
    mov saved_envp(%rip), %rdx
    syscall

    mov $SYS_EXIT, %rax
    mov $1, %rdi
    syscall

parent_process:
    /* wait4(child_pid, NULL, 0, NULL) */
    mov %rax, %rdi
    mov $SYS_WAIT4, %rax
    xor %rsi, %rsi
    xor %rdx, %rdx
    xor %r10, %r10
    syscall

skip_wallpaper:
    /* 3. Run Matrix Rain */
    mov $SYS_WRITE, %rax
    mov $STDOUT, %rdi
    lea init_term(%rip), %rsi
    mov $init_len, %rdx
    syscall

rain_loop:
    mov $SYS_GETRANDOM, %rax
    lea rand_buf(%rip), %rdi
    mov $32, %rsi
    xor %rdx, %rdx
    syscall

    mov $32, %r13
    lea rand_buf(%rip), %rbx

print_chars:
    test %r13, %r13
    jz do_sleep

    movzbq (%rbx), %rax
    inc %rbx

    mov %rax, %rdx
    and $3, %rdx
    jz print_space

    xor %rdx, %rdx
    mov $charset_len, %rcx
    div %rcx
    lea charset(%rip), %rsi
    add %rdx, %rsi
    jmp do_write

print_space:
    lea space(%rip), %rsi

do_write:
    mov $SYS_WRITE, %rax
    mov $STDOUT, %rdi
    mov $1, %rdx
    syscall

    dec %r13
    jmp print_chars

do_sleep:
    mov $SYS_NANOSLEEP, %rax
    lea sleep_req(%rip), %rdi
    xor %rsi, %rsi
    syscall

    jmp rain_loop
