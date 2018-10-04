package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/syndtr/goleveldb/leveldb"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var Db *leveldb.DB
var keyState = []byte("STATEKEY")
var symbols = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var codeLenght = 4

type status int

const (
	issued status = 1
	dumped status = 2
)

type Error struct {
	msg string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func main() {
	initDb()

	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/code", getCode)
	e.GET("/info", getFreeCodeCount)
	e.PUT("/code/:code", dumpCode)
	e.PUT("/check/:code", checkCode)

	go func() {
		if err := e.Start(":8080"); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

}

func checkCode(c echo.Context) error {

	code := c.Param("code")
	s, err := getCodeStatus(code)
	if err != nil {
		return c.HTML(http.StatusBadRequest, "Код не найден\n")
	}
	if s == issued {
		return c.HTML(http.StatusOK, "Статус кода "+code+": выдан\n")
	} else if s == dumped {
		return c.HTML(http.StatusOK, "Статус кода "+code+": погашен\n")
	}
	return c.HTML(http.StatusBadRequest, "Неизвестная ошибка\n")
}

func dumpCode(c echo.Context) error {
	code := c.Param("code")
	tr, err := Db.OpenTransaction()
	if err != nil {
		return c.HTML(http.StatusBadRequest, "Ошибка транзакции\n")
	}

	s, err := getCodeStatusTr(code, tr)
	if err != nil {
		tr.Discard()
		return c.HTML(http.StatusBadRequest, "Код не найден\n")
	}
	if s == issued {
		err = tr.Put([]byte(code), []byte(string(dumped)), nil)
		if err != nil {
			tr.Discard()
			return c.HTML(http.StatusBadRequest, "Ошибка транзакции\n")
		} else {
			tr.Commit()
			return c.HTML(http.StatusOK, "Код "+code+" погашен\n")
		}
	} else if s == dumped {
		tr.Discard()
		return c.HTML(http.StatusBadRequest, "Код "+code+" уже погашен\n")
	}
	tr.Discard()
	return c.HTML(http.StatusBadRequest, "Неизвестная ошибка\n")
}

func getCodeStatusTr(code string, tr *leveldb.Transaction) (status, error) {
	data, err := tr.Get([]byte(code), nil)
	if err != nil {
		return 0, err
	}
	return status(int(data[0])), nil
}

func getCodeStatus(code string) (status, error) {
	data, err := Db.Get([]byte(code), nil)
	if err != nil {
		return 0, err
	}
	return status(int(data[0])), nil
}

func getCode(c echo.Context) error {
	code, err := makeCode()
	if err != nil {
		return c.HTML(http.StatusBadRequest, "Ошибка создания кода")
	}
	return c.HTML(http.StatusOK, code)
}

func getFreeCodeCount(c echo.Context) error {
	all := math.Pow(float64(len(symbols)), float64(codeLenght))
	var used float64
	state := getState()

	for _, v := range state {
		if v != 0 {
			if used == 0 {
				used = 1
			}
			used = used * float64(v)
		}
	}
	all = all - used
	res := strconv.FormatFloat(all, 'f', 0, 64)

	return c.HTML(http.StatusOK, res)

}

func initDb() {
	var err error
	Db, err = leveldb.OpenFile("level.Db", nil)
	for err != nil {
		time.Sleep(10 * time.Millisecond)
		Db, err = leveldb.OpenFile("level.Db", nil)
	}

	data, err := json.Marshal([]int{0, 0, 0, 0})
	if err != nil {
		log.Panicf("Ошибка инициализации!\n")
	}

	_, err = Db.Get(keyState, nil)
	if err != nil {

		Db.Put(keyState, data, nil)
	}

}

func makeCode() (string, error) {
	code := make([]rune, codeLenght)
	tr, err := Db.OpenTransaction()
	if err != nil {
		log.Printf("Ошибка транзакции\n")
		tr.Discard()
		return "", err
	}

	state := getStateTr(tr)

	if len(state) == 0 {
		tr.Discard()
		return "", &Error{"Генератор кодов исчерпан!\n"}
	}

	for i, xi := range state {
		code[i] = symbols[xi]
	}

	for i := len(state) - 1; i >= 0; i-- {
		state[i]++
		if state[i] < len(symbols) {
			break
		}
		state[i] = 0
		if i <= 0 {
			state = state[0:0]
			break
		}
	}
	strCode := string(code)
	saveState(state, strCode, tr)
	err = tr.Commit()
	if err != nil {
		return "", err
	}
	return strCode, nil
}

func getStateTr(tr *leveldb.Transaction) []int {
	data, err := tr.Get(keyState, nil)
	if err != nil {
		tr.Discard()
		log.Panicf("Ошибка получения состояния!\n")
	}
	state := make([]int, 4)
	err = json.Unmarshal(data, &state)
	if err != nil {
		tr.Discard()
		log.Panicf("Ошибка получения состояния!\n")
	}
	return state
}

func getState() []int {
	data, err := Db.Get(keyState, nil)
	if err != nil {
		log.Panicf("Ошибка получения состояния!\n")
	}
	state := make([]int, 4)
	err = json.Unmarshal(data, &state)
	if err != nil {
		log.Panicf("Ошибка получения состояния!\n")
	}
	return state
}

func saveState(state []int, key string, tr *leveldb.Transaction) {
	data, err := json.Marshal(state)
	if err != nil {
		tr.Discard()
		log.Panicf("Ошибка сохранения состояния!\n")
	}

	err = tr.Put(keyState, data, nil)
	if err != nil {
		tr.Discard()
		log.Panicf("Ошибка сохранения состояния!\n")
	}
	err = tr.Put([]byte(key), []byte(string(issued)), nil)
	if err != nil {
		tr.Discard()
		log.Panicf("Ошибка сохранения ключа!\n")
	}
}
