package main

import (
	"encoding/json"
	"fmt"
)

type (
	st struct {
		Name     string
		Age      int
		Password string
	}

	st1 struct {
		S string
		I int
	}
)

func main() {
	s := &st{
		Name:     "zbb",
		Age:      30,
		Password: "haha",
	}
	s1 := &st1{
		S: "abc",
		I: 1,
	}
	jsonData, err := json.Marshal([]interface{}{s, s1, "abc", 1})
	if err != nil {
		return
	}
	fmt.Println("d:", string(jsonData))
	//var m map[string]interface{}
	//m := make(map[string]interface{})
	slice := make([]interface{}, 0)

	json.Unmarshal(jsonData, &slice)
	fmt.Println("m:", slice)

}
