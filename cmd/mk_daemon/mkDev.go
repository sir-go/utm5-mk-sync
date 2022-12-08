package main

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/routeros.v2"
)

type MkDevice struct {
	Cfg         MkConnParams
	Conn        *routeros.Client
	QueuesUsage float32
}

func (mkd *MkDevice) Connect() (err error) {
	LOG.Debugf("connect to Mk %s", mkd.Cfg.Addr)
	mkd.Conn, err = routeros.Dial(mkd.Cfg.Addr, mkd.Cfg.Username, mkd.Cfg.Password)
	return
}

func (mkd *MkDevice) Disconnect() {
	LOG.Debugf("disconnect from firewall Mk %s", mkd.Cfg.Addr)
	mkd.Conn.Close()
}

func (mkd *MkDevice) Reconnect() error {
	mkd.Disconnect()
	return mkd.Connect()
}

func (mkd *MkDevice) reTry(fn func() (*routeros.Reply, error)) (res *routeros.Reply, err error) {
	res, err = fn()
	if err != nil && strings.Contains(err.Error(), "broken pipe") {
		LOG.Debug(err)
		if err = mkd.Reconnect(); err != nil {
			return
		}
		return fn()
	}
	return
}

func (mkd *MkDevice) killPPPoE(ip string) error {
	LOG.Debugf("try to kill pppoe with ip '%s' on MK %s", ip, mkd.Cfg.Addr)

	reply, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/ppp/active/print", "?address="+ip,
			"=.proplist=.id")
	})
	if err != nil {
		return err
	}

	if len(reply.Re) == 0 {
		return errors.New("not found in active pppoe")
	}

	for _, pairMap := range reply.Re {
		if _, err := mkd.Conn.Run(
			"/ppp/active/remove", "=numbers="+pairMap.Map[".id"]); err != nil {
			return err
		}
	}

	return nil
}

func (mkd *MkDevice) AclGetAll(listName string, result *[]*ARec) error {

	reply, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/ip/firewall/address-list/print", "?list="+listName,
			"=.proplist=.id,address,comment")
	})
	if err != nil {
		return err
	}

	for _, pairMap := range reply.Re {
		arec := &ARec{
			Id:       pairMap.Map[".id"],
			ListName: listName,
			Address:  pairMap.Map["address"],
			Comment:  pairMap.Map["comment"],
		}
		//arec.CalcSum()
		*result = append(*result, arec)
	}

	return nil
}

func (mkd *MkDevice) AclAdd(alrec *ARec) error {
	LOG.Debugf("add %+v", alrec)

	reply, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/ip/firewall/address-list/add",
			"=address="+alrec.Address,
			"=list="+alrec.ListName,
			"=comment="+alrec.Comment)
	})

	if err != nil {
		if strings.Contains(err.Error(), "already have such entry") {
			LOG.Warningf("err: %s, %+v", err.Error(), alrec)
			return nil
		} else {
			LOG.Errorf("err: %s, %+v", err.Error(), alrec)
			return err
		}
	}

	if len(reply.Done.List) > 0 {
		alrec.Id = reply.Done.List[0].Value
	}
	return nil
}

func (mkd *MkDevice) AclRemove(alrec *ARec) error {
	LOG.Debugf("del %+v", alrec)

	_, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/ip/firewall/address-list/remove", "=.id="+alrec.Id)
	})

	if err != nil {
		if strings.Contains(err.Error(), "no such item") {
			LOG.Warningf("no address list entries found with id '%s'\n", alrec.Id)
		} else {
			LOG.Errorf("addrlist remove entry with id '%s' ERROR: %s\n", alrec.Id, err.Error())
			return err
		}
	}
	return nil
}

func (mkd *MkDevice) AclChange(alrec *ARec, fieldName string, newValue string) error {
	_, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/ip/firewall/address-list/set",
			"=.id="+alrec.Id,
			"="+fieldName+"="+newValue)
	})

	if err != nil {
		LOG.Error(err)
		return err
	}
	return nil
}

func (mkd *MkDevice) QueueGetAll() (res []*QRec) {
	reply, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/queue/simple/print", "=.proplist=name,target,max-limit,comment")
	})
	eh(err)

	mkd.QueuesUsage = 0

	for _, resPair := range reply.Re {
		speed := parseSpeedFromQueueLimit(resPair.Map["max-limit"])

		record := QRec{
			Name:    resPair.Map["name"],
			Target:  parseIps(resPair.Map["target"]),
			Speed:   speed,
			Comment: resPair.Map["comment"],
			Dev:     mkd,
		}
		record.CalcSum()

		res = append(res, &record)
		mkd.QueuesUsage += float32(speed) / mkd.Cfg.Coef
	}
	return
}

func (mkd *MkDevice) QueueAdd(qrec *QRec) error {
	maxLimit := fmt.Sprintf("%dM/%dM", qrec.Speed, qrec.Speed)
	ips := strings.Join(qrec.Target, ",")
	LOG.Debugf("+q [dev: %s] %s [%s] %d Mbps %s",
		qrec.Dev.Cfg.Addr, qrec.Name, ips, qrec.Speed, qrec.Comment)

	_, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		params := []string{
			"/queue/simple/add",
			"=name=" + qrec.Name,
			"=target=" + ips,
			"=limit-at=" + maxLimit,
			"=max-limit=" + maxLimit,
			"=comment=" + qrec.Comment,
		}
		if qrec.Comment == "PG" {
			params = append(params, "=time=7h-0s,sun,mon,tue,wed,thu,fri,sat")
		}
		return mkd.Conn.RunArgs(params)
	})
	if err != nil {
		if strings.Contains(err.Error(), "already have such name") {
			LOG.Warningf("queue (name = %s) already exists", qrec.Name)
		} else {
			LOG.Errorf("queue (name = %s) add ERROR: %s", qrec.Name, err.Error())
			return err
		}
	}
	mkd.QueuesUsage += float32(qrec.Speed) / mkd.Cfg.Coef
	return nil
}

func (mkd *MkDevice) QueueRemove(qrec *QRec) error {
	LOG.Debugf("-q [dev: %s] %s [%s] %d Mbps %s",
		qrec.Dev.Cfg.Addr, qrec.Name, strings.Join(qrec.Target, ","), qrec.Speed, qrec.Comment)

	_, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/queue/simple/remove", "=numbers="+qrec.Name)
	})

	if err != nil {
		if strings.Contains(err.Error(), "no such item") {
			LOG.Warningf("queue (name = %s) not found\n", qrec.Name)
		} else {
			LOG.Errorf("queue (name = %s) remove ERROR: %s\n", qrec.Name, err.Error())
			return err
		}
	}
	mkd.QueuesUsage -= float32(qrec.Speed) / mkd.Cfg.Coef
	return nil
}

func (mkd *MkDevice) QueueSetTarget(rec *QRec, target string) error {
	LOG.Debugf("set target queue %s [%s] -> %s", rec.Name, strings.Join(rec.Target, ","), target)

	_, err := mkd.reTry(func() (reply *routeros.Reply, err error) {
		return mkd.Conn.Run("/queue/simple/set",
			"=numbers="+rec.Name,
			"=target="+target)
	})

	if err != nil {
		if strings.Contains(err.Error(), "no such item") {
			LOG.Warningf("queue (name = %s) not found\n", rec.Name)
		} else {
			LOG.Errorf("queue (name = %s) set target ERROR: %s\n", rec.Name, err.Error())
			return err
		}
	}
	return nil
}
