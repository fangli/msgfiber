/*************************************************************************
* This file is a part of msgfiber, A decentralized and distributed message
* synchronization system

* Copyright (C) 2014  Fang Li <surivlee@gmail.com> and Funplus, Inc.
*
* This program is free software; you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation; either version 2 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License along
* with this program; if not, see http://www.gnu.org/licenses/gpl-2.0.html
*************************************************************************/

package parsecfg

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var SYS_VER string
var SYS_BUILD_VER string
var SYS_BUILD_DATE string

type Config struct {
	TcpListen    string
	HttpListen   string
	ConnExpires  time.Duration
	SyncInterval time.Duration
	Dsn          string
	Nodes        []string
	Psk          []byte
}

func Parse() Config {
	var cfg Config
	tcplisten := flag.String("tcp-listen", "0.0.0.0:3264", "Which interface and port we'll listen to for internal TCP connnections")
	httplisten := flag.String("http-listen", "0.0.0.0:3265", "Which interface and port we'll listen to for HTTP connections")
	expires := flag.String("conn-expires", "10s", "The timeout and expires of connections from clients, unit could be s, m, h or d")
	dsn := flag.String("store", "root@tcp(localhost:3306)/msgfiber?charset=utf8", "The connection DSN of cache storage")
	syncinterval := flag.String("sync-interval", "1s", "The interval of syncing from DB, unit could be s, m, h or d")
	nodesRaw := flag.String("nodes", "localhost:3264", "All nodes in this msgfiber cluster")
	psk := flag.String("psk", "", "The pre-shared key of this cluster, must be same in a cluster")
	version := flag.Bool("version", false, "Show version information")
	v := flag.Bool("v", false, "Show version information")

	flag.Parse()

	if *version || *v {
		fmt.Println("Msgfiber: A decentralized and distributed message synchronization system")
		fmt.Println("Version", SYS_VER)
		fmt.Println("Build", SYS_BUILD_VER)
		fmt.Println("Compile at", SYS_BUILD_DATE)
		os.Exit(0)
	}

	cfg.TcpListen = *tcplisten
	cfg.HttpListen = *httplisten
	cfg.Psk = []byte(*psk)
	cfg.Dsn = *dsn
	cfg.Nodes = strings.Split(*nodesRaw, ",")

	connDuration, err := time.ParseDuration(*expires)
	if err != nil {
		log.Fatal("Unrecognized parameter for connexpires: ", *expires, ". Use --help to get more information.")
	}
	cfg.ConnExpires = connDuration

	syncDuration, err := time.ParseDuration(*syncinterval)
	if err != nil {
		log.Fatal("Unrecognized parameter for syncinterval: ", *syncinterval, ". Use --help to get more information.")
	}
	cfg.SyncInterval = syncDuration

	return cfg
}
