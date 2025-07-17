# lan-c2
A command and control program operating over a LAN

## Introduction 
This document explains how the command and control program is designed. The command and control program, C2 for short, consists of two entities: commander and victim. The commander controls the victim through a covert channel that uses a combination of networking stenography and normal TCP connection. It is assumed that the victim is already infected with sudo privilege. 

    1. Disconnecting from the victim. 
    2. Uninstalling the malware from the victim machine’s filesystem. 
    3. Starting a keylogger on the victim that saves every keystroke on a file saved on the victim’s filesystem and also displays every keystroke on the commander’s screen.
    4. Stopping the keylogger on the victim. 
    5. Transfering the keylog file from the victim’s filesystem to the commander’s filesystem.
    6. Transferring a binary or a text file from the victim’s filesystem onto the commander’s filesystem. 
    7. Transfering a binary or a text file from the commander’s filesystem onto the victim’s filesystem. 
    8. Watching a single binary or text file on the victim’s filesystem. 
    9. Watching a directory non-recursively on the victim’s filesystem. 
    10. Running Bash Shell commands on the victim’s machine. 
    11. Spoofing the process name of the malware on the victim.
    12. Using port knocking to establish a session between the commander and the victim.

The next subsection explains how the commander performs port knocking on the victim:

### Beacon Protocol
In order to establish a balance between performance and stealthiness, this C2 uses a hybrid approach where one part of communication is performed via a covert channel, and the other part is performed via regular TCP connections. The most critical part of the communication handling synchronization and control between the commander and the victim is performed via covert channels minimizing the probability of it being detected. And the functionality part of the communication, e.g., file transfer, file watching, etc., is performed over TCP to deliver an optimal performance.  


Once port knocking is completed, the commander and the victim communicate critical control and synchronization information via a series of half open TCP connections. In that, the commander sends a TCP SYN packet to a closed port on the victim, and the victim responds by sending a TCP RST+ACK packet. The following figure demonstrates how a half open TCP connection looks like: 
![Half Open TCP Connection](./docs/images/4-half-open-tcp-connection.png)

In the figure above, the TCP SYN packet is sent by the commander, and the TCP RST+ACK packet sent by the victim is not part of a normal TCP connection initiated by Linux’s networking stack, but rather they are raw packets crafted using libpcap library. 

The critical control and synchronization information meant to be exchanged between the commander and the victim are hidden in TCP\IPv4 packets’ fields using steganography. The next subsection explains how critical control and synchronization be hidden is encoded in TCP\IPv4 packets’ fields.

##### Beacon Field Overview
This C2 uses Sequence Number field and Window Size field of TCP in addition to IPv4’s Identification field. TCP’s Sequence Number field consists of thirty-two bits. These thirty-two bits are divided into three sections: Code Section, Options Section, and Port Section. Code Section is four bits long, Options section is twelve bits long, and Port Section is six-teen bits long. The Code Section is meant to represent what kind of operation is about to be performed on the victim, for example, starting the keylogger or watching a text file. Each operation has a corresponding unique number from the 2^4 = 16 available numbers, which can be represented using the four bits of the Code Section. At this time, Options Section has no use and is reserved for future use. Moreover, since the Port Section is six-teen bits long, it is enough to hold Transport Layer’s port numbers. As mentioned earlier, this program uses a combination of normal TCP and a covert channel to facilitate the communication between the command and the victim. The Port Section is used to hold a port number that is used to establish a normal TCP connection between the victim and the commander. Henceforth, this document refers to the Code Section, Options Section, and Port Section collectively as a Beacon Field. To sum up, the following figure demonstrates how the thirty-two bits of a Beacon Field are broken down into Code Section, Options Section, and Port Section:
![Beacon Field Structure](./docs/images/5-beacon-field-structure.png)

As for IPv4’s Identification Field, it is used to hold a pseudo-randomly generated six-teen bit number, which is used to derive an encryption key that will be used to encrypt the value of the Beacon Field. The six-teen bit number that is embedded into IPv4’s Identification Field is given as an input into a mathematical function that transforms the six-teen bit number into a thirty-two bit number. This mathematical function uses a series of predefined and deterministic mathematical operations to transform the six-teen bit number into a thirty-two bit number. Henceforth, this document will refer to the mathematical function as the Scramble Function and the pseudo-randomly generated six-teen bit number as the Key. The thirty-two bit number returned by the Scramble Function is then XORed with the value of the Beacon Field. 

