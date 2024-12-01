package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	buildinCmd := map[string]bool{"type": true, "exit": true, "echo": true}

	for {
		// Uncomment this block to pass the first stage
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')

		if err != nil {
			fmt.Fprint(os.Stderr, "Error reading input:", err)
			os.Exit(1)
		}

		command = strings.TrimSpace(command)
		cmd_lst := strings.Split(command, " ")

		// fmt.Printf("[%s]\n", strings.Join(cmd_lst, ","))

		switch cmd_lst[0] {
		case "exit":
			if len(cmd_lst) == 1 {
				os.Exit(0)
			}

			nextArg, _, err := nextNonEmptyString(1, cmd_lst)
			if err != nil {
				fmt.Fprintf(os.Stdout, "%s\n", err)
				break
			}

			code, err := strconv.Atoi(nextArg)

			if err != nil {
				os.Exit(1)
			}
			os.Exit(code)
		case "echo":
			fmt.Fprintf(os.Stdout, "%s\n", strings.Join(cmd_lst[1:], " "))
		case "type":
			cmdToCheck, _, err := nextNonEmptyString(1, cmd_lst)
			if err != nil {
				fmt.Fprintf(os.Stdout, "%s\n", err)
				break
			}

			_, exist := buildinCmd[cmdToCheck]
			if exist {
				fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", cmdToCheck)
				break
			}

			path, err := searchFile(os.Getenv("PATH"), cmdToCheck)
			if err == nil {
				fmt.Fprintf(os.Stdout, "%s\n", path)
				break
			}

			fmt.Fprintf(os.Stdout, "%s: not found\n", cmdToCheck)

		default:
			program, err := searchFile(os.Getenv("PATH"), cmd_lst[0])

			if err == nil {
				cmd := exec.Command(program, strings.Join(cmd_lst[1:], " "))
				output, err := cmd.Output()
				if err != nil {
					fmt.Fprintf(os.Stdout, "%s\n", err)
				} else {
					fmt.Fprintf(os.Stdout, "%s\n", string(output))
				}
				break
			}

			fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		}

	}
}

func nextNonEmptyString(start int, arr []string) (string, int, error) {
	if start >= len(arr) {
		return "", 0, fmt.Errorf("Invalid starting index")
	}

	for idx := start; idx < len(arr); idx++ {
		cur := arr[idx]
		if len(cur) > 0 {
			return cur, idx + 1, nil
		}

	}

	return "", 0, fmt.Errorf("No non-empty string found")
}

func searchFile(dirString, filename string) (string, error) {
	dirs := strings.Split(dirString, ":")

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err == nil {
			for _, file := range files {
				if file.Name() == filename {
					return dir + "/" + filename, nil
				}
			}
		}
	}

	return "", fmt.Errorf("File not found")

}
