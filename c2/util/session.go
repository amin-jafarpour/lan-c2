package util  

import (
	"net"
	"errors"
	"time"
    "fmt"
)

func AcceptTCP(lIP string, lPort int) (net.Conn, error){
    var conn net.Conn
    lIPv4 := net.ParseIP(lIP)
    if lIPv4 == nil{
        return conn, errors.New("invalid local ip")
    }
    if lPort > 65535 || lPort < 0{
        return conn, errors.New("invalid local port")
    }
    laddr := &net.TCPAddr{
        IP: lIPv4,
        Port: lPort,
    }
    listener, err := net.ListenTCP("tcp", laddr)
    if err != nil{
        return conn, err 
    }
    defer listener.Close()
    conn, err = listener.AcceptTCP()
    if err != nil{
        return conn, err 
    }
    return conn, nil 
}

func ConnectTCP(lIP string, lPort int, rIP string, rPort int) (net.Conn, error){
    var conn net.Conn
    lIPv4 := net.ParseIP(lIP)
    if lIPv4 == nil{
        return conn, errors.New("invalid local ip")
    }
    if lPort > 65535 || lPort < 0{
        return conn, errors.New("invalid local port")
    }
    if res := net.ParseIP(rIP); res == nil{
        return conn, errors.New("invalid remote ip")
    }
    if rPort > 65535 || rPort < 0{
        return conn, errors.New("invalid remote port")
    }
    localAddr := &net.TCPAddr{
        IP:   lIPv4,
        Port: lPort,
    }
    dialer := &net.Dialer{
        LocalAddr: localAddr,
        Timeout:   time.Second * 3,
    }
    conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", rIP, rPort))
    if err != nil {
        return conn, nil 
    }
    return conn, nil 
}
