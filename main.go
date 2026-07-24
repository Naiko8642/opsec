package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/term"
)

//go:embed elliot.jpg
var embeddedImageData []byte

const (
	brightGreen = "\033[1;32m"
	dimGreen    = "\033[2;32m"
	brightWhite = "\033[1;97m"
	jwmWhite    = "\033[1;97m"
)

var matrixChars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%^&*()*&^%~`|/\\{}[]")

type Drop struct {
	y      float64
	speed  float64
	length int
}

func main() {
	_ = extractAndSetWallpaper()

	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))

	fmt.Print("\033[?25l\033[2J")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cleanExit(oldState)
	}()

	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err == nil && n > 0 {
				if buf[0] == 27 || buf[0] == 3 {
					cleanExit(oldState)
				}
			}
		}
	}()

	runMatrixRain()
}

func cleanExit(oldState *term.State) {
	fmt.Print("\033[?25h\033[0m\033[2J\033[1;1H")
	if oldState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}
	os.Exit(0)
}

func extractAndSetWallpaper() error {
	tempDir := os.TempDir()
	imgPath := filepath.Join(tempDir, "elliot.jpg")

	if err := os.WriteFile(imgPath, embeddedImageData, 0644); err != nil {
		return err
	}

	return setWallpaper(imgPath)
}

func setWallpaper(imagePath string) error {
	switch runtime.GOOS {
	case "windows":
		psCmd := fmt.Sprintf(
			`Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public class Wallpaper { [DllImport("user32.dll", CharSet = CharSet.Auto)] public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni); }'; [Wallpaper]::SystemParametersInfo(20, 0, "%s", 3)`,
			imagePath,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
		return cmd.Run()

	case "linux":
		var err error
		if _, e := exec.LookPath("gsettings"); e == nil {
			err = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://"+imagePath).Run()
			_ = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", "file://"+imagePath).Run()
		} else if _, e := exec.LookPath("hyprctl"); e == nil {
			err = exec.Command("hyprctl", "hyprpaper", "wallpaper", ","+imagePath).Run()
		} else if _, e := exec.LookPath("swaymsg"); e == nil {
			err = exec.Command("swaymsg", "output * bg", imagePath, "fill").Run()
		} else if _, e := exec.LookPath("feh"); e == nil {
			err = exec.Command("feh", "--bg-fill", imagePath).Run()
		}
		return err

	default:
		return fmt.Errorf("unsupported operating system")
	}
}

func runMatrixRain() {
	rand.Seed(time.Now().UnixNano())
	writer := bufio.NewWriter(os.Stdout)

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 || height <= 0 {
		width, height = 80, 24
	}

	drops := make([]Drop, width)
	for i := range drops {
		resetDrop(&drops[i], height)
		drops[i].y = float64(rand.Intn(height))
	}

	ticker := time.NewTicker(45 * time.Millisecond)
	defer ticker.Stop()

	jwmText := []rune("JWM")
	jwmX := -1
	jwmY := -1

	spawnJWM := func(w, h int) {
		if w > 5 && h > 2 {
			jwmX = 2 + rand.Intn(w-5)
			jwmY = 2 + rand.Intn(h-2)
		}
	}

	spawnJWM(width, height)

	for range ticker.C {
		if newW, newH, err := term.GetSize(int(os.Stdout.Fd())); err == nil && newW > 0 && newH > 0 {
			if newW != width || newH != height {
				width, height = newW, newH
				drops = make([]Drop, width)
				for i := range drops {
					resetDrop(&drops[i], height)
				}
				writer.WriteString("\033[2J")
				spawnJWM(width, height)
			}
		}

		jwmHit := false

		for x := 0; x < width; x++ {
			d := &drops[x]
			headY := int(d.y)

			if headY > 0 && headY <= height {
				char := matrixChars[rand.Intn(len(matrixChars))]
				writer.WriteString(fmt.Sprintf("\033[%d;%dH%s%c", headY, x+1, brightWhite, char))
			}

			for i := 1; i <= d.length; i++ {
				trailY := headY - i
				if trailY > 0 && trailY <= height {
					char := matrixChars[rand.Intn(len(matrixChars))]
					var color string
					if i < d.length/3 {
						color = brightGreen
					} else {
						color = dimGreen
					}
					writer.WriteString(fmt.Sprintf("\033[%d;%dH%s%c", trailY, x+1, color, char))
				}
			}

			clearY := headY - d.length - 1
			if clearY > 0 && clearY <= height {
				writer.WriteString(fmt.Sprintf("\033[%d;%dH ", clearY, x+1))
			}

			if jwmX > 0 && jwmY > 0 {
				for i := 0; i < len(jwmText); i++ {
					targetX := jwmX + i
					if x+1 == targetX {
						if headY >= jwmY && headY-d.length <= jwmY {
							jwmHit = true
						}
					}
				}
			}

			d.y += d.speed
			if int(d.y)-d.length > height {
				resetDrop(d, height)
			}
		}

		if jwmHit {
			spawnJWM(width, height)
		} else if jwmX > 0 && jwmY > 0 {
			for i, ch := range jwmText {
				targetX := jwmX + i
				if targetX <= width {
					writer.WriteString(fmt.Sprintf("\033[%d;%dH%s%c", jwmY, targetX, jwmWhite, ch))
				}
			}
		}

		writer.Flush()
	}
}

func resetDrop(d *Drop, height int) {
	d.y = 0
	d.speed = 0.25 + rand.Float64()*0.45
	d.length = 10 + rand.Intn(14)
}
