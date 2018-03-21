package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"


	"github.com/whypro/dxinkube/cmd/app"
	"github.com/whypro/dxinkube/pkg/version"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

func main() {
	fs := pflag.CommandLine
	zkControllerOptions := app.NewZKControllerOptions()
	zkControllerOptions.AddFlags(fs)
	pflag.Parse()

	// suppress the glog "logging before flag.Parse" error
	glogCommandLine, err := app.GetGlogCommandLine(fs)
	if err != nil {
		die(err)
	}
	if err := flag.CommandLine.Parse(glogCommandLine); err != nil {
		die(err)
	}

	if zkControllerOptions.Version {
		fmt.Printf("dubbox-zk-controller %s\n", version.Get().String())
		os.Exit(0)
	}

	if err := app.Run(zkControllerOptions); err != nil {
		die(err)
	}
}
