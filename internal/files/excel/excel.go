package excel

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"sync"
)

type Excel struct {
	file    *excelize.File
	textSms string
	mu      sync.Mutex
}

func (t *Excel) Open(filename string) error {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return err
	}

	t.file = f

	err = t.setTextSms()
	if err != nil {
		return err
	}

	return nil
}

func (t *Excel) setTextSms() error {
	text, err := t.file.GetCellValue("Лист1", "H2")
	if err != nil {
		return nil
	}

	t.textSms = text
	return nil
}

func (t *Excel) GetTextSms() string {
	return t.textSms
}

func (t *Excel) GetPhoneNumbers() ([]string, error) {
	rows, err := t.file.GetCols("Лист1")
	if err != nil {
		return nil, err
	}
	return rows[0], nil
}

func (t *Excel) SetCellValue(cell, value string) {
	t.mu.Lock()
	if err := t.file.SetCellValue("Лист1", cell, value); err != nil {
		fmt.Printf("Ошибка записи в ячейку %s: %v\n", cell, err)
	}
	t.mu.Unlock()
}

func (t *Excel) SaveFile() error {
	return t.file.Save()
}

func (t *Excel) Close() error {
	return t.file.Close()
}
