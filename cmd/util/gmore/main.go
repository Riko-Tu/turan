package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

const defaultReadLine = 40

func main() {
	var startLine int
	var readLine int
	var filename string
	var err error

	flag.Parse()
	args := flag.Args()
	switch len(args) {
	case 0:
		fmt.Println("Miss file")
		return
	case 1:
		filename = args[0]
		readLine = defaultReadLine
	case 2:
		filename = args[0]
		startLine, err := strconv.Atoi(args[1])
		if err != nil || startLine < 0 {
			fmt.Println("startLine is integers greater than 0")
			return
		}
		readLine = defaultReadLine
	default:
		filename = args[0]
		startLine, err = strconv.Atoi(args[1])
		if err != nil || startLine < 0 {
			fmt.Println("startLine is Integers greater than or equal to 0")
			return
		}
		readLine, err = strconv.Atoi(args[2])
		if err != nil || readLine <= 0 {
			fmt.Println("endLine is integers greater than 0")
			return
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(fmt.Sprintf("No such file:%s", filename))
		return
	}
	defer file.Close()

	var currentLine int = 0
	var content string = ""
	reader := bufio.NewReader(file)
	for currentLine < (startLine + readLine) {
		str, err := reader.ReadString('\n') //读到一个换行就结束
		if err == io.EOF {                  //io.EOF 表示文件的末尾
			break
		}
		if currentLine >= startLine {
			content += str
		}
		currentLine += 1
	}
	fmt.Printf(content)
}
