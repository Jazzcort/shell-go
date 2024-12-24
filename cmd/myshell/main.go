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

type OutputMethod int
type Channel int

const (
	Truncate OutputMethod = iota
	Append
)

const (
	None Channel = iota
	Stdout
	Stderr
	StdoutAndStderr
)

func main() {
	buildinCmd := map[string]bool{"type": true, "exit": true, "echo": true, "pwd": true, "cd": true}

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
		cmd_lst, err := stripQuotes(command)
		cmd_args, truncateMap, appendMap, aggregatedChannel, err := filterArgs(cmd_lst[1:])
		if err != nil {
			fmt.Fprintf(os.Stdout, "%s\n", err)
			continue
		}

		outputArr := []string{}
		refArr := []Channel{}

		switch cmd_lst[0] {
		case "exit":
			if len(cmd_args) == 0 {
				executeMap(truncateMap, outputArr, refArr, Truncate)
				executeMap(appendMap, outputArr, refArr, Append)
				os.Exit(0)
			}

			code, err := strconv.Atoi(cmd_args[0])

			if err != nil {
				executeMap(truncateMap, outputArr, refArr, Truncate)
				executeMap(appendMap, outputArr, refArr, Append)
				os.Exit(1)
			}
			executeMap(truncateMap, outputArr, refArr, Truncate)
			executeMap(appendMap, outputArr, refArr, Append)
			os.Exit(code)
		case "echo":
			out := strings.Join(cmd_args, " ") + "\n"
			outputArr = append(outputArr, out)
			refArr = append(refArr, Stdout)

		case "type":
			for _, cmd := range cmd_args {
				_, exist := buildinCmd[cmd]
				if exist {
					outputArr = append(outputArr, fmt.Sprintf("%s is a shell builtin\n", cmd))
					refArr = append(refArr, Stdout)
					continue
				}

				path, err := searchFile(os.Getenv("PATH"), cmd)
				if err == nil {
					outputArr = append(outputArr, fmt.Sprintf("%s\n", path))
					refArr = append(refArr, Stdout)
				} else {
					outputArr = append(outputArr, fmt.Sprintf("%s: not found\n", cmd))
					refArr = append(refArr, Stdout)
				}
			}
		case "pwd":
			if len(cmd_args) != 0 {
				outputArr = append(outputArr, "pwd: too many arguments\n")
				refArr = append(refArr, Stderr)
			} else {
				wd, err := os.Getwd()
				if err != nil {
					fmt.Fprintf(os.Stdout, "Error getting current directory: %s", err)
					break
				}
				outputArr = append(outputArr, fmt.Sprintf("%s\n", wd))
				refArr = append(refArr, Stdout)
			}
		case "cd":
			if len(cmd_args) == 0 {
				homeDir := os.Getenv("HOME")
				err = os.Chdir(homeDir)
				if err != nil {
					outputArr = append(outputArr, fmt.Sprintf("cd: %s: No such file or directory\n", homeDir))
					refArr = append(refArr, Stderr)
				}

			} else if len(cmd_args) == 1 {
				dir := cmd_args[0]
				start := dir[0]
				if start == byte('~') {
					homeDir := os.Getenv("HOME")
					dir = homeDir + dir[1:]
				}
				err = os.Chdir(dir)
				if err != nil {
					outputArr = append(outputArr, fmt.Sprintf("cd: %s: No such file or directory\n", dir))
					refArr = append(refArr, Stderr)
				}

			} else {
				dir := cmd_args[0]
				outputArr = append(outputArr, fmt.Sprintf("cd: string not in pwd: %s\n", dir))
				refArr = append(refArr, Stderr)
			}

		default:
			program, err := searchFile(os.Getenv("PATH"), cmd_lst[0])

			if err == nil {
				switch cmd_lst[0] {
				case "ls":
					if len(cmd_args) == 0 {
						wd, err := os.Getwd()
						if err != nil {
							break
						}
						cmd_args = append(cmd_args, wd)
					}
				}

				for _, cmd_arg := range cmd_args {
					cmd := exec.Command(program, cmd_arg)
					output, err := cmd.CombinedOutput()
					if err != nil {
						outputArr = append(outputArr, string(output))
						refArr = append(refArr, Stderr)

					} else {
						outputArr = append(outputArr, string(output))
						refArr = append(refArr, Stdout)
					}

				}
			} else {
				outputArr = append(outputArr, fmt.Sprintf("%s: command not found\n", cmd_lst[0]))
				refArr = append(refArr, Stderr)
			}

		}

		executeMap(truncateMap, outputArr, refArr, Truncate)
		executeMap(appendMap, outputArr, refArr, Append)

		switch aggregatedChannel {
		case None:
			fmt.Fprintf(os.Stdout, "%s", strings.Join(outputArr, ""))
		case Stdout:
			stderr := strings.Join(filterOutput(outputArr, refArr, Stderr), "")
			fmt.Fprint(os.Stdout, stderr)
		case Stderr:
			stdout := strings.Join(filterOutput(outputArr, refArr, Stdout), "")
			fmt.Fprint(os.Stdout, stdout)
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

func stripQuotes(command string) ([]string, error) {
	runeSlice := []rune(command)
	res, tmp := []string{}, ""
	prevMode, mode := 0, 0

	for curIdx := 0; curIdx < len(runeSlice); curIdx++ {
		switch mode {
		case 0:
			switch runeSlice[curIdx] {
			case '\'':
				mode = 1
			case '"':
				mode = 2
			case '\\':
				prevMode = 0
				mode = 3
			case '>':
				mode = 4

				if len(tmp) == 1 {
					if _, err := strconv.Atoi(tmp); err == nil {
						tmp += ">"
						break
					}
				}

				if len(tmp) != 0 {
					res = append(res, tmp)
				}
				tmp = ">"
			case ' ':
				if len(tmp) != 0 {
					res = append(res, tmp)
					tmp = ""
				}
			default:
				tmp += string(runeSlice[curIdx])
			}
		case 1:
			switch runeSlice[curIdx] {
			case '\'':
				prevMode = 1
				mode = 0
			default:
				tmp += string(runeSlice[curIdx])
			}

		case 2:
			switch runeSlice[curIdx] {
			case '"':
				mode = 0
			case '\\':
				prevMode = 2
				mode = 3
			default:
				tmp += string(runeSlice[curIdx])
			}
		case 3:
			if prevMode == 2 {
				switch cur := runeSlice[curIdx]; cur {
				case '\\':
					tmp += string('\\')
				case '\n':
					tmp += string('\n')
				case '"':
					tmp += string('"')
				case '$':
					tmp += string('$')
				default:
					tmp += "\\" + string(cur)
				}
			} else {
				tmp += string(runeSlice[curIdx])
			}
			mode = prevMode
		case 4:
			mode = 0
			switch cur := runeSlice[curIdx]; cur {
			case '>':
				res = append(res, tmp+">")
				tmp = ""
			case ' ':
				res = append(res, tmp)
				tmp = ""
			default:
				res = append(res, tmp)
				tmp = string(cur)

			}
		default:
			return []string{}, fmt.Errorf("Failed to stripe the command")

		}

	}

	if len(tmp) != 0 {
		res = append(res, tmp)
	}

	return res, nil
}

func filterArgs(cmdLst []string) (newCmdLst []string, truncateMap map[string]Channel, appendMap map[string]Channel, aggregatedChannel Channel, err error) {
	length := len(cmdLst)
	skip := false
	aggregatedChannel = None
	truncateMap = make(map[string]Channel)
	appendMap = make(map[string]Channel)

	for idx, cmd := range cmdLst {
		if skip {
			skip = false
			continue
		}
		switch cmd {
		case ">", "1>":
			if idx+1 >= length {
				err = fmt.Errorf("parse error")
				return newCmdLst, truncateMap, appendMap, aggregatedChannel, err
			}

			if channel, exist := truncateMap[cmdLst[idx+1]]; exist {
				truncateMap[cmdLst[idx+1]] = channel | Stdout
			} else {
				truncateMap[cmdLst[idx+1]] = Stdout
			}

			aggregatedChannel = aggregatedChannel | Stdout
			skip = true
		case "2>":
			if idx+1 >= length {
				err = fmt.Errorf("parse error")
				return newCmdLst, truncateMap, appendMap, aggregatedChannel, err
			}

			if channel, exist := truncateMap[cmdLst[idx+1]]; exist {
				truncateMap[cmdLst[idx+1]] = channel | Stderr
			} else {
				truncateMap[cmdLst[idx+1]] = Stderr
			}

			aggregatedChannel = aggregatedChannel | Stderr
			skip = true
		case ">>", "1>>":
			if idx+1 >= length {
				err = fmt.Errorf("parse error")
				return newCmdLst, truncateMap, appendMap, aggregatedChannel, err
			}

			if channel, exist := appendMap[cmdLst[idx+1]]; exist {
				appendMap[cmdLst[idx+1]] = channel | Stdout
			} else {
				appendMap[cmdLst[idx+1]] = Stdout
			}

			aggregatedChannel = aggregatedChannel | Stdout
			skip = true
		case "2>>":
			if idx+1 >= length {
				err = fmt.Errorf("parse error")
				return newCmdLst, truncateMap, appendMap, aggregatedChannel, err
			}

			if channel, exist := appendMap[cmdLst[idx+1]]; exist {
				appendMap[cmdLst[idx+1]] = channel | Stderr
			} else {
				appendMap[cmdLst[idx+1]] = Stderr
			}

			aggregatedChannel = aggregatedChannel | Stderr
			skip = true
		case "3>", "4>", "5>", "6>", "7>", "8>", "9>", "3>>", "4>>", "5>>", "6>>", "7>>", "8>>", "9>>":
			if idx+1 >= length {
				err = fmt.Errorf("parse error")
				return newCmdLst, truncateMap, appendMap, aggregatedChannel, err
			}
			skip = true
		default:
			newCmdLst = append(newCmdLst, cmd)

		}
	}

	return newCmdLst, truncateMap, appendMap, aggregatedChannel, err
}

func redirectOutput(path string, out string) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(out)
}

func appendOutput(path string, out string) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(out)

}

