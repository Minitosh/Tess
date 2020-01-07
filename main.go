package main

import (
	"Tess/drive"
	"Tess/gmail"
)

func main() {
	gmail.GetFromGmail()
	drive.SendToDrive()
}
