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
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/fangli/msgfiber/structure"
)

func (c *Processor) writeHttpPayload(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Powered-By", "Msgfiber")
	w.Header().Set("Cache-Control", "max-age=0")
	data, _ := json.Marshal(payload)
	w.Write(data)
}

func (c *Processor) CheckPskHeader(w http.ResponseWriter, r *http.Request) error {
	psk := strings.Trim(r.Header.Get("PSK"), " ")
	if psk == string(c.Config.Psk) || len(c.Config.Psk) == 0 {
		return nil
	} else {
		resp := structure.NewErrorResponse()
		resp.Info = "PSK not acceptable, access denied!!!"
		c.writeHttpPayload(w, resp)
		return errors.New("PSK not Acceptable!!!")
	}
}

func (c *Processor) httpStatsHandler(w http.ResponseWriter, r *http.Request) {
	if c.CheckPskHeader(w, r) != nil {
		return
	}
	status := c.genStatsInfo(0)
	c.writeHttpPayload(w, status)
}

func (c *Processor) httpMemStatsHandler(w http.ResponseWriter, r *http.Request) {
	if c.CheckPskHeader(w, r) != nil {
		return
	}
	status := c.getMemStats()
	c.writeHttpPayload(w, status)
}

func (c *Processor) httpClusterStatsHandler(w http.ResponseWriter, r *http.Request) {
	if c.CheckPskHeader(w, r) != nil {
		return
	}
	status := c.genClusterStatsInfo()
	c.writeHttpPayload(w, status)
}

func (c *Processor) httpSetHandler(w http.ResponseWriter, r *http.Request) {
	if c.CheckPskHeader(w, r) != nil {
		return
	}
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", 500)
		}

		channel := strings.Trim(r.Header.Get("CHANNEL"), " ")
		if channel != "" && len(body) > 0 {
			resp := c.set("", channel, body)
			c.writeHttpPayload(w, resp)
		} else {
			http.Error(w, "Missing HTTP header 'CHANNEL' or body data", 500)
		}

	} else {
		http.Error(w, "Not Found", 404)
	}
}

func (c *Processor) serveHttp() {
	go func() {
		http.HandleFunc("/cluster_stats", c.httpClusterStatsHandler)
		http.HandleFunc("/mem_stats", c.httpMemStatsHandler)
		http.HandleFunc("/stats", c.httpStatsHandler)
		http.HandleFunc("/set", c.httpSetHandler)
		log.Println("Starting HTTP service at port", c.Config.HttpListen)
		err := http.ListenAndServe(c.Config.HttpListen, nil)
		log.Fatal("Can't create HTTP server:", err.Error())
	}()
}
