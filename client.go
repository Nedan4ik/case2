package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"net"
)

func main() {
	config := &tls.Config{InsecureSkipVerify: true}

	conn, err := tls.Dial("tcp", "localhost:8443", config)
	if err != nil {
		log.Fatalf("Ошибка подключения к серверу: %v", err)
	}
	defer conn.Close()

	log.Println("Подключение к серверу установлено.")
	reader := bufio.NewReader(os.Stdin)
	go readMessages(conn)

	for {
		fmt.Print("> ")
		message, _ := reader.ReadString('\n')
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			break
		}
	}
}

func readMessages(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Соединение с сервером разорвано: %v", err)
			break
		}
		fmt.Print(string(buf[:n]))
	}
}

