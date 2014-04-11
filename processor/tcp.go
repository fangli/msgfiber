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

package processor

import (
	"errors"
	"github.com/fangli/msgfiber/nodepool"
	"github.com/fangli/msgfiber/parsecfg"
	"github.com/fangli/msgfiber/storemidware"
	"github.com/fangli/msgfiber/structure"
	"github.com/vmihailenco/msgpack"
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	Name     string
	Channels []string
	Outgoing chan interface{}
	Conn     net.Conn
	Quit     chan bool
}

type Processor struct {
	Config         parsecfg.Config
	store          storemidware.StoreMidware
	nodes          nodepool.Pool
	broadcastChan  chan structure.SyncResponse
	startTime      int64
	ClientList     map[string]*Client
	ClientListLock sync.Mutex
}

func (p *Processor) ContainChannel(client *Client, channel string) bool {
	for _, t := range client.Channels {
		if t == channel {
			return true
		}
	}
	return false
}

func (p *Processor) Remove(client *Client) {
	client.Quit <- true
	client.Quit <- true
	p.ClientListLock.Lock()
	for name, _ := range p.ClientList {
		if name == client.Name {
			delete(p.ClientList, client.Name)
		}
	}
	p.ClientListLock.Unlock()
	client.Conn.Close()
}

func (p *Processor) BroadcastOthers(client *Client, channel string, msg []byte) {
	payload := structure.NewSyncResponse()
	payload.Channel = channel
	payload.Message = msg
	p.ClientListLock.Lock()
	for _, c := range p.ClientList {
		if p.ContainChannel(c, channel) {
			if client.Name != c.Name {
				p.Write(c, payload)
			}
		}
	}
	p.ClientListLock.Unlock()
}

func (p *Processor) Write(client *Client, payload interface{}) {
	client.Outgoing <- payload
}

func (p *Processor) NodesOk() error {
	if p.nodes.AllConnected() {
		return nil
	}
	return errors.New("One or more nodes in this cluster are disconnected. In order to prevent from data inconsistent we rejected your request")
}

func (p *Processor) BroadcastHandler() {
	for {
		msg := <-p.broadcastChan
		p.ClientListLock.Lock()
		for _, c := range p.ClientList {
			if p.ContainChannel(c, msg.Channel) {
				p.Write(c, msg)
			}
		}
		p.ClientListLock.Unlock()
	}
}

func (p *Processor) stats(reqTime int64) interface{} {
	return p.genStatsInfo(reqTime)
}

func (p *Processor) nodesStats() interface{} {
	return p.genClusterStatsInfo()
}

func (p *Processor) get(client *Client) interface{} {
	msgMap := make(map[string][]byte)
	for _, channel := range client.Channels {
		msgMap[channel] = p.store.Get(channel)
	}
	resp := structure.NewGetResponse()
	resp.Channel = msgMap
	return resp
}

func (p *Processor) subscribe(client *Client, channels []string) interface{} {
	client.Channels = channels
	resp := structure.NewSubscribeResponse()
	resp.Channel = channels
	return resp
}

func (p *Processor) set(client *Client, channel string, msg []byte) interface{} {
	resp := structure.NewSetResponse()
	log.Println("CMD SET: Checking NodesOK")
	err := p.NodesOk()
	if err != nil {
		resp.Status = 0
		resp.Info = err.Error()
		return resp
	}
	log.Println("CMD SET: Updating store and DB")
	err = p.store.Update(channel, msg)
	if err != nil {
		resp.Status = 0
		resp.Info = err.Error()
		return resp
	}
	log.Println("CMD SET: Start node sync")
	p.nodes.NodeSync(channel, msg)
	log.Println("CMD SET: Start broadcastothers")
	p.BroadcastOthers(client, channel, msg)

	resp.Status = 1
	resp.Info = "OK"
	return resp
}

func (p *Processor) sync(client *Client, channel string, msg []byte) interface{} {
	log.Println("CMD SYNC: Trying update store")
	if p.store.UpdateWithoutDb(channel, msg) == nil {
		log.Println("CMD SYNC: start node sync")
		p.nodes.NodeSync(channel, msg)
		log.Println("CMD SYNC: start broadcast")
		p.BroadcastOthers(client, channel, msg)
	} else {
		log.Println("CMD SYNC: No need sync to others")
	}
	return nil
}

func (p *Processor) ping() interface{} {
	return structure.NewPingResponse()
}

