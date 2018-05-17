package databoxerrors

import "fmt"

var cliErr []error

const debug = true

func ChkErr(err error) {
	if err == nil {
		return
	}
	if debug == true {
		fmt.Println("[Databox Error] ", err)
	}
	cliErr = append(cliErr, err)
}
