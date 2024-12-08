package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type User struct {
	Username string
	Password string
}

type Message struct {
	From    string
	To      string
	Content string
}

var (
	accounts     = make(map[string]User)
	onlineUsers  = make(map[string]net.Conn)
	messageQueue = make(map[string][]Message)
	mu           sync.Mutex
)

func main() {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Fatalf("Ошибка загрузки сертификата: %v", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", ":8443", config)
	if err != nil {
		log.Fatalf("Ошибка создания сервера: %v", err)
	}
	defer listener.Close()

	log.Println("Сервер запущен на порту 8443...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка соединения: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	var currentUser string

	conn.Write([]byte("Добро пожаловать в чат. Зарегистрируйтесь или войдите. Используйте команды /register или /login\n"))

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Соединение с клиентом разорвано: %v", err)
			break
		}

		command := strings.TrimSpace(string(buf[:n]))
		parts := strings.SplitN(command, " ", 2)
		cmd := parts[0]
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}

		switch cmd {
		case "/register":
			args := strings.SplitN(arg, " ", 2)
			if len(args) != 2 {
				conn.Write([]byte("Формат: /register username password\n"))
				continue
			}
			username, password := args[0], args[1]
			registerUser(conn, username, password)

		case "/login":
			args := strings.SplitN(arg, " ", 2)
			if len(args) != 2 {
				conn.Write([]byte("Формат: /login username password\n"))
				continue
			}
			username, password := args[0], args[1]
			if loginUser(conn, username, password) {
				currentUser = username
				sendOfflineMessages(conn, username)
			}

		case "/logout":
			logoutUser(conn, currentUser)
			currentUser = ""

		case "/online":
			listOnlineUsers(conn)

		case "/history":
			viewMessageHistory(conn, currentUser)

		case "/send":
			args := strings.SplitN(arg, " ", 2)
			if len(args) != 2 {
				conn.Write([]byte("Формат: /send username сообщение\n"))
				continue
			}
			sendMessage(conn, currentUser, args[0], args[1])

		default:
			conn.Write([]byte("Неизвестная команда. Доступные команды: /register, /login, /logout, /online, /history, /send\n"))
		}
	}
}

func registerUser(conn net.Conn, username, password string) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := accounts[username]; exists {
		conn.Write([]byte("Пользователь с таким именем уже существует.\n"))
		return
	}

	accounts[username] = User{Username: username, Password: password}
	conn.Write([]byte("Регистрация успешна.\n"))
}

func loginUser(conn net.Conn, username, password string) bool {
	mu.Lock()
	defer mu.Unlock()

	user, exists := accounts[username]
	if !exists || user.Password != password {
		conn.Write([]byte("Неверное имя пользователя или пароль.\n"))
		return false
	}

	onlineUsers[username] = conn
	conn.Write([]byte("Вход выполнен успешно.\n"))
	return true
}

func logoutUser(conn net.Conn, username string) {
	mu.Lock()
	defer mu.Unlock()

	delete(onlineUsers, username)
	conn.Write([]byte("Вы вышли из системы.\n"))
}

func listOnlineUsers(conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()

	conn.Write([]byte("Онлайн пользователи:\n"))
	for user := range onlineUsers {
		conn.Write([]byte(user + "\n"))
	}
}

func sendMessage(conn net.Conn, from, to, content string) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := accounts[to]; !exists {
		conn.Write([]byte("Пользователь не найден.\n"))
		return
	}

	message := Message{From: from, To: to, Content: content}
	if recipientConn, online := onlineUsers[to]; online {
		recipientConn.Write([]byte(fmt.Sprintf("[%s]: %s\n", from, content)))
	} else {
		messageQueue[to] = append(messageQueue[to], message)
	}

	conn.Write([]byte("Сообщение отправлено.\n"))
}

func viewMessageHistory(conn net.Conn, username string) {
	mu.Lock()
	defer mu.Unlock()

	conn.Write([]byte("История сообщений:\n"))
	for _, msg := range messageQueue[username] {
		conn.Write([]byte(fmt.Sprintf("[%s -> %s]: %s\n", msg.From, msg.To, msg.Content)))
	}
}

func sendOfflineMessages(conn net.Conn, username string) {
	mu.Lock()
	defer mu.Unlock()

	if messages, exists := messageQueue[username]; exists {
		for _, msg := range messages {
			conn.Write([]byte(fmt.Sprintf("[OFFLINE %s]: %s\n", msg.From, msg.Content)))
		}
		delete(messageQueue, username)
	}
}

