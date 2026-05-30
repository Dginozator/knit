// Diagnostic tool to test YDS API connectivity
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	apiKey := os.Getenv("YDS_API_KEY")
	if apiKey == "" {
		fmt.Println("Set YDS_API_KEY environment variable")
		os.Exit(1)
	}

	endpoint := "https://yds.serverless.yandexcloud.net/ru-central1/b1g69tdr85vjfc52uu2l/etnbpg0l0v6jltjp2r8h"
	body := `{"StreamName":"b1g69tdr85vjfc52uu2l/messenger"}`

	req, _ := http.NewRequest("POST", endpoint, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Kinesis_20131202.DescribeStream")
	req.Header.Set("Authorization", "Api-Key "+apiKey)

	// Print the full request
	dump, _ := httputil.DumpRequestOut(req, true)
	fmt.Println("=== REQUEST ===")
	fmt.Println(string(dump))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("=== RESPONSE ===")
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Body:", string(respBody))
}