func (p *Processor) errResponse(client *Client, op string, errNotice string) interface{} {
	resp := structure.NewErrorResponse()
	resp.Info = errNotice
	resp.Op = op
	return resp
}

func (p *Processor) execCommand(client *Client, cmd *structure.Command) interface{} {
	switch cmd.Op {
	case "ping":
		return p.ping()
	case "stats":
		if !(cmd.Reqtime > 0) {
			return p.errResponse(client, "stats", "No Reqtime!")
		}
		return p.stats(cmd.Reqtime)
	case "cluster_stats":
		return p.nodesStats()
	case "get":
		return p.get(client)
	case "subscribe":
		if cmd.Channel == nil {
			return p.errResponse(client, "subscribe", "You must specific valid 'Channel' field for the channels you want to subscribe")
		}
		return p.subscribe(client, cmd.Channel)
	case "set":
		if cmd.Channel == nil || len(cmd.Channel) != 1 {
			return p.errResponse(client, "set", "You must specific valid 'Channel' field for the channels you want to update")
		}
		if cmd.Message == nil {
			return p.errResponse(client, "set", "You must specific valid 'Message' for those channels")
		}
		return p.set(client, cmd.Channel[0], cmd.Message)
	case "sync":
		if cmd.Channel == nil || len(cmd.Channel) != 1 {
			return errors.New("Invalid sync command received, no valid 'Channel'")
		}
		if cmd.Message == nil {
			return errors.New("Invalid sync command received, no valid 'Message'")
		}
		return p.sync(client, cmd.Channel[0], cmd.Message)
	default:
		return p.errResponse(client, "Unknown", "Unrecognized Command")
	}
}

func (p *Processor) ClientSender(client *Client) {
	defer p.Remove(client)
	defer log.Println("Sender exit:", client.Name)
	for {
		select {
		case resp := <-client.Outgoing:
			payload, _ := msgpack.Marshal(resp)
			client.Conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
			_, err := client.Conn.Write(payload)
			if err != nil {
				return
			}
		case <-client.Quit:
			return
		}
	}
}

func (p *Processor) ClientReader(client *Client) {
	defer p.Remove(client)
	defer log.Println("Reader exit:", client.Name)
	var err error
	decoder := msgpack.NewDecoder(client.Conn)
	for {
		cmd := &structure.Command{}
		client.Conn.SetReadDeadline(time.Now().Add(p.Config.ConnExpires))
		err = decoder.Decode(cmd)
		if err != nil {
			return
		}
		result := p.execCommand(client, cmd)
		if result != nil {
			p.Write(client, result)
		}
	}
}

func (p *Processor) ClientHandler(conn net.Conn) {
	name := conn.RemoteAddr().String()
	newClient := &Client{
		Name:     name,
		Outgoing: make(chan interface{}, 100000),
		Channels: []string{},
		Conn:     conn,
		Quit:     make(chan bool, 2),
	}
	p.ClientListLock.Lock()
	p.ClientList[name] = newClient
	p.ClientListLock.Unlock()
	go p.ClientSender(newClient)
	go p.ClientReader(newClient)
}

func (p *Processor) serveTcp() {
	ln, err := net.Listen("tcp", p.Config.TcpListen)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go p.ClientHandler(conn)
	}
}

func (p *Processor) genStatsInfo(reqTime int64) structure.NodeStatus {
	var status structure.NodeStatus
	status.Uptime = time.Now().Unix() - p.startTime
	status.Reqtime = reqTime
	p.ClientListLock.Lock()
	status.Connections = len(p.ClientList)
	p.ClientListLock.Unlock()
	status.Channels_count = p.store.MsgCount()
	status.Storage_trend = p.store.DbStatus()
	return status
}

func (p *Processor) genClusterStatsInfo() interface{} {
	return p.nodes.Stats()
}

func (p *Processor) ServeForever() {
	p.init()
	p.serveHttp()
	p.serveTcp()
}

func (p *Processor) init() {

	p.broadcastChan = make(chan structure.SyncResponse)
	p.ClientList = make(map[string]*Client)

	go p.BroadcastHandler()

	p.startTime = time.Now().Unix()

	p.store = storemidware.StoreMidware{
		Dsn:                p.Config.Dsn,
		SyncInterval:       p.Config.SyncInterval,
		ChangeNotification: p.broadcastChan,
	}
	p.store.Init()

	p.nodes = nodepool.Pool{
		NodeAddrs: p.Config.Nodes,
	}
	p.nodes.Init()

}
