/*
Copyright 2020 The Ceph-CSI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"sync"
	"time"

	"github.com/ceph/go-ceph/rados"
)

type connEntry struct {
	conn     *rados.Conn
	lastUsed time.Time
	unique   string
	users    int
}

type ConnPool struct {
	// interval to run the garbage collector
	interval time.Duration
	// timeout for a connEntry to get garbage collected
	expiry time.Duration
	// Timer used to schedule calls to the garbage collector
	timer *time.Timer
	// Mutex for loading and touching connEntry's from the conns Map
	lock *sync.RWMutex
	// all connEntry's in this pool
	// TODO: this does not need to be a sync.Map, there is locking around its usage
	conns *sync.Map
}

// Create a new ConnPool instance and start the garbage collector running
// every @interval.
func NewConnPool(interval, expiry time.Duration) *ConnPool {
	cp := ConnPool{
		interval: interval,
		expiry:   expiry,
		lock:     &sync.RWMutex{},
		conns:    &sync.Map{},
	}
	cp.timer = time.AfterFunc(interval, cp.gc)

	return &cp
}

// loop through all cp.conns and destroy objects that have not been used for cp.expiry.
func (cp *ConnPool) gc() {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	now := time.Now()
	expireUnused := func(key, ce interface{}) bool {
		if ce.(*connEntry).users == 0 && (now.Sub(ce.(*connEntry).lastUsed)) > cp.expiry {
			ce.(*connEntry).destroy()
			cp.conns.Delete(key.(string))
		}
		return true
	}

	cp.conns.Range(expireUnused)

	// schedule the next gc() run
	cp.timer.Reset(cp.interval)
}

// Stop the garbage collector and destroy all connections in the pool.
func (cp *ConnPool) Destroy() {
	// TODO: stop timer, remove all connEntry's from the cp.conns and call
	// conn.Shutdown() to free resources.
	cp.timer.Stop()
	// wait until gc() has finished, in case it is running
	cp.lock.Lock()
	defer cp.lock.Unlock()

	destroyConnEntry := func(key, ce interface{}) bool {
		if ce.(*connEntry).users != 0 {
			// should never happen
			// TODO: return error, or panic()?
			return false
		}

		ce.(*connEntry).destroy()
		cp.conns.Delete(key.(string))

		return true
	}

	cp.conns.Range(destroyConnEntry)
}

// Return a rados.Conn for the given arguments. Creates a new rados.Conn in
// case there is none in the pool.
func (cp *ConnPool) Get(pool, monitors, keyfile string) (*rados.Conn, string, error) {
	unique := fmt.Sprintf("%s|%s|%s", pool, monitors, keyfile)

	// need a lock while calling ce.touch()
	cp.lock.RLock()
	ce, exists := cp.conns.Load(unique)
	if exists {
		ce.(*connEntry).get()
		cp.lock.RUnlock()
		return ce.(*connEntry).conn, unique, nil
	}
	cp.lock.RUnlock()

	// construct and connect a new rados.Conn
	args := []string{"--pool", pool, "-m", monitors, "--keyfile=" + keyfile}
	conn, err := rados.NewConn()
	if err != nil {
		return nil, "", err
	}
	err = conn.ParseCmdLineArgs(args)
	if err != nil {
		return nil, "", err
	}

	err = conn.Connect()
	if err != nil {
		return nil, "", err
	}
	// connection gets automatically shutdown when it goes out of scope
	// defer conn.Shutdown()

	ce = &connEntry{
		conn:     conn,
		lastUsed: time.Now(),
		unique:   unique,
		users:    1,
	}

	cp.lock.Lock()
	defer cp.lock.Unlock()
	oldCe, loaded := cp.conns.LoadOrStore(unique, ce)
	if loaded {
		// there was a race, oldCe already exists
		ce.(*connEntry).destroy()
		// TODO: increase refcount on oldCe
		return oldCe.(*connEntry).conn, unique, nil
	}

	return conn, unique, nil
}

func (cp *ConnPool) Put(unique string) {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	ce, ok := cp.conns.Load(unique)
	if !ok {
		return
	}

	if ce.(*connEntry).put() {
		cp.conns.Delete(unique)
	}
}

// Add a reference to the connEntry.
// /!\ Only call this while holding the ConnPool.lock.
func (ce *connEntry) get() {
	ce.lastUsed = time.Now()
	ce.users++
}

// Reduce number of references. If this returns true, there are no more users.
// /!\ Only call this while holding the ConnPool.lock.
func (ce *connEntry) put() bool {
	ce.users--
	return ce.users == 0
	// do not call ce.destroy(), let ConnPool.gc() do that
}

func (ce *connEntry) destroy() {
	if ce.conn != nil {
		ce.conn.Shutdown()
		ce.conn = nil
	}
}
