package arkose

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	cReset  = "\x1b[0m"
	cDim    = "\x1b[2m"
	cCyan   = "\x1b[36m"
	cGreen  = "\x1b[32m"
	cGreenB = "\x1b[1;32m"
	cYellow = "\x1b[33m"
	cRedB   = "\x1b[1;31m"
	cGray   = "\x1b[90m"
)

var colorsEnabled = true

func enableWindowsANSI() {
	const enableVTP = 0x0004
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getStdHandle := kernel32.NewProc("GetStdHandle")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	handle, _, _ := getStdHandle.Call(uintptr(^uint32(11 - 1)))
	if handle == 0 || handle == ^uintptr(0) {
		colorsEnabled = false
		return
	}
	var mode uint32
	r1, _, _ := getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		colorsEnabled = false
		return
	}
	newMode := mode | enableVTP
	r2, _, _ := setConsoleMode.Call(handle, uintptr(newMode))
	if r2 == 0 {
		colorsEnabled = false
	}
}

func colorize(color, s string) string {
	if !colorsEnabled {
		return s
	}
	return color + s + cReset
}

func timestamp() string {
	return time.Now().Format("15:04:05.000")
}

func logInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s  %s\n",
		colorize(cGray, "["+timestamp()+"]"),
		colorize(cCyan, "[INFO ]"),
		msg,
	)
}

func logPhase(step, total int, name string, elapsed time.Duration, extra string) {
	pad := 32 - len(name)
	if pad < 1 {
		pad = 1
	}
	spaces := ""
	for i := 0; i < pad; i++ {
		spaces += " "
	}
	elapsedStr := fmt.Sprintf("ok in %4d ms", elapsed.Milliseconds())
	if extra != "" {
		elapsedStr += " " + extra
	}
	fmt.Fprintf(os.Stdout, "%s %s  %d/%d %s ...%s%s\n",
		colorize(cGray, "["+timestamp()+"]"),
		colorize(cGreen, "[PHASE]"),
		step, total, name, spaces,
		colorize(cDim, elapsedStr),
	)
}

func logPhaseSkipped(step, total int, name, reason string) {
	pad := 32 - len(name)
	if pad < 1 {
		pad = 1
	}
	spaces := ""
	for i := 0; i < pad; i++ {
		spaces += " "
	}
	fmt.Fprintf(os.Stdout, "%s %s  %d/%d %s ...%s%s\n",
		colorize(cGray, "["+timestamp()+"]"),
		colorize(cGreen, "[PHASE]"),
		step, total, name, spaces,
		colorize(cDim, "skipped ("+reason+")"),
	)
}

func logOK(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s  %s\n",
		colorize(cGray, "["+timestamp()+"]"),
		colorize(cGreenB, "[OK   ]"),
		msg,
	)
}

func logWarn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s  %s\n",
		colorize(cGray, "["+timestamp()+"]"),
		colorize(cYellow, "[WARN ]"),
		msg,
	)
}

func logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s  %s\n",
		colorize(cGray, "["+timestamp()+"]"),
		colorize(cRedB, "[ERROR]"),
		msg,
	)
}
