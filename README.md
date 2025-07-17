# lan-c2
A command and control program operating over a LAN


#### Directory Organization
The structure of the source directory is as follows: 
./c2/ 
	./c2/commander.go
	./c2/victim.go
	./c2/commander.json
	./c2/victim.json
	./c2/go.mod
	./c2/so.sum
	./c2/build.sh
	./c2/ctrl/
./c2/beacon/beacon.go	
./c2/beacon/cli.go
	./c2/beacon/knock.go
	./c2/beacon/netutil.go
	./c2/beacon/commander_beacon.go
	./c2/beacon/victim_beacon.go
	./c2/util/
		./c2/util/fs.go
		./c2/util/psname.go
		./c2/util/keylogger.go
		./c2/util/rshell.go
		./c2/util/session.go


The following table explains the purpose of each file and directory within the source directory: 

| File/Directory Name | File/Directory Purpose|
|---------------------|-----------------------|
| c2 directory        | c2 directory is the main directory that contains all the source code of the project. |
| commander.go file   | commander.go file implements the logic of the commander. |
| commander.json      | commander.json file must be present in the same directory as the commander when executing the commander. commander.json file provides routing information to help commander send and receive packets to and from the victim. It includes service port number, IPv4 address, MAC address of the commander and the victim. And commander’s network adaptor that has been assigned the IPv4 address of the commander. | 
| victim.json         | victim.json file must be present in the same directory as the victim when executing the victim. victim.json file provides routing information to help commander send and receive packets to and from the commander. It includes service port number, IPv4 address, MAC address of commander and the victim. And the victim’s network adapter that has been assigned the IPv4 address of the victim.|
| victim.go file      | victim.go file implements the logic of the victim. |
| go.mod file         | go.mod file declares the Go module's name and lists all the required external modules, along with their specific versions. go.mod also specifies the Go being used. |
| go.sum file         | go.sum file is a checksum of all external modules being used. Basically, go.sum ensures integrity and security of the module dependencies by storing cryptographic checksums. |
| build.sh file       | A Bash shell script that compiles commander.go and victim.go |
| beacon.go file      | beacon.go file implements the Beacon protocol discussed earlier in this document | 
| cli.go file         | cli.go file implements the command line interface commander uses to interact with the program | 
| knock.go file       | knock.go file handles port knocking functionality. | 
| netutil.go file     | netutil.go implements the necessary components needed for networking including the Sender and Receiver components discussed earlier | 
| commander_beacon.go file | commander_beacon.go file implements the commander’s side of the Beacon protocol.  | 
| victim_beacon.go file | victim_beacon.go file implements the victim’s side of the Beacon protocol | 
| fs.go file | fs.go file handles file transfer and file watching functionalities. | 
| keylogger.go file | keylogger.go file implements the keylogging functionality. | 
| psname.go file | psname.go file handles process name spoofing functionality.  | 
| rshell.go file | rshell.go file implements the remote shell functionality  | 
| session.go file  | session.go file implements the TCP channel that is used to facilitate the file transfer, file/directory watching, keylogging, and remote shell functionalities | 

