package beacon 

import (
    "net"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"
    "errors"
    "fmt"
    "time"
    "math/rand"
)

const (
    Max16BitNum = (1 << 16) - 1
	MaxRetransmissionCount = 5
	TimeoutSecSYN = 5 
	TimeoutSecRSTACK = 2 
)

type RouteInfoStruct struct{
    IfaceStr 	string `json:"IfaceStr"`
    SrcMACStr 	string `json:"SrcMACStr"`
    DstMACStr 	string `json:"DstMACStr"`
    SrcIPStr 	string `json:"SrcIPStr"`
    DstIPStr 	string `json:"DstIPStr"`
    SrcPortInt 	uint16 `json:"SrcPortInt"`
    DstPortInt 	uint16 `json:"DstPortInt"`
}

type RouteStruct struct{
    Iface   *net.Interface
    SrcMAC 	 net.HardwareAddr
    DstMAC 	 net.HardwareAddr
    SrcIP 	 net.IP
    DstIP 	 net.IP
    SrcPort  layers.TCPPort
    DstPort  layers.TCPPort
}

type ReceiverOutputStruct struct{
    Err 			error 
    StenoFields 	StenoFieldsStruct
    IsCriticalErr   bool
}

type SenderOutputStruct struct{
    Err             error 
    IsCriticalErr   bool
}

type ReceiverParamsStruct struct{
    Route		RouteStruct
    PktOutCh 	chan<- ReceiverOutputStruct
    IsCommander bool 
}

type SenderParamsStruct struct{
    Route 		RouteStruct
    PktInCh 	    <-chan  StenoFieldsStruct
	ResultOutCh	    chan<- SenderOutputStruct   			  
}

func parseRoute(routeInfo RouteInfoStruct) (RouteStruct, error){
	var err error 
    route := RouteStruct{}
	if route.Iface, err = net.InterfaceByName(routeInfo.IfaceStr); err != nil{
		return route, err 
	} 
    if route.SrcMAC, err = net.ParseMAC(routeInfo.SrcMACStr); err != nil{
        return route, err 
    }
    if route.DstMAC, err = net.ParseMAC(routeInfo.DstMACStr); err != nil{
        return route, err
    }
    if route.SrcIP = net.ParseIP(routeInfo.SrcIPStr); route.SrcIP == nil{
        return route, errors.New("Failed to parse src IP")
    }
    if route.DstIP = net.ParseIP(routeInfo.DstIPStr); route.DstIP == nil{
        return route, errors.New("Failed to parse dst IP") 
    }
	if routeInfo.SrcPortInt < 1024 {
		return route, errors.New("src port has to be greater than 1024") 
	}
	if routeInfo.DstPortInt < 1024 {
		return route, errors.New("dst port has to be greater than 1024")
	}
	route.SrcPort = layers.TCPPort(routeInfo.SrcPortInt)
    route.DstPort = layers.TCPPort(routeInfo.DstPortInt)
    return route, nil 
}

func receiver(params ReceiverParamsStruct){
    receiverOutput := ReceiverOutputStruct{}
    handle, err := pcap.OpenLive(params.Route.Iface.Name, 65535, false, pcap.BlockForever)
    if err != nil {
        receiverOutput.Err = err 
        receiverOutput.IsCriticalErr = true 
        params.PktOutCh <- receiverOutput
        return 
    }
    defer handle.Close()
    // tcp[13] = offset of TCP flags, 0x02 SYN only and 0x14 RST+ACK only.
    part := "(tcp[13] = 0x2)"
    if params.IsCommander{
        part = "(tcp[13] = 0x14) and tcp[8:4] != 0"
    }
    filter := fmt.Sprintf(
            "ip and tcp and " +
            "src host %s and dst host %s and " +
            "src port %d and dst port %d and " +
            "ip[4:2] != 0 and " + 
            part, 
            params.Route.DstIP, params.Route.SrcIP, params.Route.DstPort, params.Route.SrcPort,
    )
    
    if err := handle.SetBPFFilter(filter); err != nil {
        receiverOutput.Err = err 
        receiverOutput.IsCriticalErr = true
        params.PktOutCh <- receiverOutput
        return 
    }
    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
    for packet := range packetSource.Packets() { 
        if errorLayer := packet.ErrorLayer(); errorLayer != nil {
            receiverOutput.Err = errorLayer.Error() 
            params.PktOutCh <- receiverOutput
            continue 
        } 
		if ipv4, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4); !ok{
            receiverOutput.Err = errors.New("Failed to fetch IPv4 layer.") 
            params.PktOutCh <- receiverOutput
            continue 
		} else{
            receiverOutput.StenoFields.Key = ipv4.Id
        }
		if tcp, ok := packet.Layer(layers.LayerTypeTCP).(*layers.TCP); !ok{
            receiverOutput.Err = errors.New("Failed to fetch TCP layer.")
            params.PktOutCh <- receiverOutput
            continue 
		} else{
            receiverOutput.StenoFields.CRC = tcp.Window
            receiverOutput.StenoFields.EncryptedSeqNum = tcp.Seq
            if tcp.SYN{
                receiverOutput.StenoFields.IsSYN = true 
            } else if tcp.RST && tcp.ACK {
                receiverOutput.StenoFields.IsSYN = false 
                receiverOutput.StenoFields.AckNum = tcp.Ack 
            } else {
                receiverOutput.Err = errors.New("Not RST+ACK or SYN")
                params.PktOutCh <- receiverOutput
                continue  
            }
            params.PktOutCh <- receiverOutput
        }
    }
}

