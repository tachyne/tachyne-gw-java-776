// Command mcping performs a Minecraft server-list status ping — the
// operational probe for a deployed gateway (and any other Java server).
//
//	go run ./cmd/mcping -addr <server-ip>:25565 [-proto 776]
//
// Prints the status JSON and the ping round-trip time; exits non-zero if the
// exchange fails, so it can double as a smoke test in scripts.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/tachyne/tachyne-common/protocol"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:25565", "server address")
	proto := flag.Int("proto", 776, "protocol version to announce")
	name := flag.String("login", "", "attempt a login with this player name and print the outcome")
	flag.Parse()

	var err error
	if *name != "" {
		err = login(*addr, int32(*proto), *name)
	} else {
		err = ping(*addr, int32(*proto))
	}
	if err != nil {
		log.Fatal(err)
	}
}

// login performs handshake + Login Start and prints whatever the server
// answers (for the gateway today: always a Login Disconnect whose reason
// tells the story — version gate, access deny, or under-construction).
func login(addr string, proto int32, name string) error {
	c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	br := bufio.NewReader(c)

	host, _, _ := net.SplitHostPort(addr)
	body := protocol.AppendVarInt(nil, proto)
	body = protocol.AppendString(body, host)
	body = protocol.AppendU16(body, 25565)
	body = protocol.AppendVarInt(body, 2) // intent: login
	if err := protocol.WritePacket(c, 0x00, body); err != nil {
		return err
	}
	start := protocol.AppendString(nil, name)
	start = append(start, make([]byte, 16)...) // uuid: server derives its own
	if err := protocol.WritePacket(c, 0x00, start); err != nil {
		return err
	}
	pkt, err := protocol.ReadPacket(br)
	if err != nil {
		return fmt.Errorf("login response: %w", err)
	}
	if pkt.ID != 0x00 { // not a Login Disconnect
		return fmt.Errorf("unexpected login packet %#x", pkt.ID)
	}
	reason, err := protocol.ReadString(pkt.Body())
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "disconnect: %s\n", reason)
	return nil
}

func ping(addr string, proto int32) error {
	c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(10 * time.Second))
	br := bufio.NewReader(c)

	host, _, _ := net.SplitHostPort(addr)
	body := protocol.AppendVarInt(nil, proto)
	body = protocol.AppendString(body, host)
	body = protocol.AppendU16(body, 25565)
	body = protocol.AppendVarInt(body, 1) // intent: status
	if err := protocol.WritePacket(c, 0x00, body); err != nil {
		return err
	}
	if err := protocol.WritePacket(c, 0x00, nil); err != nil { // status request
		return err
	}
	pkt, err := protocol.ReadPacket(br)
	if err != nil {
		return fmt.Errorf("status response: %w", err)
	}
	if pkt.ID != 0x00 {
		return fmt.Errorf("status response: unexpected packet %#x", pkt.ID)
	}
	payload, err := protocol.ReadString(pkt.Body())
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, payload)

	start := time.Now()
	if err := protocol.WritePacket(c, 0x01, make([]byte, 8)); err != nil { // ping
		return err
	}
	pkt, err = protocol.ReadPacket(br)
	if err != nil {
		return fmt.Errorf("pong: %w", err)
	}
	if pkt.ID != 0x01 || len(pkt.Data) != 8 {
		return fmt.Errorf("pong: id=%#x len=%d", pkt.ID, len(pkt.Data))
	}
	fmt.Fprintf(os.Stderr, "ping: %s\n", time.Since(start).Round(time.Microsecond))
	return nil
}
