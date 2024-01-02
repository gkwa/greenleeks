package main

import (
	"os"

	"github.com/taylormonacelli/greenleeks"
)

func main() {
	code := greenleeks.Execute()
	os.Exit(code)
}