func sender(params SenderParamsStruct){
    var senderOutput SenderOutputStruct
    handle, err := pcap.OpenLive(params.Route.Iface.Name, 65535, false, pcap.BlockForever)
    if err != nil {
        senderOutput.Err = err 
        senderOutput.IsCriticalErr = true
        params.ResultOutCh <- senderOutput
        return 
    }
    defer handle.Close()
    for stenoFields := range params.PktInCh {
        eth := layers.Ethernet{
            SrcMAC:       params.Route.SrcMAC,
            DstMAC:       params.Route.DstMAC,
            EthernetType: layers.EthernetTypeIPv4,
        }
        ip := layers.IPv4{
            Version: 4, IHL: 5, TTL: 64,
            Protocol: layers.IPProtocolTCP,
            SrcIP:    params.Route.SrcIP,
            DstIP:    params.Route.DstIP,
            Id:       stenoFields.Key,
            Flags:    layers.IPv4DontFragment, // $CHANGE: Does it raise suspicion.
        }
        tcp := layers.TCP{
            SrcPort: params.Route.SrcPort,
            DstPort: params.Route.DstPort,
            Seq: stenoFields.EncryptedSeqNum,
            Ack: stenoFields.AckNum, 
            SYN: stenoFields.IsSYN,
            ACK: !stenoFields.IsSYN, 
            RST: !stenoFields.IsSYN,
            Window:  stenoFields.CRC, 
        }
        tcp.SetNetworkLayerForChecksum(&ip) 
        opts := gopacket.SerializeOptions{
            FixLengths: true, 
            ComputeChecksums: true, 
        }
        serializeBuffer := gopacket.NewSerializeBuffer()
        if err := gopacket.SerializeLayers(serializeBuffer, opts, &eth, &ip, &tcp); err != nil {
            senderOutput.Err = err 
            params.ResultOutCh <- senderOutput
            continue 
        }
        if err := handle.WritePacketData(serializeBuffer.Bytes()); err != nil {
            senderOutput.Err = err 
            params.ResultOutCh <- senderOutput
            continue 
        }
        params.ResultOutCh <- senderOutput
    }  
}

func trasmitStenoFields(fields StenoFieldsStruct, putCh chan<- StenoFieldsStruct, 
    getCh <-chan SenderOutputStruct) SenderOutputStruct{

    senderOutput := SenderOutputStruct{}
    select{
    case putCh <- fields: 
    case <-time.After(2 * time.Second):
        senderOutput.Err = errors.New("timed out putting to sender")
        return senderOutput
    }
    select{
    case senderOutput = <- getCh:
    case <-time.After(2 * time.Second):
        senderOutput.Err = errors.New("timed out getting from sender")
        return senderOutput 
    }
    return senderOutput
}

func recvStenoFields(getCh <- chan ReceiverOutputStruct) ReceiverOutputStruct{
    receiverOutput := ReceiverOutputStruct{}
    select{
    case receiverOutput = <- getCh:
    case <-time.After(2 * time.Second):
        receiverOutput.Err = errors.New("timed out getting from receiver")
        return receiverOutput 
    }
    return receiverOutput 
}

func randPort() (uint16, error) {
    const minPort int = 1024
    const maxPort int = 65535
    const maxAttemptsCount int = 50
    rand.Seed(time.Now().UnixNano())
    for i := 0; i < maxAttemptsCount; i++ {
        port := uint16(rand.Intn(maxPort-minPort+1) + minPort)
        addr := fmt.Sprintf(":%d", port)
        ln, err := net.Listen("tcp", addr)
        if err != nil {
            continue
        }
        ln.Close()
        return port, nil
    }
    return 0, errors.New("could not find an available TCP port")
}
