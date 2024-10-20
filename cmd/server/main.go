package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"smssend/internal/files"
	"smssend/internal/files/excel"
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

type SMSData struct {
	Index  int
	Number string
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
	var file files.FileHandler = &excel.Excel{}
	err := file.Open("city_persons.xlsx")
	if err != nil {
		return err
	}
	defer file.Close()

	err = processNumbers(file)
	if err != nil {
		return err
	}

	if err := file.SaveFile(); err != nil {
		return err
	}

	return nil
}

func processNumbers(f files.FileHandler) error {
	var wg sync.WaitGroup

	//использую шаблон многопоточности - Worker Pool
	const numWorkers = 5
	numPhones := make(chan SMSData, numWorkers)

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go processNumber(numPhones, f, &wg)
	}

	defer func() {
		close(numPhones)
		wg.Wait()
	}()

	phoneNumbers, err := f.GetPhoneNumbers()
	if err != nil {
		return err
	}

	for i, number := range phoneNumbers {
		if number == "phone" || number == "" {
			continue
		}
		numPhones <- SMSData{i, number}
	}

	return nil
}

func processNumber(ch <-chan SMSData, f files.FileHandler, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for data := range ch {
		result, err := sendSMS(ctx, data.Number, f.GetTextSms())
		if err != nil {
			result = err.Error()
		}

		cell := fmt.Sprintf("B%d", data.Index+1)
		f.SetCellValue(cell, result)
	}
}

func sendSMS(ctx context.Context, number, text string) (string, error) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", smsAPIUrl, bytes.NewBuffer(jsonData))
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

func endApp() {
	fmt.Println("Нажмите Enter для завершения программы...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