##### Port Knocking
The commander uses port knocking to let the victim know that it wishes to establish a covert channel. The commander sends four TCP SYN packets to ports 6666, 7777, 8888, and 6666 on the victim in the given order. Once the victim receives four UDP packets in the correct order, the victim starts listening for raw SYN TCP packets coming from the commander. The next section explains what “Beacon Protocol” is, and how it is used to facilitate the stenographic communication between the commander and the victim.    

##### TCP/IPv4 Stenography
TCP’s Sequence Number field is set to the encrypted value of the Beacon Field, and the Identification field of IPv4 is set to the Key. In addition, Cyclic Redundancy Check (CRC) of the encrypted Beacon Field is computed and stored in TCP’s Window Size field. The CRC function used must take a thirty-two bit number and return a six-teen bit number. Only the SYN flag is set, and all other fields are filled in as they would when constructing a regular TCP/IPv4 packet, and then the resulting packet is sent on wire. 

Please note, this stenography strategy relies on the fact that most operating systems assign random numbers to TCP’s Sequence Number and IPv4’s Identification. That is, IPv4’s Identification is indeed pseudo-randomly generated, and since the value placed in TCP’s Sequence Number is encrypted, it appears to be random. Therefore, both TCP’s Sequence Number and IPv4’s Identification appear random. 

##### TCP/IPv4 Stenography: Victim’s Side
Once the victim receives the handcrafted TCP/IPv4 packet from the commander, the victim decodes the TCP/IPv4 packet and processes. The victim extracts the value of IPv4’s Identification field and inputs it into the Scramble Function. Then, the victim XORes the thirty-two bit number returned by the Scramble Function with the value of TCP’s Sequence Number in order to decrypt the Beacon Field. Then, the victim breaks down the Beacon Field into Code Section, Options Section, and Port Section. The first four MSB bits belong to the Code Section, and the next twelve bits belong to the Options Section, and the remaining six-teen bits belong to the Port Section.    

The value of the Code section tells the victim what kind of operation the commander wishes to perform, and the value of the Port section tells the victim the port number the commander will use to carry out the desired operation over TCP. 

In response to the commander’s TCP SYN packet the victim crafts a TCP RST+ACK packet. The victim sets the value of its Code Section to the same value as the commander’s Code Section to acknowledge the operation the commander wishes to perform. If the victim sets its Code Section to any other value than the commander’s Code Section, it indicates failure to perform the operation the commander is asking for. Then the victim sets its Options Section to zero as there is no use for the Options Section at the moment. And finally the victim sets its Port Section to the port number the victim will use to perform the operation the commander has asked for. 

However, the encryption process of Beacon Field at the victim’s side is a bit different than that of the commander’s. The victim adds the value of the commander's IPv4 Identification packet with the value of the commander’s Sequence Number and takes the modulus of two to the power of six-teen to drive its Key instead of pseudo-randomly generating the Key. The following is the formula the victim uses to derive its Key: 

(IPv4.Identification:16-bit + TCP.Sequence_Number:32-bit ) % 2^16 = Victim’s Key:32-bit

Then the victim puts its Key into the same Scramble Function used by the commander to derive a thirty-two bit number that will be XORed with the victim’s Beacon Field. The encrypted value of the victim’s Beacon Field is stored in TCP’s Sequence Number, and IPv4’s Identification field is set to the same value as the commander packet’s IPv4 Identification field. The same CRC function is used on the commander’s side to compute the checksum of the encrypted Beacon Field, and the checksum computed is stored in TCP’s Window Size field. Only the RST and ACK flags are set, and all other fields are filled in as they would when constructing a regular TCP/IPv4 packet, and then the resulting packet is sent on wire. 

