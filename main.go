package main

import (
	"bufio"
	"crypto/md5"
	"crypto/rand"
	"encoding/base32"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const dialTimeout = 60
const joinAdHocTimout = 60
const findMACTimeout = 60

func main() {

	if len(os.Args) == 1 {
		printUsage()
		return
	}
	var outFile, inFile, peer string
	var port int
	flag.StringVar(&outFile, "send", "", "File to be sent.")
	flag.StringVar(&inFile, "receive", "", "Destination path of file to be received.")
	flag.IntVar(&port, "port", 3290, "TCP port to use (must match on both ends).")
	flag.StringVar(&peer, "peer", "", "Use \"-peer mac\" or \"-peer windows\" to match the other computer.")
	flag.Parse()

	receiveChan := make(chan bool)
	sendChan := make(chan bool)

	if peer == "" || (peer != "mac" && peer != "windows") {
		log.Fatal("Must choose [ -peer mac ] or [ -peer windows ].")
	}
	t := Transfer{
		Port:      port,
		Peer:      peer,
		AdHocChan: make(chan bool),
	}
	var n Network

	// sending
	if outFile != "" && inFile == "" {
		t.Passphrase = getPassword()
		pwBytes := md5.Sum([]byte(t.Passphrase))
		prefix := pwBytes[:3]
		t.SSID = fmt.Sprintf("flyingCarpet_%x", prefix)
		t.Filepath = outFile

		if runtime.GOOS == "windows" {
			w := WindowsNetwork{Mode: "sending"}
			w.PreviousSSID = w.getCurrentWifi()
			n = w
		} else if runtime.GOOS == "darwin" {
			n = MacNetwork{Mode: "sending"}
		}
		n.connectToPeer(&t)

		if connected := t.sendFile(sendChan, n); connected == false {
			fmt.Println("Could not establish TCP connection with peer")
			return
		}
		<-sendChan
		fmt.Println("Send complete, resetting WiFi and exiting.")

		//receiving
	} else if inFile != "" && outFile == "" {
		t.Passphrase = generatePassword(8)
		pwBytes := md5.Sum([]byte(t.Passphrase))
		prefix := pwBytes[:3]
		t.SSID = fmt.Sprintf("flyingCarpet_%x", prefix)
		fmt.Printf("=============================\n"+
			"Transfer password: %s\nPlease use this password on sending end when prompted to start transfer.\n"+
			"=============================\n", t.Passphrase)

		if runtime.GOOS == "windows" {
			n = WindowsNetwork{Mode: "receiving"}
		} else if runtime.GOOS == "darwin" {
			n = MacNetwork{Mode: "receiving"}
		}
		n.connectToPeer(&t)

		t.Filepath = inFile
		go t.receiveFile(receiveChan, n)
		// wait for listener to be up
		<-receiveChan
		// wait for reception to finish
		<-receiveChan
		fmt.Println("Reception complete, resetting WiFi and exiting.")
	} else {
		printUsage()
		return
	}
	n.resetWifi(&t)
}

func (t *Transfer) receiveFile(receiveChan chan bool, n Network) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(t.Port))
	if err != nil {
		n.teardown(t)
		log.Fatal("Could not listen on :", t.Port)
	}
	fmt.Println("Listening on", ":"+strconv.Itoa(t.Port))
	receiveChan <- true
	for {
		conn, err := ln.Accept()
		if err != nil {
			n.teardown(t)
			log.Fatal("Error accepting connection on :", t.Port)
		}
		t.Conn = conn
		fmt.Println("Connection accepted")
		go t.receiveAndAssemble(receiveChan, n)
	}
}

func (t *Transfer) sendFile(sendChan chan bool, n Network) bool {
	var conn net.Conn
	var err error
	for i := 0; i < dialTimeout; i++ {
		err = nil
		conn, err = net.DialTimeout("tcp", t.RecipientIP+":"+strconv.Itoa(t.Port), time.Millisecond*10)
		if err != nil {
			fmt.Printf("\rFailed connection %2d to %s, retrying.", i, t.RecipientIP)
			time.Sleep(time.Second * 1)
			continue
		} else {
			fmt.Printf("\n")
			t.Conn = conn
			go t.chunkAndSend(sendChan, n)
			return true
		}
	}
	fmt.Printf("Waited %d seconds, no connection. Exiting.", dialTimeout)
	return false
}

func generatePassword(length int) string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return base32.StdEncoding.EncodeToString(randomBytes)[:length]
}

func getPassword() (pw string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password from receiving end: ")
	pw, err := reader.ReadString('\n')
	if err != nil {
		panic("Error getting password.")
	}
	pw = strings.TrimSpace(pw)
	return
}

func printUsage() {
	fmt.Println("\nUsage (Windows): flyingcarpet.exe -send ./picture.jpg -peer mac")
	fmt.Println("[Enter password from receiving end.]\n")
	fmt.Println("Usage (Mac): ./flyingcarpet -receive ./newpicture.jpg -peer windows")
	fmt.Println("[Enter password into sending end.]\n")
	return
}

type Transfer struct {
	Filepath    string
	Passphrase  string
	SSID        string
	Conn        net.Conn
	Port        int
	RecipientIP string
	Peer        string
	AdHocChan   chan bool
}

type Network interface {
	connectToPeer(*Transfer)
	getCurrentWifi() string
	resetWifi(*Transfer)
	teardown(*Transfer)
}

type WindowsNetwork struct {
	Mode         string // sending or receiving
	PreviousSSID string
}

type MacNetwork struct {
	Mode string // sending or receiving
}
