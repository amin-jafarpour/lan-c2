package beacon 

import (
    "fmt"
    "context"
)

type CommanderParamsStruct struct{
    Route                   RouteStruct
    UserSelectionCh         chan UserSelection
    ReceiverResponseCh      chan ReceiverOutputStruct
    SenderFeedCh            chan StenoFieldsStruct
    SenderResponseCh        chan SenderOutputStruct
}

func CommanderSetup(routeInfo RouteInfoStruct)(CommanderParamsStruct, error){
    commanderParams := CommanderParamsStruct{}
    route, err := parseRoute(routeInfo)
    if err != nil{
        return commanderParams, err 
    }
    commanderParams.Route              = route
    commanderParams.UserSelectionCh    =  make(chan UserSelection)
    commanderParams.ReceiverResponseCh =  make(chan ReceiverOutputStruct)
    commanderParams.SenderFeedCh       =  make(chan StenoFieldsStruct)
    commanderParams.SenderResponseCh   =  make(chan SenderOutputStruct)
    go receiver(ReceiverParamsStruct{
        Route: commanderParams.Route, 
        PktOutCh: commanderParams.ReceiverResponseCh,  
        IsCommander: true,   
    })
    go sender(SenderParamsStruct{
        Route: route, 
        PktInCh: commanderParams.SenderFeedCh,
        ResultOutCh: commanderParams.SenderResponseCh,   			  
    })
    return commanderParams, nil 
}

func Commander(commanderParams CommanderParamsStruct, callback CommanderBeaconCallback){
    ctx, cancel := context.WithCancel(context.Background())
    for userSelection := range commanderParams.UserSelectionCh{
        if err := validateCode(userSelection.Code); err != nil{ 
            fmt.Println("DEBUG: validateCode failed")
            fmt.Println(err)
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
        beaconSYN := BeaconStruct{
            Type: userSelection.Code,
            Options: 0,
            Data: portNum,
            IsSYN: true, 
        }
        stenoFieldsSYN := makeStenoFieldsSYN(&beaconSYN)
        senderOutput := trasmitStenoFields(stenoFieldsSYN, commanderParams.SenderFeedCh, commanderParams.SenderResponseCh)
        if senderOutput.IsCriticalErr{
            fmt.Println("sender critical error", senderOutput.Err)
            return 
        }
        if senderOutput.Err != nil{
            fmt.Println("sender error:", senderOutput.Err)
            continue
        }
        receiverOutput := recvStenoFields(commanderParams.ReceiverResponseCh)
        if receiverOutput.IsCriticalErr{
            fmt.Println("receiver critical error", receiverOutput.Err)
            return
        }
        if receiverOutput.Err != nil { 
            fmt.Println("receiver error", receiverOutput.Err)
            continue
        }

        beaconRSTACK, err := parseStenoFieldsRSTACK(&receiverOutput.StenoFields, &stenoFieldsSYN)
        if err != nil{
            fmt.Println("parseStenoFieldsRSTACK error", err)
            continue
        }
        if beaconRSTACK.Type == InvalidPktCode{
            fmt.Println("InvalidPktCode", beaconRSTACK.Type)
            continue
        }
        if beaconRSTACK.Type == CrptdPktCode{
            fmt.Println("CrptdPktCode", beaconRSTACK.Type)
            continue
        }
        if beaconRSTACK.Type != beaconSYN.Type{
            fmt.Printf("beaconRSTACK.Type != beaconSYN.Type, %d != %d\n", 
            beaconRSTACK.Type, beaconSYN.Type)
            continue
        }
     

        fmt.Printf("beaconSYN[Code = %s, Options = %d, Port = %d, IsSYN = %t]\n",
          CodeStrToCodeNum[beaconSYN.Type], beaconSYN.Options,  beaconSYN.Data, beaconSYN.IsSYN)
        fmt.Printf("beaconRSTACK[Code = %s, Options = %d, Port = %d, IsSYN = %t]\n", 
          CodeStrToCodeNum[beaconRSTACK.Type],  beaconRSTACK.Options,  beaconRSTACK.Data, 
        beaconRSTACK.IsSYN)


        if beaconSYN.Type != DiscCode{
            ctx, cancel = context.WithCancel(context.Background())
            go callback(beaconSYN, beaconRSTACK, commanderParams.Route, userSelection, ctx)
        } 
    }
}






