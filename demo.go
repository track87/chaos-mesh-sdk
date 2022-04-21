// Package main
// marsdong 2022/4/21
package main

import (
	"context"
	"fmt"

	"github.com/google/martian/log"
	"github.com/track87/chaos-mesh-sdk/sdk"
	v1 "k8s.io/api/core/v1"
)

func main()  {
	cli := sdk.NewClientOrDie()
	events, err := cli.ListEvents(context.TODO(), "aefdc968-570b-466b-b1a3-615792dcfe75",
		v1.EventTypeWarning, "")
	if err != nil {
		log.Errorf(err.Error())
	}
	fmt.Println(len(events))
}
