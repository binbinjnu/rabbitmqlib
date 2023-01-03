package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"runtime"
)

func main() {
	//_, err := IntFromInt64(math.MaxInt32 + 1)
	//_, err := IntFromInt64(0)
	//if err != nil {
	//	fmt.Println("err is ", err)
	//}
	err := cover(0)
	fmt.Println("err is ", err)
}

func cover(i int) (err error) {
	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			log.Println("callback err: ", recoverErr)
			err = errors.New("callback err")
		}
	}()

	fmt.Println("d is ", 100/i)
	if i < 0 {
		err1 := errors.New("too small")
		return err1
	}
	return nil
}

func ConvertInt64ToInt(i64 int64) int {
	if math.MinInt32 <= i64 && i64 <= math.MaxInt32 {
		fmt.Println("i:", 1/i64)
		return int(i64)
	}
	panic("can't covert int64 to int")
}

func IntFromInt64(i64 int64) (i int, err error) {
	//defer func() {
	//	if err2 := recover(); err2 != nil {
	//		i = 0                   //这里
	//		err = errors.New("ttt") //这里
	//	}
	//}()

	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			i = 0
			switch recoverErr.(type) {
			case runtime.Error:
				err = recoverErr.(runtime.Error)
			case string:
				err = errors.New(recoverErr.(string))
			default:
				err = recoverErr.(error)
			}
		}
	}()

	i = ConvertInt64ToInt(i64)
	return i, nil
}