func executeMap(fileMap map[string]Channel, outputArr []string, refArr []Channel, outputMethod OutputMethod) {
	outputFunc := redirectOutput
	if outputMethod == Append {
		outputFunc = appendOutput
	}

	stdout, stderr, both := "", "", ""
	stdoutFiltered, stderrFiltered, bothFiltered := false, false, false

	for path, channel := range fileMap {
		switch channel {
		case Stdout:
			if !stdoutFiltered {
				stdout = strings.Join(filterOutput(outputArr, refArr, Stdout), "")
				stdoutFiltered = true
			}
			outputFunc(path, stdout)
		case Stderr:
			if !stderrFiltered {
				stderr = strings.Join(filterOutput(outputArr, refArr, Stderr), "")
				stderrFiltered = true
			}
			outputFunc(path, stderr)
		case StdoutAndStderr:
			if !bothFiltered {
				both = strings.Join(filterOutput(outputArr, refArr, StdoutAndStderr), "")
				bothFiltered = true
			}
			outputFunc(path, both)
		}
	}
}

func filterOutput(outputArr []string, refArr []Channel, selectedChannel Channel) (filteredOutput []string) {
	switch selectedChannel {
	case Stdout:
		for idx, output := range outputArr {
			if refArr[idx] == Stdout {
				filteredOutput = append(filteredOutput, output)
			}
		}
	case Stderr:
		for idx, output := range outputArr {
			if refArr[idx] == Stderr {
				filteredOutput = append(filteredOutput, output)
			}
		}
	case StdoutAndStderr:
		filteredOutput = outputArr
	}
	return filteredOutput
}
