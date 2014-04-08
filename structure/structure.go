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

package structure

import (
	"net"
)

type Command struct {
	Op      string
	Channel []string
	Message []byte
}

type NodeStatus struct {
	Status            int
	Op                string
	Channels_count    int
	Pending_msg_count int
	Storage_trend     [100]int
	Subscriber_count  int
	Uptime            int64
}

type ClusterNodeStatus struct {
	Address     string
	Ping_result int
	Last_ping   int64
	Delay       int64
	Node_stats  interface{}
	Conn        net.Conn
}

/////////////////////////////////////////////////
//  The response struct form server to client
/////////////////////////////////////////////////

type SubscribeResponse struct {
	Status  int
	Op      string
	Channel []string
}

func NewSubscribeResponse() SubscribeResponse {
	return SubscribeResponse{
		Status: 1,
		Op:     "subscribe",
	}
}

type SetResponse struct {
	Status int
	Op     string
	Info   string
}

func NewSetResponse() SetResponse {
	return SetResponse{
		Status: 1,
		Op:     "set",
	}
}

type GetResponse struct {
	Status  int
	Op      string
	Channel map[string][]byte
}

func NewGetResponse() GetResponse {
	return GetResponse{
		Status: 1,
		Op:     "get",
	}
}

type ErrorResponse struct {
	Status int
	Op     string
	Info   string
}

func NewErrorResponse() ErrorResponse {
	return ErrorResponse{
		Status: 0,
		Op:     "UNKNOWN",
	}
}

type SyncResponse struct {
	Status  int
	Op      string
	Channel string
	Message []byte
}

func NewSyncResponse() SyncResponse {
	return SyncResponse{
		Status: 1,
		Op:     "sync",
	}
}

/////////////////////////////////////////////////

/////////////////////////////////////////////////
//  The command struct form client to server
/////////////////////////////////////////////////

type StatsRequest struct {
	Op string
	T0 int64
}

func NewStatsRequest() StatsRequest {
	return StatsRequest{
		Op: "stats",
	}
}
