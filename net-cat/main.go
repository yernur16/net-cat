package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	clients     = make(map[string]net.Conn)
	leaving     = make(chan message)
	connecting  = make(chan message)
	messages    = make(chan message)
	tempHistory = []byte{}
	m           sync.Mutex
)

type message struct {
	text    string
	name    string
	time    string
	address string
}

func main() {
	port := "8989"
	if len(os.Args) > 1 {
		if checkValidPort(os.Args[1]) && len(os.Args) == 2 {
			port = os.Args[1]
		} else {
			fmt.Println("[USAGE]: ./TCPChat $port")
			return
		}
	}

	fmt.Println("Server listening on " + port)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	go sendMessage(messages)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		m.Lock()
		if len(clients) == 10 {
			conn.Write([]byte("\nRoom is full\n"))
			conn.Close()
		} else {
			conn.Write([]byte("Welcome to TCP-Chat!\n"))
			printLogo := printLogo()
			conn.Write([]byte(printLogo))
			go listenConnection(conn)
		}
		m.Unlock()
	}
}

func listenConnection(conn net.Conn) {
	defer conn.Close()
	time := time.Now().Format("2006-01-02 15:04:05")
	name, msgCon, err := getName(conn, time)
	if err != nil {
		return
	}
	checkName(name, conn)
	m.Lock()
	clients[name] = conn
	m.Unlock()

	conn.Write(tempHistory)
	tempHistory = append(tempHistory, msgCon+"\n"...)
	ioutil.WriteFile("history.txt", tempHistory, 0777)

	for {
		conn.Write([]byte("[" + time + "]" + "[" + name + "]" + ": "))
		msg, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			leaving <- newMessage(name+" has left our chat...", conn, name, time)
			m.Lock()
			delete(clients, name)
			m.Unlock()
			log.Println(name, "disconected")
			tempHistory = append(tempHistory, name+" has left our chat...\n"...)
			ioutil.WriteFile("history.txt", tempHistory, 0o777)
			return
		} else if !isValidStr(msg) {
			continue
		} else {
			messages <- newMessage(msg, conn, name, time)

			tempHistory = append(tempHistory, "["+time+"]"+"["+name+"]"+msg...)
			ioutil.WriteFile("history.txt", tempHistory, 0o777)
		}

	}
}

func newMessage(msg string, conn net.Conn, name, time string) message {
	addr := conn.RemoteAddr().String()
	return message{
		text:    msg,
		name:    "[" + name + "]" + ": ",
		time:    "[" + time + "]",
		address: addr,
	}
}

func getName(conn net.Conn, time string) (string, string, error) {
	name := ""

	for name == "" || !checkName(name, conn) || !isValidStr(name) {
		conn.Write([]byte("\n[ENTER YOUR NAME]: "))

		n, _, err := bufio.NewReader(conn).ReadLine()
		if err != nil {
			return "", "", err
		}
		if !isValidStr(string(n)) {
			conn.Write([]byte("\nIncorrect name\n"))
		}
		name = string(n)
		if len(name) > 15 {
			conn.Write([]byte("\nLen of name is too long. Enter the name equal to or less than fifteen symbols\n"))

			name = ""
		}
	}

	log.Println(name + " connected")
	msg := name + " has joined our chat..."
	connecting <- newMessage(msg, conn, name, time)

	return strings.TrimSpace(name), msg, nil
}

func checkName(name string, conn net.Conn) bool {
	for n := range clients {
		if n == name {
			conn.Write([]byte("\nName already taken\n"))
			return false
		}
	}
	return true
}

func sendMessage(message <-chan message) {
	for {
		select {
		case msg := <-connecting:
			m.Lock()
			for name, conn := range clients {
				if msg.address == conn.RemoteAddr().String() {
					continue
				}
				conn.Write([]byte("\n" + msg.text + "\n"))
				conn.Write([]byte(msg.time + "[" + name + "]:"))
			}
			m.Unlock()
		case msg := <-message:
			m.Lock()
			for name, conn := range clients {
				if msg.address == conn.RemoteAddr().String() {
					continue
				}

				conn.Write([]byte("\n" + msg.time + msg.name + msg.text))
				conn.Write([]byte(msg.time + "[" + name + "]:"))

			}
			m.Unlock()
		case msg := <-leaving:
			m.Lock()
			for name, conn := range clients {
				conn.Write([]byte("\n" + msg.text + "\n"))
				conn.Write([]byte(msg.time + "[" + name + "]:"))
			}
			m.Unlock()
		}
	}
}

func printLogo() []byte {
	logo, err := os.ReadFile("./logo.txt")
	if err != nil {
		log.Fatal(err)
	}
	return logo
}

func checkValidPort(arg string) bool {
	if len(arg) != 4 {
		return false
	}
	for _, v := range arg {
		if v < '0' || v > '9' {
			return false
		}
	}
	return true
}

func isValidStr(str string) bool {
	for _, v := range str {
		if v != ' ' && v != '\n' {
			return true
		}
	}
	return false
}
