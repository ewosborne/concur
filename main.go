/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/ewosborne/concur/cmd"
)

var version string

func main() {
	cmd.SetVersionInfo(version)
	cmd.Execute()
}
