package toolkit

import (
	"fmt"
	"github.com/bwmarrin/snowflake"
)

var snowNode *snowflake.Node

func init() {
	var err error
	snowNode, err = snowflake.NewNode(0)
	if err != nil {
		panic(err)
	}
}

func UniqueID() string {
	return fmt.Sprintf("%d", snowNode.Generate())
}