##### TCP/IPv4 Stenography: Commander’s Side
When Commander receives the victim’s TCP RST+ACK packet, it derives the victim's Key by using the same formula the victim used to drive the victim's Key. The commander inputs the victim's Key into the Scramble Function and XORes the thirty-two bit number returned by the Scramble function with the victim’s Sequence Number. The commander breaks down the victim’s decrypted Beacon Field into the Code Section, Options Section, Port Section. If the victim's Code Section has the same value as the commander’s Code Section, it implies everything went well and the Commander establishes a normal TCP connection with the victim using the ports specified in the commander’s and the victim’s Port Section values. The following figures demonstrate this process:
![Beacon Commander](./docs/images/6-beacon-commander.png)

##### Networking Components 
Both the victim and the commander have one Sender Component, one Receiver Component, and one Controller Component. The Sender Component is responsible for sending packets to the other side. The Receiver Component is responsible for receiving packets from the other side. And the Controller Component is responsible for packet processing logic of the victim or the commander. The Controller Component uses the Receiver Component to get packets from the other side, processes the packets and uses the Sender Component to send response packets to the other side accordingly. Sender Component, Receiver Component, and Controller Component are all run in separate threads concurrently to ensure one component does not impede another component. The following figure gives a high-level picture of how all the aforementioned components work together: 
![Beacon Commander-victim Interaction](./docs/images/7-beacon-commander-victim-interaction.png)

##### High Level Beacon Protocol Logic: Commander
The following figure gives a detailed picture of how Beacon Protocol works from the commander’s side: 
![High-level Beacon Protocol Logic Commander](./docs/images/8-high-level-beacon-protocol-logic-commander.png)


##### High Level Beacon Protocol Logic: Victim
The following figure gives a detailed picture of how Beacon Protocol works from the victim side: 
![High-level Bracon Protocol Victim](./docs/images/9-high-level-beacon-protocol-victim.png)

##### High-level Hybrid Model Example
The following figure gives a high-level example of how an operation can be performed using a combination of Beacon Protocol and regular TCP. The commander asks the victim to perform a file transfer from the commander to the victim, which is performed using the Beacon Protocol (i.e., the asking part is done over Beacon Protocol). And the actual file transfer is done via regular TCP as follows: 
![High-level Hybird Model Example](./docs/images/10-high-level-hybird-model-example.png)





















### User Guide
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

#### Requirements
    • Both the victim machine and the commander machine must be Linux/amd64.
    • Both the victim machine and the commander machine must have libnet-dev and libpcap-dev Linux libraries installed.
    • Go 1.23 and above is required to compile the program.
    • This program only works on Little Endian machines. 
    • Remove all iptables and nftables rules before running the program.
    • Commander and victim must be both connected to a private network.

#### Environment Setup
Run the following commands to prepare environment before compiling and running the program:

sudo apt install libpcap-dev libnet-dev iptables -y
sudo iptables -F
sudo apt install go 
cd ./c2
go mod download

#### Compiling 
Run the following to compile the victim and the commander:

go build commander.go
go build victim.go 

Or alternatively run the provided build.sh to compile both the victim and the commander as follows:

./build 

#### Running Commander
Ensure commander.json and commander executable are both present in the same directory. Run as follows:

sudo ./commander 

#### Running Victim
Ensure victim.json and victim executable are both present in the same directory. Run as follows:

sudo ./victim

#### commander.json and victim.json
As mentioned earlier, commander.json is required to be in the same directory as the command executable and the same rules apply to the victim. The structure of commander.json amd victim.json is as follows: 

{
	"IfaceStr": "<network adaptor to be used>",
	"SrcMACStr": "<MAC address of network adaptor>",
	"DstMACStr": "<MAC address of other side's network adaptor>",
	"SrcIPStr": "<IPv4 address of network adaptor>",
	"DstIPStr": "<IPv4 address of other side's network adaptor>",
	"SrcPortInt": 	"<port number>",
	"DstPortInt": 	"<other side's port number>"
}

#### Running Commander Example
![Example 1](./docs/images/1-example1.png)

#### Running Victim Example
![Example 2](./docs/images/2-example2.png)

#### Commander CLI
Enter ‘h’ or ‘help’ to see the help menu has follows:
![Example 3](./docs/images/3-example3.png)

Enter ‘c’ or ‘clear’ to clear the commander's screen.

If the error message, “parseRSTACK failed: key corrupted” is shown, the commander and the victim are out of synchronisation, and the commander has one or more state packets from the victim in its buffer. Enter ‘f’ or ‘flush’ to flush out the stale packets.







