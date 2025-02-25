// Copyright 2018-2020 go-m3ua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Command m3ua-server works as M3UA server.
*/
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"time"

	"github.com/wmnsk/go-m3ua/messages/params"

	"github.com/ishidawataru/sctp"
	"github.com/wmnsk/go-m3ua"
)

func serve(conn *m3ua.Conn) {
	defer conn.Close()

	buf := make([]byte, 1500)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			// this indicates the conn is no longer alive. close M3UA conn and wait for INIT again.
			if err == io.EOF {
				log.Printf("Closed M3UA conn with: %s, waiting to come back...", conn.RemoteAddr())
				return
			}
			// this indicates some unexpected error occurred on M3UA conn.
			log.Printf("Error reading from M3UA conn: %s", err)
			return
		}

		log.Printf("Read: %x", buf[:n])
		
		data := "0901030e1b0b12060012040913320003020d12910111042143658709214305b36481b04904000000006c81a7a281a402010030819e020155308198a00a80086161616161616161a17c0407919099881811f3800435353535861c536f6d654e616d655468617449734c6f6e6765725468616e4e696e65880f4141412d5365727665722d4e616d65891d536f6d654e616d655468617449734c6f6e6765725468616e4e696e65328a1d536f6d654e616d655468617449734c6f6e6765725468616e4e696e65338405048e100305850504c0a80101"

		msg, err := hex.DecodeString(data)
		if err != nil {
			continue
		}

		if _, err := conn.Write(msg); err != nil {
			log.Printf("Failed to write M3UA data: %s", err)
		}
	}
}

func main() {
	var (
		addr    = flag.String("addr", "127.0.0.1:2905", "Source IP and Port listen.")
		hbInt   = flag.Duration("hb-interval", 0, "Interval for M3UA BEAT. Put 0 to disable")
		hbTimer = flag.Duration("hb-timer", time.Duration(5*time.Second), "Expiration timer for M3UA BEAT. Ignored when hb-interval is 0")
	)
	flag.Parse()

	// create *Config to be used in M3UA connection
	config := m3ua.NewServerConfig(
		&m3ua.HeartbeatInfo{
			Enabled:  true,
			Interval: *hbInt,
			Timer:    *hbTimer,
		},
		0x22222222,                  // OriginatingPointCode
		0x11111111,                  // DestinationPointCode
		1,                           // AspIdentifier
		params.TrafficModeLoadshare, // TrafficModeType
		0,                           // NetworkAppearance
		0,                           // CorrelationID
		[]uint32{1, 2},              // RoutingContexts
		params.ServiceIndSCCP,       // ServiceIndicator
		0,                           // NetworkIndicator
		0,                           // MessagePriority
		1,                           // SignalingLinkSelection
	)
	// set nil on unnecessary parameters.
	config.AspIdentifier = nil
	config.CorrelationID = nil

	// setup SCTP listener on the specified IPs and Port.
	laddr, err := sctp.ResolveSCTPAddr("sctp", *addr)
	if err != nil {
		log.Fatalf("Failed to resolve SCTP address: %s", err)
	}

	listener, err := m3ua.Listen("m3ua", laddr, config)
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}
	log.Printf("Waiting for connection on: %s", listener.Addr())

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			log.Fatalf("Failed to accept M3UA: %s", err)
		}
		log.Printf("Connected with: %s", conn.RemoteAddr())

		go serve(conn)
	}
}
