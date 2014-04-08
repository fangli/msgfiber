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

package nodepool

import (
	"errors"
	"github.com/fangli/msgfiber/structure"
	"github.com/vmihailenco/msgpack"
	"log"
	"net"
	"sync"
	"time"
)

type Pool struct {
	Nodes         []string
	NodesChannels []chan structure.NodeStatus
	Clients       []structure.ClusterNodeStatus
	syncMaplock   sync.Mutex
}

func (n *Pool) setConnMap(i int, conn net.Conn) {
	n.syncMaplock.Lock()
	defer n.syncMaplock.Unlock()

	if n.Clients[i].Conn != nil {
		n.Clients[i].Conn.Close()
	}
	n.Clients[i].Conn = conn
	n.Clients[i].Last_ping = time.Now().Unix()
	if conn != nil {
		n.Clients[i].Ping_result = 1
	} else {
		n.Clients[i].Ping_result = 0
	}
}

func (n *Pool) reconnect(i int) bool {
	conn, err := net.DialTimeout("tcp", n.Nodes[i], time.Second*5)
	if err != nil {
		n.setConnMap(i, nil)
		return false
	}
	n.setConnMap(i, conn)
	return true
}

func (n *Pool) statsPing(i int) error {
	defer n.syncMaplock.Unlock()

	statsRequest := structure.NewStatsRequest()
	statsRequest.T0 = time.Now().Unix()
	payload, _ := msgpack.Marshal(statsRequest)

	n.syncMaplock.Lock()
	if n.Clients[i].Conn == nil {
		return errors.New("No connection established")
	}
	_, err := n.Clients[i].Conn.Write(payload)
	return err
}

func (n *Pool) NodeSync(channel string, msg []byte) {
	defer n.syncMaplock.Unlock()

	cmd := structure.Command{}
	cmd.Op = "sync"
	cmd.Channel = []string{channel}
	cmd.Message = msg

	payload, _ := msgpack.Marshal(cmd)

	n.syncMaplock.Lock()
	for i := 0; i < len(n.Nodes); i++ {
		if n.Clients[i].Conn != nil {
			n.Clients[i].Conn.Write(payload)
		}
	}
}

func (n *Pool) maintainConn(i int) {
	n.reconnect(i)
	for {
		errPing := n.statsPing(i)
		if errPing != nil {
			log.Println(n.Nodes[i], ": ", errPing.Error())
			n.reconnect(i)
		}
		// n.syncMaplock[i].Lock()
		// n.Clients[i].Address = n.Nodes[i]
		// n.Clients[i].Last_ping = time.Now().Unix()
		// if errPing != nil {
		// 	n.Clients[i].Ping_result = 0
		// 	n.Clients[i].Delay = 0
		// 	n.Clients[i].Node_stats = nil
		// } else {
		// 	n.Clients[i].Ping_result = 1
		// 	n.Clients[i].Delay = int64((t2 - t1) / 10e6)
		// 	n.Clients[i].Node_stats = stats
		// }
		// n.syncMaplock[i].Unlock()

		time.Sleep(time.Second)
	}
}

func (n *Pool) makeOutConn() {
	time.Sleep(time.Second)
	for i := 0; i < len(n.Nodes); i++ {
		go n.maintainConn(i)
	}
}

func (n *Pool) AllConnected() bool {
	defer n.syncMaplock.Unlock()
	n.syncMaplock.Lock()
	for _, node := range n.Clients {
		if node.Ping_result == 0 {
			return false
		}
	}
	return true
}

func (n *Pool) Stats() []structure.ClusterNodeStatus {
	return n.Clients
}

func (n *Pool) Init() {
	for i := 0; i < len(n.Nodes); i++ {
		n.NodesChannels = append(n.NodesChannels, make(chan structure.NodeStatus))
		n.Clients = append(n.Clients, structure.ClusterNodeStatus{})
	}
	go n.makeOutConn()
}
