package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	smsAPIUrl   = ""
	bearerToken = ""
)

type SMSRequest struct {
	SMS struct {
		Text     string `json:"text"`
		Phone    string `json:"phone"`
		Priority int    `json:"priority"`
	} `json:"sms"`
}

type SMSResponse struct {
	Message string `json:"message"`
}

func main() {
	err := process()

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Смс отправлены. Результат сохранён в файл")
	endApp()
}

func process() error {
	f, err := openExcelFile("city_persons.xlsx")
	if err != nil {
		return err
	}
	defer f.Close()

	textSms, err := getTextSms(f)
	if err != nil {
		return err
	}

	numbers, err := getPhoneNumbers(f)
	if err != nil {
		return err
	}

	err = processNumbers(numbers, textSms, f)
	if err != nil {
		return err
	}

	if err := saveFile(f); err != nil {
		return err
	}

	return nil
}

func sendSMS(number, text string) (string, error) {
	//Для тестирования на локали отключаю проверку ssl сертификата
	//tr := &http.Transport{
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}
	//
	//client := &http.Client{
	//	Timeout:   10 * time.Second,
	//	Transport: tr,
	//}
	//Для тестирования на локали отключаю проверку ssl сертификата

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	smsReq := SMSRequest{}
	smsReq.SMS.Text = text
	smsReq.SMS.Phone = number
	smsReq.SMS.Priority = 1

	jsonData, err := json.Marshal(smsReq)
	if err != nil {
		return "", fmt.Errorf("ошибка marshalling JSON: %v", err)
	}

	req, err := http.NewRequest("POST", smsAPIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка при чтение ответа: %v", err)
	}
	//Из-за того, что в случаи ошибки проходит объект, а в случаи успеха просто bool, то сперва проверю на код 200
	if resp.StatusCode == http.StatusOK {
		return "OK", nil
	}

	var smsResp SMSResponse
	if err := json.Unmarshal(body, &smsResp); err != nil {
		return "", fmt.Errorf("ошибка unmarshalling ответа: %v", err)
	}

	return "", fmt.Errorf("ошибка API: %s", smsResp.Message)
}

func openExcelFile(filename string) (*excelize.File, error) {
	return excelize.OpenFile(filename)
}

func getTextSms(f *excelize.File) (string, error) {
	return f.GetCellValue("Лист1", "H2")
}

func getPhoneNumbers(f *excelize.File) ([]string, error) {
	rows, err := f.GetCols("Лист1")
	if err != nil {
		return nil, err
	}
	return rows[0], nil
}

func processNumbers(numbers []string, textSms string, f *excelize.File) error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, number := range numbers {
		if number == "phone" || number == "" {
			continue
		}
		wg.Add(1)
		go func(i int, number string) {
			defer wg.Done()
			processNumber(i, number, textSms, f, &mu)
		}(i, number)
	}

	wg.Wait()
	return nil
}

func processNumber(i int, number, textSms string, f *excelize.File, mu *sync.Mutex) {
	result, err := sendSMS(number, textSms)
	if err != nil {
		result = err.Error()
	}

	mu.Lock()
	defer mu.Unlock()

	cell := fmt.Sprintf("B%d", i+1)
	if err := f.SetCellValue("Лист1", cell, result); err != nil {
		fmt.Printf("Ошибка записи в ячейку %s: %v\n", cell, err)
	}
}

func saveFile(f *excelize.File) error {
	return f.Save()
}

func endApp() {
	fmt.Println("Нажмите Enter для завершения программы...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
