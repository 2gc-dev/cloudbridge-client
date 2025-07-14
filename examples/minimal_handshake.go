package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type HandshakeResult struct {
	Client   int
	Iter     int
	Latency  time.Duration
	Success  bool
	ErrorMsg string
}

func doHandshake(address, token string, clientNum, iter int, wg *sync.WaitGroup, results chan<- HandshakeResult) {
	defer wg.Done()
	for i := 0; i < iter; i++ {
		start := time.Now()
		conn, err := net.Dial("tcp", address)
		if err != nil {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: 0, Success: false, ErrorMsg: fmt.Sprintf("Failed to connect: %v", err)}
			continue
		}
		writer := bufio.NewWriter(conn)
		reader := bufio.NewReader(conn)

		// 1. Отправить hello
		hello := map[string]interface{}{
			"type":    "hello",
			"version": "1.0.0",
			"features": []string{"tls", "jwt", "tunneling"},
		}
		helloData, _ := json.Marshal(hello)
		writer.Write(append(helloData, '\n'))
		writer.Flush()

		// 2. Прочитать hello-ответ
		helloResp, err := reader.ReadString('\n')
		if err != nil {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: time.Since(start), Success: false, ErrorMsg: fmt.Sprintf("Failed to read hello response: %v", err)}
			conn.Close()
			continue
		}

		// 3. Отправить auth
		auth := map[string]interface{}{
			"type":  "auth",
			"token": token,
			"version": "1.0.0",
			"client_info": map[string]interface{}{"os": runtime.GOOS, "arch": runtime.GOARCH},
		}
		// Для теста невалидного токена можно добавить поле или изменить токен
		if os.Getenv("INVALID_TOKEN") == "1" {
			auth["token"] = "invalid-token-value"
		}
		authData, _ := json.Marshal(auth)
		writer.Write(append(authData, '\n'))
		writer.Flush()

		// 4. Прочитать auth_response
		authResp, err := reader.ReadString('\n')
		if err != nil {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: time.Since(start), Success: false, ErrorMsg: fmt.Sprintf("Failed to read auth response: %v", err)}
			conn.Close()
			continue
		}

		if i == 0 && clientNum < 10 {
			fmt.Printf("[CLIENT %d] First handshake: hello: %s\nauth_response: %s\n", clientNum, helloResp, authResp)
		}

		if !((len(authResp) > 0) && (authResp[0] == '{')) {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: time.Since(start), Success: false, ErrorMsg: fmt.Sprintf("Invalid auth response: %s", authResp)}
			conn.Close()
			continue
		}

		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(authResp), &resp); err != nil {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: time.Since(start), Success: false, ErrorMsg: fmt.Sprintf("Invalid JSON in auth response: %s", authResp)}
			conn.Close()
			continue
		}
		status, _ := resp["status"].(string)
		latency := time.Since(start)
		if status == "ok" || status == "success" {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: latency, Success: true}
		} else {
			results <- HandshakeResult{Client: clientNum, Iter: i, Latency: latency, Success: false, ErrorMsg: fmt.Sprintf("Handshake FAIL: %s", authResp)}
		}
		conn.Close()
		// time.Sleep(10 * time.Millisecond)
	}
}

func main() {
	address := "edge.2gc.ru:3456"
	// Увеличьте нагрузку для стресс-теста:
	nClients := 100 // Количество параллельных клиентов
	nIters := 100   // Количество handshake на клиента
	if v := os.Getenv("N_CLIENTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			nClients = n
		}
	}
	if v := os.Getenv("N_ITERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			nIters = n
		}
	}
	token := os.Getenv("RELAY_TEST_TOKEN")
	if token == "" {
		token = "test-token"
	}

	var wg sync.WaitGroup
	results := make(chan HandshakeResult, nClients*nIters)
	start := time.Now()

	for c := 0; c < nClients; c++ {
		wg.Add(1)
		go doHandshake(address, token, c, nIters, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	success, fail := 0, 0
	var latencies []time.Duration
	var minLatency, maxLatency time.Duration
	for res := range results {
		if res.Success {
			latencies = append(latencies, res.Latency)
			if minLatency == 0 || res.Latency < minLatency {
				minLatency = res.Latency
			}
			if res.Latency > maxLatency {
				maxLatency = res.Latency
			}
			success++
		} else {
			fail++
			if fail <= 10 {
				fmt.Printf("[FAIL] Client %d Iter %d: %s\n", res.Client, res.Iter, res.ErrorMsg)
			}
		}
	}
	dur := time.Since(start)
	var avgLatency time.Duration
	if len(latencies) > 0 {
		total := time.Duration(0)
		for _, l := range latencies {
			total += l
		}
		avgLatency = total / time.Duration(len(latencies))
	}
	fmt.Printf("\nTotal: %d clients x %d handshakes = %d\n", nClients, nIters, nClients*nIters)
	fmt.Printf("Success: %d, Fail: %d\n", success, fail)
	fmt.Printf("Elapsed: %s\n", dur)
	fmt.Printf("Latency (ms): min=%v avg=%v max=%v\n", minLatency.Milliseconds(), avgLatency.Milliseconds(), maxLatency.Milliseconds())

	// Для мониторинга сервера используйте top, htop, iostat, iftop, netstat и т.д. параллельно с этим тестом.
	// Например: top -p <pid_relay> или htop, чтобы смотреть CPU/RAM, iftop/netstat для трафика.
} 