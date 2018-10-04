package main

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
)

var (
	dbPath     = path.Join(os.TempDir(), "testdb")
	issueStr   = "0101"
	dumpStr    = "0202"
	nfStr      = "1111"
	issuedCode = []byte(issueStr)
	dumpdeCode = []byte(dumpStr)
)

type testingStorage struct {
	storage.Storage
}

func TestCheckCode(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	if Db.Put(issuedCode, []byte(string(issued)), nil) != nil ||
		Db.Put(dumpdeCode, []byte(string(dumped)), nil) != nil {
		t.Errorf("Error of insert test records in Db: %s\n", err)
	}

	e := echo.New()
	req := httptest.NewRequest(echo.PUT, "/check/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues(issueStr)
	checkCode(c)
	if (rec.Code != http.StatusOK) &&
		(rec.Body.String() != "Статус кода "+issueStr+": выдан\n") {
		t.Errorf("Error check issued code: %s\n", rec.Body.String())
	}

	req = httptest.NewRequest(echo.PUT, "/check/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues(nfStr)
	checkCode(c)
	if (rec.Code != http.StatusBadRequest) &&
		(rec.Body.String() != "Код не найден\n") {
		t.Errorf("Error check bad code: %s\n", rec.Body.String())
	}

	req = httptest.NewRequest(echo.PUT, "/check/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues(dumpStr)
	checkCode(c)
	if (rec.Code != http.StatusBadRequest) &&
		(rec.Body.String() != "Статус кода "+dumpStr+": погашен\n") {
		t.Errorf("Error check dumped code: %s\n", rec.Body.String())
	}

}

func TestDumpCode(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	if Db.Put(issuedCode, []byte(string(issued)), nil) != nil ||
		Db.Put(dumpdeCode, []byte(string(dumped)), nil) != nil {
		t.Errorf("Error of insert test records in Db: %s\n", err)
	}

	e := echo.New()
	req := httptest.NewRequest(echo.PUT, "/code/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues(issueStr)
	dumpCode(c)
	if (rec.Code != http.StatusOK) &&
		(rec.Body.String() != "Код "+issueStr+" погашен\n") {
		t.Errorf("Error dump issued code: %s\n", rec.Body.String())
	}

	req = httptest.NewRequest(echo.PUT, "/code/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues(nfStr)
	dumpCode(c)
	if (rec.Code != http.StatusBadRequest) &&
		(rec.Body.String() != "Код не найден\n") {
		t.Errorf("Error dump bad code: %s\n", rec.Body.String())
	}

	req = httptest.NewRequest(echo.PUT, "/code/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues(dumpStr)
	dumpCode(c)
	if (rec.Code != http.StatusBadRequest) &&
		(rec.Body.String() != "Код "+dumpStr+" уже погашен\n") {
		t.Errorf("Error dump dumped code: %s\n", rec.Body.String())
	}

}

func TestGetCodeStatusTr(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	if err != nil {
		t.Errorf("Error open transaction: %s\n", err)
	}

	if Db.Put(issuedCode, []byte(string(issued)), nil) != nil ||
		Db.Put(dumpdeCode, []byte(string(dumped)), nil) != nil {
		t.Errorf("Error of insert test records in Db: %s\n", err)
	}

	tr, err := Db.OpenTransaction()
	status, err := getCodeStatusTr(issueStr, tr)
	if status != issued && err == nil {
		t.Errorf("Error check issued status\n")
	}

	status, err = getCodeStatusTr(dumpStr, tr)
	if status != dumped && err == nil {
		t.Errorf("Error check dumped status\n")
	}

	status, err = getCodeStatusTr(nfStr, tr)
	if status != 0 && err != nil {
		t.Errorf("Error check bad status\n")
	}
	tr.Commit()
}

func TestGetCodeStatus(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	if err != nil {
		t.Errorf("Error open transaction: %s\n", err)
	}

	if Db.Put(issuedCode, []byte(string(issued)), nil) != nil ||
		Db.Put(dumpdeCode, []byte(string(dumped)), nil) != nil {
		t.Errorf("Error of insert test records in Db: %s\n", err)
	}

	status, err := getCodeStatus(issueStr)
	if status != issued && err == nil {
		t.Errorf("Error check issued status\n")
	}

	status, err = getCodeStatus(dumpStr)
	if status != dumped && err == nil {
		t.Errorf("Error check dumped status\n")
	}

	status, err = getCodeStatus(nfStr)
	if status != 0 && err != nil {
		t.Errorf("Error check bad status\n")
	}
}

func TestGetCode(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	initData(t)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/code/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	getCode(c)
	if (rec.Code != http.StatusOK) &&
		(rec.Body.String() != "0000") {
		t.Errorf("Error generate code: %s\n", rec.Body.String())
	}
}

func TestMakeCode(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()
	initData(t)

	code, err := makeCode()

	if err != nil && code != "0000" {
		t.Errorf("Error of create firts code: %s\n", err)
	}

	data, err := json.Marshal([]int{})
	if err != nil {
		t.Errorf("Ошибка инициализации!\n")
	}
	Db.Put(keyState, data, nil)

	code, err = makeCode()
	if err == nil && code != "" {
		t.Errorf("Error more code: %s - %s\n", code, err)
	}
}

func TestGetStateTr(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	initData(t)
	tr, err := Db.OpenTransaction()
	if err != nil {
		t.Errorf("Error open transaction: %s\n", err)
	}

	state := getStateTr(tr)
	tr.Commit()
	for i := range state {
		if state[i] != 0 {
			t.Errorf("Error check state\n")
		}
	}

}

func TestGetState(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	initData(t)

	if err != nil {
		t.Errorf("Error open transaction: %s\n", err)
	}
	state := getState()
	for i := range state {
		if state[i] != 0 {
			t.Errorf("Error check state\n")
		}
	}
}

func TestSaveState(t *testing.T) {
	os.RemoveAll(dbPath)
	stor, err := storage.OpenFile(dbPath, false)
	if err != nil {
		t.Errorf("Error of create storage: %s\n", err)
	}
	tstor := &testingStorage{stor}
	defer tstor.Close()

	Db, err = leveldb.Open(tstor, nil)
	if err != nil {
		t.Errorf("Error of create Db: %s\n", err)
	}
	defer Db.Close()

	tr, err := Db.OpenTransaction()
	if err != nil {
		t.Errorf("Error open transaction: %s\n", err)
	}

	saveState([]int{1, 1, 1, 1}, issueStr, tr)
	tr.Commit()

	data, _ := Db.Get(keyState, nil)

	state := make([]int, 4)
	json.Unmarshal(data, &state)
	for i := range state {
		if state[i] != 1 {
			t.Errorf("Error check state after save\n")
		}
	}
	data, _ = Db.Get([]byte(issueStr), nil)
	if status(int(data[0])) != issued {
		t.Errorf("Error check status after save\n")
	}
}

func initData(t *testing.T) {
	t.Helper()
	data, err := json.Marshal([]int{0, 0, 0, 0})
	if err != nil {
		t.Errorf("Ошибка инициализации: %s\n", err)
	}

	Db.Put(keyState, data, nil)

}
