package main

import (
	"fmt"
	"github.com/fairlyblank/pop3"
)

func fail(err interface{}) {
	if err != nil {
		panic(err)
	}
}

func main() {
	client, err := pop3.DialTLS("pop3.163.com:995")
	fail(err)
	
	err = client.AuthBasic("username", "password")
	fail(err)

	err = client.Noop()
	fail(err)

	num, octs, err := client.Status()
	fail(err)
	fmt.Println(num, octs)

	fds, err := client.ListAll()
	fail(err)
	fmt.Println(fds)

	for i:=0; i<num; i++ {
		j, s, err := client.List(i+1)
		fail(err)
		fmt.Println(j, s)
	}

	bts, err := client.Retrieve(num)
	fail(err)
	fmt.Println(string(bts))

	err = client.Quit()
	fail(err)
	
}
