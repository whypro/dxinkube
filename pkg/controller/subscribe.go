package controller

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/golang/glog"
)

type WatchChange struct {
	Ev       zk.Event
	Children []string
}

type watchSub struct {
	conn    *zk.Conn
	path    string
	updates chan WatchChange
}

func ChildrenWSubscribe(conn *zk.Conn, path string) chan WatchChange {
	s := &watchSub{
		conn:    conn,
		path:    path,
		updates: make(chan WatchChange),
	}
	go s.loop()
	return s.updates
}

func (s *watchSub) loop() {
	children, _, refresh, err := s.conn.ChildrenW(s.path)
	updates := s.updates
	ev := zk.Event{Type: zk.EventNodeChildrenChanged, State: zk.StateConnected, Path: s.path, Err: nil}
	update := WatchChange{ev, children}
	for {
		select {
		case <-refresh:
			glog.Infof("ChildrenW %s", s.path)
			children, _, refresh, err = s.conn.ChildrenW(s.path)
			if err == zk.ErrConnectionClosed {
				glog.Errorf("ErrConnectionClosed %s", s.path)
				close(s.updates)
				return
			}
			updates = s.updates
			ev.Err = err
			update = WatchChange{ev, children}

		case updates <- update:
			updates = nil
		}
	}
}
