package downloader

import (
	"fmt"
	"os"
	"os/exec"
)

func Download(setName, url string) error {

	fmt.Println("Downloading set...")

	cmd := exec.Command("scdl", "-l", url, "--name-format", setName, "--path", "data")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
