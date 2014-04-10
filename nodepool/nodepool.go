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
	"strconv"
	"sync"
	"time"
)

type Node struct {
	Name              string
	Conn              net.Conn
	Stats             interface{}
	SuccessfulRetries int64
	FailedRetries     int64
	Delay             string
	NodeLock          sync.Mutex
}

type Pool struct {
	NodeAddrs    []string
	NodeList     map[string]*Node
	NodeListLock sync.Mutex
}

func (n *Node) reset() {
	if n.Conn != nil {
		n.Conn.Close()
		n.Conn = nil
		n.Stats = nil
	}
}

func (n *Node) Receiver() {
	defer n.reset()
	decoder := msgpack.NewDecoder(n.Conn)
	stats := structure.NodeStatus{}
	for {
		n.Conn.SetReadDeadline(time.Now().Add(time.Second * 2))
		err := decoder.Decode(&stats)
		n.NodeLock.Lock()
		if err != nil {
			log.Println("Read thread exited", n.Name, err.Error())
			n.Stats = nil
			n.Delay = "N/A"
			n.NodeLock.Unlock()
			return
		} else {
			n.Stats = stats
			n.Delay = strconv.FormatInt((time.Now().UnixNano()-stats.Reqtime)/10e6, 10) + "ms"
			n.NodeLock.Unlock()
		}

	}
}

func (n *Node) Connect() error {
	conn, err := net.DialTimeout("tcp", n.Name, time.Second*3)

	n.NodeLock.Lock()
	defer n.NodeLock.Unlock()

	if err != nil {
		n.FailedRetries++
		return err
	}
	n.SuccessfulRetries++
	n.reset()

	n.Conn = conn
	go n.Receiver()
	return nil
}

func (n *Node) statsPing() error {
	defer n.NodeLock.Unlock()
	statsRequest := structure.NewStatsRequest()
	payload, _ := msgpack.Marshal(statsRequest)
	n.NodeLock.Lock()
	if n.Conn == nil {
		return errors.New("No connection established")
	}
	n.Conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
	_, err := n.Conn.Write(payload)
	return err
}

func (p *Pool) NodeSync(channel string, msg []byte) {

	cmd := structure.Command{}
	cmd.Op = "sync"
	cmd.Channel = []string{channel}
	cmd.Message = msg

	payload, _ := msgpack.Marshal(cmd)

	for _, node := range p.NodeList {
		node.NodeLock.Lock()
		if node.Conn != nil {
			node.Conn.Write(payload)
		}
		node.NodeLock.Unlock()
	}
}

func (p *Pool) Stats() interface{} {
	ret := make(map[string]*structure.ClusterStatus)
	for name, node := range p.NodeList {
		node.NodeLock.Lock()
		ret[name] = &structure.ClusterStatus{}
		if node.Conn != nil {
			ret[name].Connected = true
			ret[name].Node_stats = node.Stats
		} else {
			ret[name].Connected = false
			ret[name].Node_stats = nil
		}
		ret[name].Delay = node.Delay
		ret[name].SuccessfulRetries = node.SuccessfulRetries
		ret[name].FailedRetries = node.FailedRetries
		node.NodeLock.Unlock()
	}
	return ret
}

func (p *Pool) AllConnected() bool {
	for _, node := range p.NodeList {
		node.NodeLock.Lock()
		if node.Conn == nil {
			node.NodeLock.Unlock()
			return false
		} else {
			node.NodeLock.Unlock()
		}
	}
	return true
}

func (p *Pool) NodeHandler(node *Node) {
	node.Connect()
	for {
		errPing := node.statsPing()
		if errPing != nil {
			log.Println(node.Name, ": ", errPing.Error())
			node.Connect()
		}
		time.Sleep(time.Second)
	}
}

func (p *Pool) makeOutConn() {
	p.NodeListLock.Lock()
	defer p.NodeListLock.Unlock()

	time.Sleep(time.Second)
	for _, n := range p.NodeList {
		go p.NodeHandler(n)
	}
}

func (p *Pool) Init() {
	p.NodeList = make(map[string]*Node)

	for _, addr := range p.NodeAddrs {
		node := &Node{
			Name:     addr,
			Conn:     nil,
			NodeLock: sync.Mutex{},
		}
		p.NodeList[addr] = node
	}

	go p.makeOutConn()
}
