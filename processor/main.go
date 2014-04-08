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
	"github.com/fangli/msgfiber/clientpool"
	"github.com/fangli/msgfiber/nodepool"
	"github.com/fangli/msgfiber/parsecfg"
	"github.com/fangli/msgfiber/storemidware"
	"github.com/fangli/msgfiber/structure"
	"net"
	"time"
)

type Processor struct {
	Config    parsecfg.Config
	clients   clientpool.Pool
	store     storemidware.StoreMidware
	cluster   nodepool.Pool
	startTime int64
}

func (c *Processor) setMsg(conn net.Conn, channel string, msg []byte) structure.SetResponse {
	resp := structure.NewSetResponse()

	if !c.cluster.AllConnected() {
		resp.Status = 0
		resp.Info = "One or more nodes in this cluster are disconnected. In order to prevent from data inconsistent we rejected your request"
		return resp
	}

	err := c.store.Update(channel, msg)
	if err != nil {
		resp.Status = 0
		resp.Info = err.Error()
	} else {
		resp.Status = 1
		resp.Info = "Channel updated succeed"
	}
	if err == nil {
		// c.cluster.NodeSync(channel, msg)
		c.broadcast(channel, msg, conn)
	}
	return resp
}

func (c *Processor) genStatsInfo() structure.NodeStatus {
	var status structure.NodeStatus
	status.Uptime = time.Now().Unix() - c.startTime
	status.Status = 1
	status.Op = "stats"
	status.Subscriber_count = c.clients.ClientCount()
	status.Pending_msg_count = c.clients.PendingMsgCount()
	status.Channels_count = c.store.MsgCount()
	status.Storage_trend = c.store.DbStatus()
	return status
}

func (c *Processor) genClusterStatsInfo() []structure.ClusterNodeStatus {
	return c.cluster.Stats()
}

func (c *Processor) ServeForever() {

	c.init()

	c.serveHttp()
	c.serveTcp()
}

func (c *Processor) init() {
	c.startTime = time.Now().Unix()

	c.clients = clientpool.Pool{}
	c.clients.Init()

	c.store = storemidware.StoreMidware{
		Dsn:          c.Config.Dsn,
		SyncInterval: c.Config.SyncInterval,
		Callback:     c.broadcast,
	}
	c.store.Init()

	c.cluster = nodepool.Pool{
		Nodes: c.Config.Nodes,
	}
	c.cluster.Init()

}
