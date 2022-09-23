package main

import (
	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/internal"
)

func test() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "check message pattern in file",

		Action: func(ctx *cli.Context) error {
			filename := ctx.Args().First()
			tester := internal.NewTester()

			return tester.Test(filename, &internal.Config{
				Pattern:  ctx.String("pattern"),
				Levels:   ctx.StringSlice("level"),
				Excludes: ctx.StringSlice("exclude"),
			})
		},
	}
}
