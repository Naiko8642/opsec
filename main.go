package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

//go:embed elliot.jpg
var elliotImage []byte

func main() {
	fmt.Print("\033]0;i am hacker\007")

	cleanupChan := make(chan os.Signal, 1)
	signal.Notify(cleanupChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cleanupChan
		restoreTerminal()
		os.Exit(0)
	}()

	homeDir, err := os.UserHomeDir()
	if err == nil {
		imgPath := filepath.Join(homeDir, ".elliot_opsec.jpg")
		if !isWallpaperAlreadySet(imgPath) {
			_ = os.WriteFile(imgPath, elliotImage, 0644)
			setWallpaper(imgPath)
		}
	}

	runNativeMatrix()
}

func restoreTerminal() {
	fmt.Print("\033[?25h\033[0m\033[2J\033[H")
}

func isWallpaperAlreadySet(imgPath string) bool {
	existingData, err := os.ReadFile(imgPath)
	if err != nil {
		return false
	}
	hashEmbedded := sha256.Sum256(elliotImage)
	hashExisting := sha256.Sum256(existingData)
	return bytes.Equal(hashEmbedded[:], hashExisting[:])
}

func setWallpaper(imgPath string) {
	if runtime.GOOS == "windows" {
		setWindowsWallpaper(imgPath)
		return
	}

	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	session := strings.ToLower(os.Getenv("DESKTOP_SESSION"))

	switch {
	case strings.Contains(desktop, "gnome"), strings.Contains(desktop, "ubuntu"):
		_ = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+imgPath).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", "file://"+imgPath).Run()

	case strings.Contains(desktop, "kde"), strings.Contains(desktop, "plasma"):
		script := fmt.Sprintf(`string:
			var Desktops = desktops();
			for (i=0;i<Desktops.length;i++) {
				d = Desktops[i];
				d.wallpaperPlugin = "org.kde.image";
				d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
				d.writeConfig("Image", "%s");
			}`, imgPath)
		_ = exec.Command("qdbus", "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", script).Run()

	case strings.Contains(desktop, "xfce"):
		_ = exec.Command("xfconf-query", "-c", "xfce4-desktop", "-p", "/backdrop/screen0/monitor0/workspace0/last-image", "-s", imgPath).Run()
		_ = exec.Command("xfconf-query", "-c", "xfce4-desktop", "-p", "/backdrop/screen0/monitor0/image-path", "-s", imgPath).Run()

	case strings.Contains(desktop, "cinnamon"):
		_ = exec.Command("gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", "file://"+imgPath).Run()

	case strings.Contains(desktop, "mate"):
		_ = exec.Command("gsettings", "set", "org.mate.background", "picture-filename", imgPath).Run()

	case strings.Contains(desktop, "lxde"), strings.Contains(session, "lxde"):
		_ = exec.Command("pcmanfm", "--set-wallpaper", imgPath).Run()

	case strings.Contains(desktop, "lxqt"):
		_ = exec.Command("pcmanfm-qt", "--set-wallpaper", imgPath).Run()

	case strings.Contains(desktop, "hyprland"):
		if err := exec.Command("hyprctl", "hyprpaper", "wallpaper", ","+imgPath).Run(); err != nil {
			_ = exec.Command("swaybg", "-i", imgPath, "-m", "fill").Start()
		}

	case strings.Contains(desktop, "sway"):
		_ = exec.Command("swaybg", "-i", imgPath, "-m", "fill").Start()

	default:
		if err := exec.Command("feh", "--bg-scale", imgPath).Run(); err != nil {
			_ = exec.Command("nitrogen", "--set-zoom-fill", imgPath).Run()
		}
	}
}

func setWindowsWallpaper(imgPath string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	systemParametersInfo := user32.NewProc("SystemParametersInfoW")
	imgPathPtr, err := syscall.UTF16PtrFromString(imgPath)
	if err != nil {
		return
	}
	_, _, _ = systemParametersInfo.Call(0x0014, 0, uintptr(unsafePointer(imgPathPtr)), 0x01|0x02)
}

func runNativeMatrix() {
	fmt.Print("\033[?25l\033[2J")

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	width, height := getTermSize()
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%^&*!~")

	drops := make([]int, width)
	isJwm := make([]bool, width)

	for i := range drops {
		drops[i] = rand.Intn(height)
		isJwm[i] = rand.Float32() < 0.25
	}

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	for {
		select {
		case <-sigwinch:
			width, height = getTermSize()
			drops = make([]int, width)
			isJwm = make([]bool, width)
			for i := range drops {
				drops[i] = rand.Intn(height)
				isJwm[i] = rand.Float32() < 0.25
			}
			out.WriteString("\033[2J")

		case <-ticker.C:
			for x := 0; x < width; x++ {
				y := drops[x]

				if isJwm[x] {
					drawJwmSequence(out, x, y, height)
				} else {
					drawNormalSequence(out, x, y, height, chars)
				}

				if y > height+12 && rand.Float32() > 0.88 {
					drops[x] = 0
					isJwm[x] = rand.Float32() < 0.25
				} else {
					drops[x]++
				}
			}
			out.Flush()
		}
	}
}

func getTermSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

func drawNormalSequence(out *bufio.Writer, x, y, height int, chars []rune) {
	if y > 0 && y <= height {
		fmt.Fprintf(out, "\033[%d;%dH\033[1;37m%c", y, x+1, chars[rand.Intn(len(chars))])
	}
	if y-1 > 0 && y-1 <= height {
		fmt.Fprintf(out, "\033[%d;%dH\033[1;32m%c", y-1, x+1, chars[rand.Intn(len(chars))])
	}
	if y-4 > 0 && y-4 <= height {
		fmt.Fprintf(out, "\033[%d;%dH\033[0;32m%c", y-4, x+1, chars[rand.Intn(len(chars))])
	}
	if y-12 > 0 && y-12 <= height {
		fmt.Fprintf(out, "\033[%d;%dH ", y-12, x+1)
	}
}

func drawJwmSequence(out *bufio.Writer, x, y, height int) {
	jwm := []rune{'J', 'w', 'm'}

	if y > 0 && y <= height {
		fmt.Fprintf(out, "\033[%d;%dH\033[1;37m%c", y, x+1, jwm[0])
	}
	if y-1 > 0 && y-1 <= height {
		fmt.Fprintf(out, "\033[%d;%dH\033[1;32m%c", y-1, x+1, jwm[1])
	}
	if y-2 > 0 && y-2 <= height {
		fmt.Fprintf(out, "\033[%d;%dH\033[1;32m%c", y-2, x+1, jwm[2])
	}
	if y-10 > 0 && y-10 <= height {
		fmt.Fprintf(out, "\033[%d;%dH ", y-10, x+1)
	}
}

func unsafePointer(ptr *uint16) uintptr {
	return uintptr(uintptr(0) + uintptr(uintptr(0)))
}