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

package storemidware

import (
	"bytes"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/fangli/msgfiber/structure"
	_ "github.com/go-sql-driver/mysql"
)

type StoreMidware struct {
	Dsn                string
	SyncInterval       time.Duration
	ChangeNotification chan structure.SyncResponse
	db                 *sql.DB
	msgPool            map[string][]byte
	msgPoolLock        sync.Mutex
	statusCounterLock  sync.Mutex
	SyncPeriod         int64
	LastSync           int64
	SyncLastsLock      sync.Mutex
	dbStatus           [100]int
}

func (s *StoreMidware) Close() error {
	return s.db.Close()
}

func (s *StoreMidware) loadData() (count int) {
	var (
		channel string
		msg     []byte
	)
	rows, err := s.db.Query("SELECT channel, msg FROM msgstore")
	if err != nil {
		log.Fatal("Couldn't restore messages from storage: ", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&channel, &msg)
		if err != nil {
			log.Fatal("Couldn't load message: ", err.Error())
		}
		s.msgPoolLock.Lock()
		s.msgPool[channel] = msg
		s.msgPoolLock.Unlock()
	}
	err = rows.Err()
	if err != nil {
		log.Fatal("Couldn't load message: ", err.Error())
	}

	s.msgPoolLock.Lock()
	cnt := len(s.msgPool)
	s.msgPoolLock.Unlock()

	return cnt
}

func (s *StoreMidware) sync() error {
	var (
		channel string
		msg     []byte
		payload structure.SyncResponse
	)

	payload = structure.NewSyncResponse()

	s.msgPoolLock.Lock()
	defer s.msgPoolLock.Unlock()

	rows, err := s.db.Query("SELECT channel, msg FROM msgstore")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&channel, &msg)
		if err != nil {
			return err
		}

		if !bytes.Equal(s.msgPool[channel], msg) {
			log.Println("Inconsistent Data detect! [", channel, "]:", string(s.msgPool[channel]), "->", string(msg))
			s.msgPool[channel] = msg
			payload.Channel = channel
			payload.Message = msg
			s.ChangeNotification <- payload
		}
	}
	return rows.Err()
}

func (s *StoreMidware) periodicSync() {
	log.Println("Synchronization from storage every", s.SyncInterval)

	var t0 int64
	var t1 int64
	var err error

	for {
		time.Sleep(s.SyncInterval)
		t0 = time.Now().UnixNano()
		err = s.sync()
		t1 = time.Now().UnixNano()
		s.SyncLastsLock.Lock()
		s.SyncPeriod = t1 - t0
		s.LastSync = time.Now().Unix()
		s.SyncLastsLock.Unlock()
		if err == nil {
			s.shiftDbStatus(1)
		} else {
			s.shiftDbStatus(0)
		}
	}
}

func (s *StoreMidware) shiftDbStatus(status int) {
	s.statusCounterLock.Lock()
	defer s.statusCounterLock.Unlock()
	for i := 0; i < len(s.dbStatus)-1; i++ {
		s.dbStatus[i] = s.dbStatus[i+1]
	}
	s.dbStatus[len(s.dbStatus)-1] = status
}

func (s *StoreMidware) Init() bool {
	var err error
	log.Println("Initializing storage", s.Dsn)
	s.msgPool = make(map[string][]byte)

	s.db, err = sql.Open("mysql", s.Dsn)
	if err != nil {
		log.Fatal("Database DSN parameters error: ", err.Error())
	}

	err = s.db.Ping()
	if err != nil {
		log.Fatal("Couldn't connect to database: ", err.Error())
	}

	log.Println("Loading saved data from storage")
	count := s.loadData()
	log.Println(count, "messages loaded")
	go s.periodicSync()
	return true
}

func (s *StoreMidware) Update(channel string, msg []byte) error {
	s.msgPoolLock.Lock()
	defer s.msgPoolLock.Unlock()

	if bytes.Equal(msg, s.msgPool[channel]) {
		return errors.New("No changes found")
	}

	_, err := s.db.Exec("INSERT INTO msgstore(channel, msg) VALUES(?, ?) ON DUPLICATE KEY UPDATE msg=?", channel, msg, msg)
	if err != nil {
		return err
	} else {
		s.msgPool[channel] = msg
		return nil
	}
}

func (s *StoreMidware) UpdateWithoutDb(channel string, msg []byte) error {
	s.msgPoolLock.Lock()
	defer s.msgPoolLock.Unlock()

	if bytes.Equal(msg, s.msgPool[channel]) {
		return errors.New("No changes found")
	} else {
		s.msgPool[channel] = msg
		return nil
	}
}

func (s *StoreMidware) MsgCount() int {
	s.msgPoolLock.Lock()
	defer s.msgPoolLock.Unlock()
	return len(s.msgPool)
}

func (s *StoreMidware) GetSyncPeriod() string {
	s.SyncLastsLock.Lock()
	defer s.SyncLastsLock.Unlock()
	return strconv.FormatInt(s.SyncPeriod/10e6, 10) + "ms"
}

func (s *StoreMidware) GetLastSync() int64 {
	s.SyncLastsLock.Lock()
	defer s.SyncLastsLock.Unlock()
	return s.LastSync
}

func (s *StoreMidware) DbStatus() [100]int {
	s.statusCounterLock.Lock()
	defer s.statusCounterLock.Unlock()
	return s.dbStatus
}

func (s *StoreMidware) Get(channel string) []byte {
	s.msgPoolLock.Lock()
	defer s.msgPoolLock.Unlock()
	if ret, ok := s.msgPool[channel]; ok {
		return ret
	}
	return []byte("")
}
