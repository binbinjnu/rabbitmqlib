package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go.slotsdev.info/server-group/gamelib/log"
	"io"
	"os"
	"strconv"
)

func main() {
	//write()
	read()
}

func write() {
	file, err := os.OpenFile("./test.log", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	//及时关闭file句柄
	defer file.Close()
	writer := bufio.NewWriter(file)
	for i := 0; i < 10; i++ {
		b, _ := json.Marshal("line " + strconv.Itoa(i))
		writer.Write(b)
		writer.Write([]byte{'\r', '\n'})
	}
	//Flush将缓存的文件真正写入到文件中
	writer.Flush()
}

func read() {
	file, err := os.OpenFile("./test.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Error("文件打开失败", err)
		return
	}
	reader := bufio.NewReader(file)
	for {
		// ReadLine有坑，默认最多读4096个byte
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if line[len(line)-1] == '\n' {
			drop := 1
			if len(line) > 1 && line[len(line)-2] == '\r' {
				drop = 2
			}
			line = line[:len(line)-drop]
		}

		fmt.Println("line:", line)
		fmt.Println("lineStr:", string(line))
		fmt.Println("len:", len(line))
	}
	// 及时关闭file句柄
	file.Close()
}
