package beacon

import (
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"
    "fmt"
    "time"
)

var PortSequence = []int{6000, 7000, 8000}

func SendKnock(route RouteStruct, ports []int) error{                     
    handle, err := pcap.OpenLive(route.Iface.Name, 65536, false, pcap.BlockForever)
    if err != nil {
        return err
    }
    defer handle.Close()
    eth := &layers.Ethernet{
        SrcMAC:       route.SrcMAC,
        DstMAC:       route.DstMAC,
        EthernetType: layers.EthernetTypeIPv4,
    }
    ip := &layers.IPv4{
        Version:  4,
        SrcIP:    route.SrcIP,
        DstIP:    route.DstIP,
        Protocol: layers.IPProtocolTCP,
    }
    tcp := &layers.TCP{
        SrcPort: route.SrcPort,
        DstPort: route.DstPort,
        SYN:     true,
    }
    tcp.SetNetworkLayerForChecksum(ip)
    buffer := gopacket.NewSerializeBuffer()

    for _, port := range append(ports, 6000){
        tcp.DstPort = layers.TCPPort(port)
        opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
        if err := gopacket.SerializeLayers(buffer, opts, eth, ip, tcp); err != nil {
            return err 
        }
        if err := handle.WritePacketData(buffer.Bytes()); err != nil {
            return err 
        }
        time.Sleep(time.Second)
        fmt.Println("Knocking port", port)
    }
	return nil 
}

func ReceiveKnock(route RouteStruct, ports []int) error{                      
    handle, err := pcap.OpenLive(route.Iface.Name, 65536, true, pcap.BlockForever)
    if err != nil {
        return err 
    }
    defer handle.Close()

    part := ""
    for _, port := range ports{
        if part == ""{
            part += fmt.Sprintf("dst port %d", port)
        } else{
            part += fmt.Sprintf(" or dst port %d", port)
        }
    }
    part = " ( " + part + " ) "

	filter := fmt.Sprintf(
		"ip and tcp and " +
		"src host %s and dst host %s and " +
        "src port %d and" + part,
		route.DstIP, route.SrcIP, route.DstPort)

    if err := handle.SetBPFFilter(filter); err != nil {
        return err
    }

    indexOf := func (target int, slice []int) int {
        for i, v := range slice {
            if v == target {
                return i
            }
        }
        return -1 
    }
    
    src := gopacket.NewPacketSource(handle, handle.LinkType())
    var ticker *time.Ticker

    posIdx := -1
    fmt.Println("Listening for port knock...")
    for packet := range src.Packets() {

        if posIdx != -1{
            select{
            case _ = <-ticker.C:
                fmt.Println("Time over. Backtracking Port Konck.")
                posIdx = -1
                continue
            default:
            }
        }

        tcpLayer := packet.Layer(layers.LayerTypeTCP)
        if tcpLayer == nil{
            continue
        }
        tcp := tcpLayer.(*layers.TCP)

        idx := indexOf(int(tcp.DstPort), ports)
        if posIdx == -1 && idx == 0{
            posIdx = 0
            ticker = time.NewTicker(30 * time.Second) 
            fmt.Println(int(tcp.DstPort))
        } else if posIdx >= 0 && (idx - posIdx) == 1{
            posIdx++ 
            fmt.Println(int(tcp.DstPort))
        } 
        if posIdx == len(ports) - 1{
            fmt.Println("port knock complete...")
            ticker.Stop() 
            break 
        }
    }
	return nil 
}


