package utils

import (
	"bytes"
	"fmt"
	"red-cloud/mod2"

	"golang.org/x/crypto/ssh"
)

func Gotossh(username string, password string, addr string, cmd string) error {
	//fmt.Println("ssh上去起teamserver!")
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		mod2.PrintOnError(err, "Failed to dial")
		return err
	}
	defer client.Close()

	// 开启一个session，用于执行一个命令
	session, err := client.NewSession()
	if err != nil {
		mod2.PrintOnError(err, "Failed to create session")
		return err
	}
	defer session.Close()

	// 执行命令，并将执行的结果写到 b 中
	var b bytes.Buffer
	session.Stdout = &b

	// 也可以使用 session.CombinedOutput() 整合输出
	if err := session.Run(cmd); err != nil {
		mod2.PrintOnError(err, "Failed to run")
		return err
	}
	fmt.Println(b.String())
	return err
}
