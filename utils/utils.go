package utils

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Edit(contents *string) error {
	f, err := os.CreateTemp("", "*.md")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	if *contents != "" {
		if _, err := io.Copy(f, strings.NewReader(*contents)); err != nil {
			log.Println(err)
		}
	}
	f.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	b, err := os.ReadFile(f.Name())
	if err != nil {
		return err
	}

	newContents := string(b)
	*contents = newContents
	return nil
}
